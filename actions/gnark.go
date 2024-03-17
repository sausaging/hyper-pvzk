package actions

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/state"
	"github.com/ava-labs/hypersdk/utils"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/plonk"
	wit "github.com/consensys/gnark/backend/witness"
	mconsts "github.com/sausaging/hyper-pvzk/consts"
	"github.com/sausaging/hyper-pvzk/storage"
)

var _ chain.Action = (*Gnark)(nil)

type Gnark struct {
	ImageID             ids.ID `json:"image_id"`
	ProvingSystem       bool   `json:"proving_sytem"`
	Curve               uint64 `json:"curve"`
	ProofValType        uint64 `json:"proof_val_type"`
	PubWitValType       uint64 `json:"pub_wit_val_type"`
	VerificationValType uint64 `json:"verification_val_type"`
}

func (g *Gnark) GetTypeID() uint8 {
	return mconsts.GnarkID
}

func (g *Gnark) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.DeployKey(g.ImageID, uint16(g.ProofValType))):        state.All,
		string(storage.DeployKey(g.ImageID, uint16(g.PubWitValType))):       state.All,
		string(storage.DeployKey(g.ImageID, uint16(g.VerificationValType))): state.All,
	}
}

func (g *Gnark) StateKeysMaxChunks() []uint16 {
	return []uint16{consts.MaxUint16, consts.MaxUint16, consts.MaxUint16}
}

func (g *Gnark) OutputsWarpMessage() bool {
	return false
}

func (g *Gnark) MaxComputeUnits(chain.Rules) uint64 {
	return GnarkComputeUnits
}

func (g *Gnark) Size() int {
	return consts.IDLen + 4*consts.Uint64Len + consts.BoolLen
}

func (g *Gnark) Marshal(p *codec.Packer) {
	p.PackID(g.ImageID)
	p.PackBool(g.ProvingSystem)
	p.PackUint64(g.Curve)
	p.PackUint64(g.ProofValType)
	p.PackUint64(g.PubWitValType)
	p.PackUint64(g.VerificationValType)
}

func UnmarshalGnark(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var gnark Gnark
	p.UnpackID(true, &gnark.ImageID)
	gnark.ProvingSystem = p.UnpackBool()
	gnark.Curve = p.UnpackUint64(true)
	gnark.ProofValType = p.UnpackUint64(true)
	gnark.PubWitValType = p.UnpackUint64(true)
	gnark.VerificationValType = p.UnpackUint64(true)
	return &gnark, nil
}

func (*Gnark) ValidRange(chain.Rules) (int64, int64) {
	return -1, -1
}

func (g *Gnark) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	if g.Curve > 11 {
		return false, 1000, utils.ErrBytes(errors.New("proving system not supported")), nil, nil
	}
	ps := ecc.ID(g.Curve)
	proofBytes, err := storage.GetDeployType(ctx, mu, g.ImageID, uint16(g.ProofValType))
	if err != nil {
		return false, 1000, utils.ErrBytes(fmt.Errorf("%s: can't get proof from state", err)), nil, nil
	}
	pubWitBytes, err := storage.GetDeployType(ctx, mu, g.ImageID, uint16(g.PubWitValType))
	if err != nil {
		return false, 2000, utils.ErrBytes(fmt.Errorf("%s: can't get public witness from state", err)), nil, nil
	}
	verificationBytes, err := storage.GetDeployType(ctx, mu, g.ImageID, uint16(g.VerificationValType))
	if err != nil {
		return false, 3000, utils.ErrBytes(fmt.Errorf("%s: can't get verification key from state", err)), nil, nil
	}
	pubWit, err := wit.New(ps.ScalarField())
	if err != nil {
		return false, 2000, utils.ErrBytes(fmt.Errorf("%s: can't create public witness", err)), nil, nil
	}
	pubWit.ReadFrom(bytes.NewBuffer(pubWitBytes))

	if g.ProvingSystem { //groth16
		proof := groth16.NewProof(ps)
		vk := groth16.NewVerifyingKey(ps)
		proof.ReadFrom(bytes.NewBuffer(proofBytes))
		vk.ReadFrom(bytes.NewBuffer(verificationBytes))
		if err := groth16.Verify(proof, vk, pubWit); err != nil {
			stri := fmt.Sprint(md5.Sum(proofBytes), md5.Sum(verificationBytes), md5.Sum(pubWitBytes))
			return false, 10000, utils.ErrBytes(fmt.Errorf("%s: verification failed, proving: %s,check sum: %s", err, ps.String(), stri)), nil, nil
		}
	} else { //plonk
		proof := plonk.NewProof(ps)
		vk := plonk.NewVerifyingKey(ps)
		proof.ReadFrom(bytes.NewBuffer(proofBytes))
		vk.ReadFrom(bytes.NewBuffer(verificationBytes))
		if err := plonk.Verify(proof, vk, pubWit); err != nil {
			return false, 10000, utils.ErrBytes(fmt.Errorf("%s: verification failed", err)), nil, nil
		}
	}
	return true, 8000, nil, nil, nil
}
