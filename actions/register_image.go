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
	ImageID    ids.ID `json:"image_id"`
	ValType    uint64 `json:"val_type"`    // 1 for elf, 1 + for proofs
	ChunkSize  uint64 `json:"chunk_size"`  // in bytes
	TotalBytes uint64 `json:"total_bytes"` // total bytes for elf/proofs
}

func (*Register) GetTypeID() uint8 {
	return mconsts.RegisterID
}

func (r *Register) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.ChunkKey(r.ImageID, uint16(r.ValType))):  state.All,
		string(storage.DeployKey(r.ImageID, uint16(r.ValType))): state.All,
	}
}

func (*Register) StateKeysMaxChunks() []uint16 {
	return []uint16{consts.Uint16Len}
}

func (*Register) OutputsWarpMessage() bool {
	return false
}

func (*Register) MaxComputeUnits(chain.Rules) uint64 {
	return RegisterComputeUnits
}

func (*Register) Size() int {
	return consts.IDLen + 3*consts.Uint64Len
}

func (r *Register) Marshal(p *codec.Packer) {
	p.PackID(r.ImageID)
	p.PackUint64(r.ValType)
	p.PackUint64(r.ChunkSize)
	p.PackUint64(r.TotalBytes)
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
) (bool, uint64, []byte, *warp.UnsignedMessage, error) { // @todo check the math
	totalBytes := r.TotalBytes
	valType := uint16(r.ValType)
	chunkSize := uint16(r.ChunkSize)
	imageID := r.ImageID
	if totalBytes > uint64(consts.MaxUint16)*64 {
		return false, RegisterComputeUnits, utils.ErrBytes(fmt.Errorf("total bytes %d is too large, max: %d", totalBytes, uint64(consts.MaxUint16)*64)), nil, nil
	}
	if err := storage.StoreRegistration(ctx, mu, imageID, valType, chunkSize, totalBytes); err != nil {
		return false, RegisterComputeUnits / 2, utils.ErrBytes(fmt.Errorf("%s: can't store registration", err)), nil, nil
	}
	if err := storage.InitiateDeployType(ctx, mu, imageID, valType, make([]byte, int(r.TotalBytes))); err != nil {
		return false, 3 * RegisterComputeUnits / 4, utils.ErrBytes(fmt.Errorf("%s: can't initiate deploy type", err)), nil, nil
	}
	return true, RegisterComputeUnits, nil, nil, nil
}

func UnmarshalRegister(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var registerImage Register
	p.UnpackID(true, &registerImage.ImageID)
	registerImage.ValType = p.UnpackUint64(true)
	registerImage.ChunkSize = p.UnpackUint64(true)
	registerImage.TotalBytes = p.UnpackUint64(true)
	return &registerImage, nil
}
