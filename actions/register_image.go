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
	"github.com/ava-labs/hypersdk/utils"
	mconsts "github.com/sausaging/hyper-pvzk/consts"
	"github.com/sausaging/hyper-pvzk/storage"
)

var _ chain.Action = (*Register)(nil)

type Register struct {
	MaxChunks uint64 `json:"max_chunks"` //@todo feature implementation of max_chunks has not yet been realised
}

func (*Register) GetTypeID() uint8 {
	return mconsts.RegisterID
}

func (r *Register) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.ChunkKey(txID, uint16(r.MaxChunks))): state.Read | state.Write,
	}
}

func (*Register) StateKeysMaxChunks() []uint16 {
	return []uint16{consts.Uint16Len}
}

func (*Register) OutputsWarpMessage() bool {
	return false
}

func (*Register) MaxComputeUnits(chain.Rules) uint64 {
	return RegisterComputeUnits + 1000
}

func (*Register) Size() int {
	return consts.Uint64Len
}

func (r *Register) Marshal(p *codec.Packer) {
	p.PackUint64(r.MaxChunks)
}

func (*Register) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (r *Register) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	if err := storage.StoreRegistration(ctx, mu, actor, txID, uint16(r.MaxChunks)); err != nil {
		return false, RegisterComputeUnits, utils.ErrBytes(fmt.Errorf("%s: can't store registration", err)), nil, nil
	}
	return true, RegisterComputeUnits, nil, nil, nil
}

func UnmarshalRegister(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var registerImage Register
	registerImage.MaxChunks = p.UnpackUint64(true)
	return &registerImage, nil
}
