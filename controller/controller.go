// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"context"
	"fmt"
	"net/http"

	ametrics "github.com/ava-labs/avalanchego/api/metrics"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/snow"
	handle "github.com/sausaging/hyper-pvzk/accept_handlers"
	"github.com/sausaging/hyper-pvzk/actions"
	"github.com/sausaging/hyper-pvzk/auth"
	"github.com/sausaging/hyper-pvzk/config"
	"github.com/sausaging/hyper-pvzk/consts"
	"github.com/sausaging/hyper-pvzk/genesis"
	"github.com/sausaging/hyper-pvzk/rpc"
	"github.com/sausaging/hyper-pvzk/storage"
	"github.com/sausaging/hyper-pvzk/trustless"
	"github.com/sausaging/hyper-pvzk/version"
	"github.com/sausaging/hypersdk/builder"
	"github.com/sausaging/hypersdk/chain"
	"github.com/sausaging/hypersdk/fees"
	"github.com/sausaging/hypersdk/filedb"
	"github.com/sausaging/hypersdk/gossiper"
	hrpc "github.com/sausaging/hypersdk/rpc"
	hstorage "github.com/sausaging/hypersdk/storage"
	"github.com/sausaging/hypersdk/vm"
	"go.uber.org/zap"
)

var _ vm.Controller = (*Controller)(nil)
var _ chain.Rules = (&genesis.Rules{})

type Controller struct {
	inner *vm.VM

	snowCtx      *snow.Context
	genesis      *genesis.Genesis
	config       *config.Config
	stateManager *storage.StateManager

	metrics *metrics

	metaDB database.Database
	fileDB *filedb.FileDB

	trustless *trustless.Trustless
}

func New() *vm.VM {
	return vm.New(&Controller{}, version.Version)
}

func (c *Controller) Initialize(
	inner *vm.VM,
	snowCtx *snow.Context,
	gatherer ametrics.MultiGatherer,
	genesisBytes []byte,
	upgradeBytes []byte, // subnets to allow for AWM
	configBytes []byte,
) (
	vm.Config,
	vm.Genesis,
	builder.Builder,
	gossiper.Gossiper,
	database.Database,
	database.Database,
	*filedb.FileDB,
	vm.Handlers,
	chain.ActionRegistry,
	chain.AuthRegistry,
	map[uint8]vm.AuthEngine,
	error,
) {
	c.inner = inner
	c.snowCtx = snowCtx
	c.stateManager = &storage.StateManager{}
	// Instantiate metrics
	var err error
	c.metrics, err = newMetrics(gatherer)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	// Load config and genesis
	c.config, err = config.New(c.snowCtx.NodeID, configBytes)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	c.snowCtx.Log.SetLevel(c.config.GetLogLevel())
	snowCtx.Log.Info("initialized config", zap.Bool("loaded", c.config.Loaded()), zap.Any("contents", c.config))

	c.genesis, err = genesis.New(genesisBytes, upgradeBytes)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf(
			"unable to read genesis: %w",
			err,
		)
	}
	snowCtx.Log.Info("loaded genesis", zap.Any("genesis", c.genesis))

	// Create DBs
	blockDB, fileDB, stateDB, metaDB, err := hstorage.New(snowCtx.ChainDataDir, gatherer)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	c.metaDB = metaDB
	c.fileDB = fileDB

	c.trustless = trustless.New(c.config.Port, c.config.ListenerPort, &snowCtx.WarpSigner, snowCtx.PublicKey, c.config.ValPrivKey, c.snowCtx.Log, c.UnitPrices, c.Submit, c.Rules)

	go c.trustless.ListenResults()
	// Create handlers
	//
	// hypersdk handler are initiatlized automatically, you just need to
	// initialize custom handlers here.
	apis := map[string]http.Handler{}
	jsonRPCHandler, err := hrpc.NewJSONRPCHandler(
		consts.Name,
		rpc.NewJSONRPCServer(c),
	)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	apis[rpc.JSONRPCEndpoint] = jsonRPCHandler

	// Create builder and gossiper
	var (
		build  builder.Builder
		gossip gossiper.Gossiper
	)
	if c.config.TestMode {
		c.inner.Logger().Info("running build and gossip in test mode")
		build = builder.NewManual(inner)
		gossip = gossiper.NewManual(inner)
	} else {
		build = builder.NewTime(inner)
		gcfg := gossiper.DefaultProposerConfig()
		gossip, err = gossiper.NewProposer(inner, gcfg)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
	}
	return c.config, c.genesis, build, gossip, blockDB, stateDB, fileDB, apis, consts.ActionRegistry, consts.AuthRegistry, auth.Engines(), nil
}

