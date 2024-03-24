package actions

import (
	"context"
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/state"
	mconsts "github.com/sausaging/hyper-pvzk/consts"
	"github.com/sausaging/hyper-pvzk/storage"
)

var _ chain.Action = (*RiscZero)(nil)

type RiscZero struct {
	ImageID         ids.ID `json:"image_id"`
	ProofValType    uint64 `json:"proof_val_type"`
	RiscZeroImageID string `json:"risc_zero_image_id"`
	TimeOutBlocks   uint64 `json:"time_out_blocks"`
}

type RiscZeroArgs struct {
	RiscZeroImageID string `json:"risc_zero_image_id"`
	ProofFilePath   string `json:"proof_file_path"`
}

type RiscZeroReplyArgs struct {
	IsSubmitted bool `json:"is_submitted"`
}

func (*RiscZero) GetTypeID() uint8 {
	return mconsts.RiscZeroID
}

func (r *RiscZero) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{string(storage.TimeOutKey(txID)): state.All}
}

func (*RiscZero) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.TimeOutChunks}
}

func (*RiscZero) OutputsWarpMessage() bool {
	return false
}

func (*RiscZero) MaxComputeUnits(chain.Rules) uint64 {
	return RiscZeroComputeUnits
}

func (r *RiscZero) Size() int {
	return consts.IDLen + consts.Uint64Len*2 + len(r.RiscZeroImageID)
}

func (r *RiscZero) Marshal(p *codec.Packer) {
	p.PackID(r.ImageID)
	p.PackUint64(r.ProofValType)
	p.PackString(r.RiscZeroImageID)
	p.PackUint64(r.TimeOutBlocks)
}

func UnmarshalRiscZero(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var riscZero RiscZero
	p.UnpackID(true, &riscZero.ImageID)
	riscZero.ProofValType = p.UnpackUint64(true)
	riscZero.RiscZeroImageID = p.UnpackString(true)
	riscZero.TimeOutBlocks = p.UnpackUint64(true)
	return &riscZero, nil
}

func (*RiscZero) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (r *RiscZero) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	ts int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	if err := storage.StoreTimeOut(ctx, mu, txID, r.TimeOutBlocks, ts); err != nil {
		return false, 4000, nil, nil, fmt.Errorf("%w: unable to store time out", err)
	}
	return true, 6000, nil, nil, nil
}
