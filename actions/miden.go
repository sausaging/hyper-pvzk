package actions

import (
	"context"
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	mconsts "github.com/sausaging/hyper-pvzk/consts"
	"github.com/sausaging/hyper-pvzk/storage"
	"github.com/sausaging/hypersdk/chain"
	"github.com/sausaging/hypersdk/codec"
	"github.com/sausaging/hypersdk/consts"
	"github.com/sausaging/hypersdk/state"
)

var _ chain.Action = (*Miden)(nil)

type Miden struct {
	ImageID      ids.ID `json:"image_id"`
	ProofValType uint64 `json:"proof_val_type"`
	//@todo future support: large code, inputs, outputs support
	CodeFrontEnd    string `json:"code_front_end"`
	InputsFrontEnd  string `json:"inputs_front_end"`
	OutputsFrontEnd string `json:"outputs_front_end"`
	TimeOutBlocks   uint64 `json:"time_out_blocks"`
}

type MidenRequestArgs struct {
	CodeFrontEnd    string `json:"code_front_end"`
	InputsFrontEnd  string `json:"inputs_front_end"`
	OutputsFrontEnd string `json:"outputs_front_end"`
	ProofFilePath   string `json:"proof_file_path"`
}

type MidenReplyArgs struct {
	//@todo intro security field
	IsValid bool `json:"is_valid"`
}

func (*Miden) GetTypeID() uint8 {
	return mconsts.MidenID
}

func (m *Miden) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{string(storage.TimeOutKey(txID)): state.All}
}

func (*Miden) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.TimeOutChunks}
}

func (*Miden) OutputsWarpMessage() bool {
	return false
}

func (*Miden) MaxComputeUnits(chain.Rules) uint64 {
	return MidenComputeUnits
}

func (m *Miden) Size() int {
	return consts.IDLen + consts.Uint64Len*2 + len(m.CodeFrontEnd) + len(m.InputsFrontEnd) + len(m.OutputsFrontEnd)
}

func (m *Miden) Marshal(p *codec.Packer) {
	p.PackID(m.ImageID)
	p.PackUint64(m.ProofValType)
	p.PackString(m.CodeFrontEnd)
	p.PackString(m.InputsFrontEnd)
	p.PackString(m.OutputsFrontEnd)
	p.PackUint64(m.TimeOutBlocks)
}

func UnmarshalMiden(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var miden Miden
	p.UnpackID(true, &miden.ImageID)
	miden.ProofValType = p.UnpackUint64(true)
	miden.CodeFrontEnd = p.UnpackString(true)
	miden.InputsFrontEnd = p.UnpackString(true)
	miden.OutputsFrontEnd = p.UnpackString(true)
	miden.TimeOutBlocks = p.UnpackUint64(true)
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
	ts int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {

	if err := storage.StoreTimeOut(ctx, mu, txID, m.TimeOutBlocks, ts); err != nil {
		return false, 4000, nil, nil, fmt.Errorf("%w: unable to store time out", err)
	}

	return true, 6000, nil, nil, nil
}
