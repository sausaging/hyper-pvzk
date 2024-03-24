package actions

import (
	"context"
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/validators"
	"github.com/ava-labs/avalanchego/utils/math"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/crypto/bls"
	"github.com/ava-labs/hypersdk/state"
	"github.com/ava-labs/hypersdk/utils"
	mauth "github.com/sausaging/hyper-pvzk/auth"
	mconsts "github.com/sausaging/hyper-pvzk/consts"
	"github.com/sausaging/hyper-pvzk/storage"
)

//@todo validators vote

var _ chain.Action = (*ValidatorVote)(nil)

type ValidatorVote struct {
	TxID      ids.ID `json:"tx_id"`
	Vote      bool   `json:"vote"`
	Signature []byte `json:"signature"`
	PublicKey []byte `json:"public_key"`
}

func (*ValidatorVote) GetTypeID() uint8 {
	return mconsts.ValidatorVote
}

func (v *ValidatorVote) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.TimeOutKey(v.TxID)):     state.All,
		string(storage.WeightKey(v.TxID)):      state.All,
		string(storage.StatusKey(txID)):        state.All,
		string(storage.VoteKey(v.TxID, actor)): state.All,
	}
}

func (*ValidatorVote) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.TimeOutChunks, storage.TimeOutChunks, storage.TimeOutChunks, storage.TimeOutChunks}
}

func (*ValidatorVote) OutputsWarpMessage() bool {
	return false
}

func (*ValidatorVote) MaxComputeUnits(chain.Rules) uint64 {
	return ValidatorVoteComputeUnits
}

func (v *ValidatorVote) Size() int {
	return consts.IDLen + consts.BoolLen + len(v.Signature) + len(v.PublicKey)
}

func (v *ValidatorVote) Marshal(p *codec.Packer) {
	p.PackID(v.TxID)
	p.PackBool(v.Vote)
	p.PackBytes(v.Signature)
	p.PackBytes(v.PublicKey)
}

func UnmarshalValidatorVote(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var vv ValidatorVote
	p.UnpackID(true, &vv.TxID)
	vv.Vote = p.UnpackBool()
	p.UnpackBytes(bls.SignatureLen, true, &vv.Signature)
	p.UnpackBytes(bls.PublicKeyLen, true, &vv.PublicKey)
	return &vv, nil
}

func (*ValidatorVote) ValidRange(chain.Rules) (int64, int64) {
	return -1, -1
}

func (v *ValidatorVote) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	ts int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	// we can also do this as a warp message.
	// but how can we verify weight? so we are following this way
	// can be played out with
	vTXID := v.TxID
	timeOut, err := storage.GetTimeOut(ctx, mu, vTXID)
	if err != nil {
		return false, 1000, utils.ErrBytes(fmt.Errorf("%s: cant get timeout from storage", err)), nil, nil
	}
	if ts > timeOut {
		return false, 1000, utils.ErrBytes(fmt.Errorf("timeout: can't vote now. timestamp: %d, timeout: %d", ts, timeOut)), nil, nil
	}
	pubKey, err := bls.PublicKeyFromBytes(v.PublicKey)
	if err != nil {
		return false, 2000, utils.ErrBytes(fmt.Errorf("%s: invalid public key", err)), nil, nil
	}
	sig, err := bls.SignatureFromBytes(v.Signature)
	if err != nil {
		return false, 2500, utils.ErrBytes(fmt.Errorf("%s: cant get signature from bytes", err)), nil, nil
	}
	f, _ := rules.FetchCustom("")
	currentValidators := f.(func(ctx context.Context) (map[ids.NodeID]*validators.GetValidatorOutput, map[string]struct{}))
	validators, publicKeys := currentValidators(ctx)
	// proceed if pubkey exists in the list of validators
	if _, ok := publicKeys[string(v.PublicKey)]; !ok {
		return false, 3000, utils.ErrBytes(fmt.Errorf("not a validator")), nil, nil
	}
	var totalWeight uint64
	for _, vdr := range validators {
		if vdr.PublicKey == pubKey {
			totalWeight, err = math.Add64(totalWeight, vdr.Weight)
			if err != nil {
				return false, 3500, utils.ErrBytes(fmt.Errorf("%s: weight overflow", err)), nil, nil
			}
			w := vdr.Weight
			bls := mauth.BLS{
				Signer:    pubKey,
				Signature: sig,
			}
			msg := GetMessage(vTXID, v.Vote)
			unSigMsg, err := warp.NewUnsignedMessage(rules.NetworkID(), rules.ChainID(), msg)
			if err != nil {
				return false, 4000, utils.ErrBytes(fmt.Errorf("%s: cant create unsigned message", err)), nil, nil
			}
			err = bls.Verify(ctx, unSigMsg.Bytes())
			if err != nil {
				return false, 4000, utils.ErrBytes(fmt.Errorf("%s: cant verify signature", err)), nil, nil
			}
			// check if this validator voted earlier -> if so dont add
			if err := storage.GetVote(ctx, mu, vTXID, actor); err != nil {
				return false, 5000, utils.ErrBytes(fmt.Errorf("%s: already voted", err)), nil, nil
			}
			storage.UpdateWeight(ctx, mu, vTXID, w, totalWeight)
			storage.StoreVote(ctx, mu, vTXID, actor)
		}
	}

	return true, ValidatorVoteComputeUnits, nil, nil, nil
}