func (c *Controller) Rules(t int64) chain.Rules {
	// TODO: extend with [UpgradeBytes]
	return c.genesis.Rules(t, c.snowCtx.NetworkID, c.snowCtx.ChainID, c.config.Client, c.inner.CurrentValidators)
}

func (c *Controller) StateManager() chain.StateManager {
	return c.stateManager
}

func (c *Controller) Accepted(ctx context.Context, blk *chain.StatelessBlock) error {
	batch := c.metaDB.NewBatch()
	defer batch.Reset()

	results := blk.Results()
	for i, tx := range blk.Txs {
		result := results[i]

		switch tx.Action.(type) {

		case *actions.SP1:
			sp1 := tx.Action.(*actions.SP1)
			c.trustless.ListenActions(tx.ID(), sp1.TimeOutBlocks)
			if err := handle.HandleSP1(tx.ID(), sp1.ImageID, uint16(sp1.ProofValType), c.fileDB.BaseDir(), c.config.Client); err != nil {
				c.inner.Logger().Info("error handling SP1", zap.Error(err))
			}

		case *actions.RiscZero:
			risc0 := tx.Action.(*actions.RiscZero)
			c.trustless.ListenActions(tx.ID(), risc0.TimeOutBlocks)
			if err := handle.HandleRiscZero(tx.ID(), risc0.ImageID, uint16(risc0.ProofValType), risc0.RiscZeroImageID, c.fileDB.BaseDir(), c.config.Client); err != nil {
				c.inner.Logger().Info("error handling RiscZero", zap.Error(err))
			}
		case *actions.Miden:
			miden := tx.Action.(*actions.Miden)
			c.trustless.ListenActions(tx.ID(), miden.TimeOutBlocks)
			if err := handle.HandleMiden(tx.ID(), miden.ImageID, uint16(miden.ProofValType), miden.CodeFrontEnd, miden.InputsFrontEnd, miden.OutputsFrontEnd, c.fileDB.BaseDir(), c.config.Client); err != nil {
				c.inner.Logger().Info("error handling Miden", zap.Error(err))
			}
			// case *actions.ValResultVote:
			// 	//@todo keep track of spendings of validators
		}
		if c.config.GetStoreTransactions() {
			err := storage.StoreTransaction(
				ctx,
				batch,
				tx.ID(),
				blk.GetTimestamp(),
				result.Success,
				result.Consumed,
				result.Fee,
			)
			if err != nil {
				return err
			}
		}
		if result.Success {
			switch tx.Action.(type) { //nolint:gocritic
			case *actions.Transfer:
				c.metrics.transfer.Inc()
			}
		}
	}
	return batch.Write()
}

func (*Controller) Rejected(context.Context, *chain.StatelessBlock) error {
	return nil
}

func (*Controller) Shutdown(context.Context) error {
	// Do not close any databases provided during initialization. The VM will
	// close any databases your provided.
	return nil
}

func (c *Controller) GetFileDB() *filedb.FileDB {
	return c.fileDB
}

func (c *Controller) UnitPrices() (fees.Dimensions, error) {
	return c.inner.UnitPrices(context.Background())
}

func (c *Controller) Submit(ctx context.Context, verifyAuth bool, txs []*chain.Transaction) []error {
	return c.inner.Submit(ctx, verifyAuth, txs)
}
