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

var _ chain.Action = (*SP1)(nil)

type SP1 struct {
	ImageID       ids.ID `json:"image_id"`
	ProofValType  uint64 `json:"proof_val_type"`
	TimeOutBlocks uint64 `json:"time_out_blocks"`
}

func (*SP1) GetTypeID() uint8 {
	return mconsts.SP1ID
}

func (s *SP1) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{string(storage.TimeOutKey(txID)): state.All}
}

func (*SP1) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.TimeOutChunks}
}

func (*SP1) OutputsWarpMessage() bool {
	return false
}

func (*SP1) MaxComputeUnits(chain.Rules) uint64 {
	return SP1ComputeUnits
}

func (s SP1) Size() int {
	return consts.IDLen + consts.Uint64Len*2
}

func (s *SP1) Marshal(p *codec.Packer) {
	p.PackID(s.ImageID)
	p.PackUint64(s.ProofValType)
	p.PackUint64(s.TimeOutBlocks)
}

func UnmarshalSP1(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var sp1 SP1
	p.UnpackID(true, &sp1.ImageID)
	sp1.ProofValType = p.UnpackUint64(true)
	sp1.TimeOutBlocks = p.UnpackUint64(true)
	return &sp1, nil
}

func (*SP1) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (s *SP1) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	ts int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	if err := storage.StoreTimeOut(ctx, mu, txID, s.TimeOutBlocks, ts); err != nil {
		return false, 4000, nil, nil, fmt.Errorf("%w: unable to store time out", err)
	}
	return true, 8000, nil, nil, nil
}
