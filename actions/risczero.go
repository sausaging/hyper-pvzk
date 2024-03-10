package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/state"
	"github.com/ava-labs/hypersdk/utils"
	mconsts "github.com/sausaging/hyper-pvzk/consts"
	"github.com/sausaging/hyper-pvzk/requester"
	"github.com/sausaging/hyper-pvzk/storage"
)

// @todo build cmd and build stratagies

var _ chain.Action = (*RiscZero)(nil)

type RiscZero struct {
	ImageID         ids.ID `json:"image_id"`
	ProofValType    uint64 `json:"proof_val_type"`
	RiscZeroImageID string `json:"risc_zero_image_id"`
}

type RiscZeroArgs struct {
	RiscZeroImageID string `json:"risc_zero_image_id"`
	ProofFilePath   string `json:"proof_file_path"`
}

type RiscZeroReplyArgs struct {
	IsValid bool `json:"is_valid"`
}

func (*RiscZero) GetTypeID() uint8 {
	return mconsts.RiscZeroID
}

func (r *RiscZero) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.DeployKey(r.ImageID, uint16(r.ProofValType))): state.All,
	}
}

func (*RiscZero) StateKeysMaxChunks() []uint16 {
	return []uint16{consts.MaxUint16}
}

func (*RiscZero) OutputsWarpMessage() bool {
	return false
}

func (*RiscZero) MaxComputeUnits(chain.Rules) uint64 {
	return RiscZeroComputeUnits
}

func (r *RiscZero) Size() int {
	return consts.IDLen + consts.Uint64Len + len(r.RiscZeroImageID)
}

func (r *RiscZero) Marshal(p *codec.Packer) {
	p.PackID(r.ImageID)
	p.PackUint64(r.ProofValType)
	p.PackString(r.RiscZeroImageID)
}

func UnmarshalRiscZero(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var riscZero RiscZero
	p.UnpackID(true, &riscZero.ImageID)
	riscZero.ProofValType = p.UnpackUint64(true)
	riscZero.RiscZeroImageID = p.UnpackString(true)
	return &riscZero, nil
}

func (*RiscZero) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (r *RiscZero) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {

	imageID := r.ImageID
	proofValType := r.ProofValType
	riscZeroImageID := r.RiscZeroImageID

	proofJsonBytes, err := storage.GetDeployType(ctx, mu, imageID, uint16(proofValType))
	if err != nil {
		return false, 1000, utils.ErrBytes(fmt.Errorf("%s: can't get proof at type %d from state", err, proofValType)), nil, nil
	}

	proofFilePath := requester.BASEFILEPATH + "risczero" + imageID.String() + txID.String() + strconv.Itoa(int(proofValType)) + ".json"
	if err := WriteFile(proofFilePath, proofJsonBytes); err != nil {
		return false, 2000, utils.ErrBytes(fmt.Errorf("%s: can't write proof to disk", err)), nil, nil
	}

	cli, uri := requester.GetRequesterInstance(rules)
	endPointUri := uri + requester.RISCZEROENDPOINT
	riscZeroArgs := RiscZeroArgs{
		RiscZeroImageID: riscZeroImageID,
		ProofFilePath:   proofFilePath,
	}

	jsonData, err := json.Marshal(riscZeroArgs)
	if err != nil {
		return false, 3000, utils.ErrBytes(fmt.Errorf("%s: can't marshal json", err)), nil, nil
	}

	req, err := requester.NewRequest(endPointUri, jsonData)
	if err != nil {
		return false, 4000, utils.ErrBytes(fmt.Errorf("%s: can't request http", err)), nil, nil
	}

	resp, err := cli.Do(req)
	if err != nil {
		return false, 5000, utils.ErrBytes(fmt.Errorf("%s: can't do request", err)), nil, nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, 6000, utils.ErrBytes(fmt.Errorf("%s: can't decode client response", err)), nil, nil
	}
	reply := new(RiscZeroReplyArgs)
	err = json.Unmarshal(body, reply)
	if err != nil {
		return false, 6000, utils.ErrBytes(fmt.Errorf("%s: can't unmarshal json", err)), nil, nil
	}

	return reply.IsValid, 6000, nil, nil, nil
}
