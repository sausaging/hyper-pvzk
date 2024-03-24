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
	"github.com/sausaging/hyper-pvzk/storage"
)

var _ chain.Action = (*RegisterImage)(nil)

type RegisterImage struct {
	ImageID  ids.ID `json:"image_id"`
	ValType  uint64 `json:"val_type"`  // 1 for elf, 1 + for proofs
	RootHash string `json:"root_hash"` // root hash of the image --> should we use keccak for this?? @todo
}

func (*RegisterImage) GetTypeID() uint8 {
	return mconsts.RegisterImageID
}

func (r *RegisterImage) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.HashKey(r.ImageID, uint16(r.ValType))): state.All,
	}
}

func (*RegisterImage) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.HashChunksMax}
}

func (*RegisterImage) OutputsWarpMessage() bool {
	return false
}

func (*RegisterImage) MaxComputeUnits(chain.Rules) uint64 {
	return RegisterImageComputeUnits
}

func (r *RegisterImage) Size() int {
	return consts.IDLen + consts.Uint64Len + len(r.RootHash)
}

func (r *RegisterImage) Marshal(p *codec.Packer) {
	p.PackID(r.ImageID)
	p.PackUint64(r.ValType)
	p.PackString(r.RootHash)
}

func UnmarshalRegisterImage(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var registerImage RegisterImage
	p.UnpackID(true, &registerImage.ImageID)
	registerImage.ValType = p.UnpackUint64(true)
	registerImage.RootHash = p.UnpackString(true)
	return &registerImage, nil
}

func (*RegisterImage) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (r *RegisterImage) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	imageID := r.ImageID
	valType := uint16(r.ValType)
	rootHash := r.RootHash
	if err := storage.StoreHashKeyType(ctx, mu, imageID, valType, []byte(rootHash)); err != nil {
		return false, 0, nil, nil, err
	}
	return true, RegisterImageComputeUnits, nil, nil, nil
}
