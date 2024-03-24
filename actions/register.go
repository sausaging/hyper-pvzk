package actions

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/state"
	mconsts "github.com/sausaging/hyper-pvzk/consts"
)

var _ chain.Action = (*Register)(nil)

type Register struct {
	ProovingSystem uint64 `json:"prooving_system"`
}

func (*Register) GetTypeID() uint8 {
	return mconsts.RegisterID
}

func (*Register) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{}
}

func (*Register) StateKeysMaxChunks() []uint16 {
	return []uint16{}
}

func (*Register) OutputsWarpMessage() bool {
	return false
}

func (*Register) MaxComputeUnits(chain.Rules) uint64 {
	return RegisterComputeUnits
}

func (*Register) Size() int {
	return consts.Uint64Len
}

func (r *Register) Marshal(p *codec.Packer) {
	p.PackUint64(r.ProovingSystem)
}

func UnmarshalRegister(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var register Register
	register.ProovingSystem = p.UnpackUint64(true)
	return &register, nil
}

func (*Register) ValidRange(chain.Rules) (int64, int64) {
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
	// @todo should we store what type of proof system is meant to be used by the imageID(i.e. txID) generated??
	return true, RegisterComputeUnits, nil, nil, nil
}
