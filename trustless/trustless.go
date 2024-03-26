package trustless

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/crypto/bls"
	"github.com/ava-labs/hypersdk/crypto/ed25519"
	"github.com/ava-labs/hypersdk/fees"
	"github.com/ava-labs/hypersdk/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/sausaging/hyper-pvzk/actions"
	"github.com/sausaging/hyper-pvzk/auth"
	"github.com/sausaging/hyper-pvzk/consts"
	"go.uber.org/zap"
)

// trustless is the module for rust-server to submit results of verification
type Trustless struct {
	port         string
	listenerPort string
	// rules        chain.Rules
	warpSigner *warp.Signer
	publicKey  *bls.PublicKey
	logger     logging.Logger

	l sync.Mutex

	queuedActions   map[ids.ID]uint64
	verifiedActions map[ids.ID]bool

	unitPrices func() (fees.Dimensions, error)
	submit     func(context.Context, bool, []*chain.Transaction) []error
	rules      func(int64) chain.Rules

	authFactory chain.AuthFactory
}

type SubmitResultArgs struct {
	TxID    string `json:"tx_id"`
	IsValid bool   `json:"is_valid"`
}

var _ chain.Parser = (*Parser)(nil)

type Parser struct {
	t *Trustless
}

func (p *Parser) Rules(t int64) chain.Rules {
	return p.t.rules(t)
}

func (p *Parser) Registry() (chain.ActionRegistry, chain.AuthRegistry) {
	return consts.ActionRegistry, consts.AuthRegistry
}

// can be a bit loose as the communiation happens between the two trusted/integrated parties
func New(port string, listenerPort string, warpSigner *warp.Signer, publicKey *bls.PublicKey, valPrivKey string, logger logging.Logger, unitPrices func() (fees.Dimensions, error), submit func(context.Context, bool, []*chain.Transaction) []error, rules func(int64) chain.Rules) *Trustless {
	authFactory := auth.NewED25519Factory(ed25519.PrivateKey(common.Hex2Bytes(valPrivKey)))
	return &Trustless{
		port:            port,
		listenerPort:    listenerPort,
		warpSigner:      warpSigner,
		publicKey:       publicKey,
		logger:          logger,
		queuedActions:   make(map[ids.ID]uint64),
		verifiedActions: make(map[ids.ID]bool),
		unitPrices:      unitPrices,
		submit:          submit,
		rules:           rules,
		authFactory:     authFactory,
	}
}

func (t *Trustless) Parser() chain.Parser {
	return &Parser{t: t}
}

func (t *Trustless) ListenActions(txID ids.ID, timeOut uint64 /*any further data? like verify type*/) error {
	t.l.Lock()
	t.queuedActions[txID] = timeOut
	t.l.Unlock()
	return nil
}

func (t *Trustless) ListenResults() {
	r := mux.NewRouter()

	r.HandleFunc("/ping", t.ping).Methods("GET")
	r.HandleFunc("/submit-result", t.submitResult).Methods("POST")
	srv := &http.Server{
		Addr:    ":" + t.listenerPort,
		Handler: r,
	}
	srv.ListenAndServe()
}

func (*Trustless) ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Pong\n")
}

func (t *Trustless) submitResult(w http.ResponseWriter, r *http.Request) {
	var req SubmitResultArgs
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := ids.FromString(req.TxID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	t.l.Lock()
	t.verifiedActions[id] = req.IsValid
	t.l.Unlock()
	// @todo should we return any response??
	msg := actions.GetMessage(id, req.IsValid)
	unSigMsg, err := warp.NewUnsignedMessage(t.rules(0).NetworkID(), t.rules(0).ChainID(), msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	sig, err := (*t.warpSigner).Sign(unSigMsg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	// signature should be valid -> any one can submit it. we check for the public key
	action := actions.ValidatorVote{
		TxID:      id,
		Vote:      req.IsValid,
		Signature: sig,
		PublicKey: bls.PublicKeyToBytes(t.publicKey),
	}
	parser := t.Parser()

	txID, err := t.GenerateTransaction(context.Background(), parser, &action, t.authFactory)
	if err != nil {
		t.logger.Error("error in submitting tx", zap.Error(err))
	}
	fmt.Fprintf(w, "Result submitted. %s\n", txID.String())
}

func (t *Trustless) GenerateTransaction(
	ctx context.Context,
	parser chain.Parser,
	action chain.Action,
	authFactory chain.AuthFactory,
) (ids.ID, error) {
	unitPrices, err := t.unitPrices()
	if err != nil {
		return ids.Empty, fmt.Errorf("%s: error in fetching unit prices", err)
	}
	maxUnits, err := chain.EstimateMaxUnits(parser.Rules(time.Now().UnixMilli()), action, authFactory, nil)
	if err != nil {
		return ids.Empty, fmt.Errorf("%s: error in estimating max units", err)
	}
	maxFee, err := fees.MulSum(unitPrices, maxUnits)
	if err != nil {
		return ids.Empty, fmt.Errorf("%s: error in calculating max fee", err)
	}
	now := time.Now().UnixMilli()
	rules := parser.Rules(now)
	base := &chain.Base{
		Timestamp: utils.UnixRMilli(now, rules.GetValidityWindow()),
		ChainID:   rules.ChainID(),
		MaxFee:    maxFee,
	}
	actionRegistry, authRegistry := parser.Registry()
	tx := chain.NewTx(base, nil, action)
	tx, err = tx.Sign(authFactory, actionRegistry, authRegistry)
	if err != nil {
		return ids.Empty, fmt.Errorf("%w: failed to sign transaction", err)
	}
	errs := t.submit(ctx, false, []*chain.Transaction{tx})
	return tx.ID(), errs[0]
}
