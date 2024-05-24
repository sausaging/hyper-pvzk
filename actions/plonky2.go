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

var _ chain.Action = (*PLONKY2)(nil)

type PLONKY2 struct {
	ImageID             ids.ID `json:"image_id"`
	ProofValType        uint64 `json:"proof_val_type"`
	CommonDataValType   uint64 `json:"common_data_val_type"`
	VerifierDataValType uint64 `json:"verifier_data_val_type"`
	TimeOutBlocks       uint64 `json:"time_out_blocks"`
}

func (*PLONKY2) GetTypeID() uint8 {
	return mconsts.Plonky2ID
}

func (s *PLONKY2) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{string(storage.TimeOutKey(txID)): state.All}
}

func (*PLONKY2) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.TimeOutChunks}
}

func (*PLONKY2) OutputsWarpMessage() bool {
	return false
}

func (*PLONKY2) MaxComputeUnits(chain.Rules) uint64 {
	return PLONKY2ComputeUnits
}

func (s PLONKY2) Size() int {
	return consts.IDLen + consts.Uint64Len*2
}

func (s *PLONKY2) Marshal(p *codec.Packer) {
	p.PackID(s.ImageID)
	p.PackUint64(s.ProofValType)
	p.PackUint64(s.CommonDataValType)
	p.PackUint64(s.VerifierDataValType)
	p.PackUint64(s.TimeOutBlocks)
}

func UnmarshalPLONKY2(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var PLONKY2 PLONKY2
	p.UnpackID(true, &PLONKY2.ImageID)
	PLONKY2.ProofValType = p.UnpackUint64(true)
	PLONKY2.CommonDataValType = p.UnpackUint64(true)
	PLONKY2.VerifierDataValType = p.UnpackUint64(true)
	PLONKY2.TimeOutBlocks = p.UnpackUint64(true)
	return &PLONKY2, nil
}

func (*PLONKY2) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (s *PLONKY2) Execute(
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
