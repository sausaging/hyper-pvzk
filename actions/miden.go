package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/state"
	"github.com/ava-labs/hypersdk/utils"
	mconsts "github.com/sausaging/hyper-pvzk/consts"
	"github.com/sausaging/hyper-pvzk/requester"
	"github.com/sausaging/hyper-pvzk/storage"
)

var _ chain.Action = (*Miden)(nil)

type Miden struct {
	ImageID      ids.ID `json:"imageID"`
	ProofValType uint64 `json:"proofValType"`
	//@todo future support: large code, inputs, outputs support
	CodeFrontEnd    string `json:"codeFrontEnd"`
	InputsFrontEnd  string `json:"inputsFrontEnd"`
	OutputsFrontEnd string `json:"outputsFrontEnd"`
}

type MidenRequestArgs struct {
	CodeFrontEnd    string `json:"codeFrontEnd"`
	InputsFrontEnd  string `json:"inputsFrontEnd"`
	OutputsFrontEnd string `json:"outputsFrontEnd"`
	ProofFilePath   string `json:"proofFilePath"`
}

type MidenReplyArgs struct {
	//@todo intro security field
	IsValid bool `json:"isValid"`
}

func (*Miden) GetTypeID() uint8 {
	return mconsts.MidenID
}

func (m *Miden) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.DeployKey(m.ImageID, m.ProofValType)): state.Read | state.Write,
	}
}

func (*Miden) StateKeysMaxChunks() []uint16 {
	return []uint16{consts.MaxUint16}
}

func (*Miden) OutputsWarpMessage() bool {
	return false
}

func (*Miden) MaxComputeUnits(chain.Rules) uint64 {
	return MidenComputeUnits
}

func (m *Miden) Size() int {
	return consts.IDLen + consts.Uint64Len + len(m.CodeFrontEnd) + len(m.InputsFrontEnd) + len(m.OutputsFrontEnd)
}

func (m *Miden) Marshal(p *codec.Packer) {
	p.PackID(m.ImageID)
	p.PackUint64(m.ProofValType)
	p.PackString(m.CodeFrontEnd)
	p.PackString(m.InputsFrontEnd)
	p.PackString(m.OutputsFrontEnd)
}

func UnmarshalMiden(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var miden Miden
	p.UnpackID(true, &miden.ImageID)
	miden.ProofValType = p.UnpackUint64(true)
	miden.CodeFrontEnd = p.UnpackString(true)
	miden.InputsFrontEnd = p.UnpackString(true)
	miden.OutputsFrontEnd = p.UnpackString(true)
	return &miden, nil
}

func (*Miden) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (m *Miden) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {

	imageID := m.ImageID
	proofValType := m.ProofValType
	proofJsonBytes, err := storage.GetDeployType(ctx, mu, imageID, proofValType)
	if err != nil {
		return false, 1000, utils.ErrBytes(fmt.Errorf("%s: can't get proof at type %d from state", err, proofValType)), nil, nil
	}

	proofFilePath := requester.BASEFILEPATH + imageID.String() + strconv.Itoa(int(proofValType)) + ".json"
	if err := utils.SaveBytes(proofFilePath, proofJsonBytes); err != nil {
		return false, 2000, utils.ErrBytes(fmt.Errorf("%s: can't write proof to disk", err)), nil, nil
	}

	cli, uri := requester.GetRequesterInstance(rules)
	endPointUri := uri + requester.MIDENENDPOINT
	midenArgs := MidenRequestArgs{
		CodeFrontEnd:    m.CodeFrontEnd,
		InputsFrontEnd:  m.InputsFrontEnd,
		OutputsFrontEnd: m.OutputsFrontEnd,
		ProofFilePath:   proofFilePath,
	}

	jsonData, err := json.Marshal(midenArgs)
	if err != nil {
		return false, 3000, utils.ErrBytes(fmt.Errorf("%s: can't marshal json", err)), nil, nil
	}

	req, err := requester.NewRequest(endPointUri, jsonData)
	if err != nil {
		return false, 4000, utils.ErrBytes(fmt.Errorf("%s: can't request http", err)), nil, nil
	}

	resp, err := cli.Do(req)
	if err != nil {
		return false, 5000, utils.ErrBytes(fmt.Errorf("%s: can't do request", err)), nil, nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, 6000, utils.ErrBytes(fmt.Errorf("%s: can't decode client response", err)), nil, nil
	}
	reply := new(MidenReplyArgs)
	err = json.Unmarshal(body, &reply)
	if err != nil {
		return false, 6000, utils.ErrBytes(fmt.Errorf("%s: can't unmarshal json", err)), nil, nil
	}

	return reply.IsValid, 6000, nil, nil, nil
}
