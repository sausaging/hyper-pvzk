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

var _ chain.Action = (*Deploy)(nil)

type Deploy struct {
	ImageID      ids.ID `json:"imageId"`
	ProofvalType uint64 `json:"proofValType"` // type will be 1 for ELF and 1 + for proofs
	Data         []byte `json:"data"`
}

func (*Deploy) GetTypeID() uint8 {
	return mconsts.DeployID
}

func (d *Deploy) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.DeployKey(d.ImageID, d.ProofvalType)): state.Read | state.Write,
	}
}

func (*Deploy) StateKeysMaxChunks() []uint16 {
	return []uint16{consts.MaxUint16}
}

func (*Deploy) OutputsWarpMessage() bool {
	return false
}

func (*Deploy) MaxComputeUnits(chain.Rules) uint64 {
	return DeployComputeUnits
}

func (d Deploy) Size() int {
	return consts.IDLen + consts.Uint64Len + len(d.Data)
}

func (d *Deploy) Marshal(p *codec.Packer) {
	p.PackID(d.ImageID)
	p.PackUint64(d.ProofvalType)
	p.PackBytes(d.Data)
}

func UnmarshalDeploy(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var deploy Deploy
	p.UnpackID(true, &deploy.ImageID)
	deploy.ProofvalType = p.UnpackUint64(true)
	p.UnpackBytes(consts.MaxInt, true, &deploy.Data)
	return &deploy, nil
}

func (*Deploy) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (d *Deploy) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	if err := storage.StoreDeployType(ctx, mu, d.ImageID, d.ProofvalType, d.Data); err != nil {
		return false, 1_000, utils.ErrBytes(fmt.Errorf("%s: deployemnt error", err)), nil, nil
	}
	return true, 10_000, nil, nil, nil
}
