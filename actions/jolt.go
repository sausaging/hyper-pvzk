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

var _ chain.Action = (*Jolt)(nil)

type Jolt struct {
	ImageID       ids.ID `json:"image_id"`
	ProofValType  uint64 `json:"proof_val_type"`
	TimeOutBlocks uint64 `json:"time_out_blocks"`
}

func (*Jolt) GetTypeID() uint8 {
	return mconsts.JoltID
}

func (j *Jolt) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{string(storage.TimeOutKey(txID)): state.All}
}

func (*Jolt) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.TimeOutChunks}
}

func (*Jolt) OutputsWarpMessage() bool {
	return false
}

func (*Jolt) MaxComputeUnits(chain.Rules) uint64 {
	return JoltComputeUnits
}

func (j Jolt) Size() int {
	return consts.IDLen + consts.Uint64Len*2
}

func (j *Jolt) Marshal(p *codec.Packer) {
	p.PackID(j.ImageID)
	p.PackUint64(j.ProofValType)
	p.PackUint64(j.TimeOutBlocks)
}

func UnmarshalJolt(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var jolt Jolt
	p.UnpackID(true, &jolt.ImageID)
	jolt.ProofValType = p.UnpackUint64(true)
	jolt.TimeOutBlocks = p.UnpackUint64(true)
	return &jolt, nil
}

func (*Jolt) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (j *Jolt) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	ts int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	if err := storage.StoreTimeOut(ctx, mu, txID, j.TimeOutBlocks, ts); err != nil {
		return false, 4000, nil, nil, fmt.Errorf("%w: unable to store time out", err)
	}
	return true, 8000, nil, nil, nil
}
