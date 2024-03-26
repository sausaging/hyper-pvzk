package handle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/sausaging/hyper-pvzk/requester"
	"github.com/sausaging/hyper-pvzk/storage"
)

type RiscZeroArgs struct {
	TxID            string `json:"tx_id"`
	RiscZeroImageID string `json:"risc_zero_image_id"`
	ProofFilePath   string `json:"proof_file_path"`
}

type RiscZeroReplyArgs struct {
	IsSubmitted bool `json:"is_submitted"`
}

func HandleRiscZero( //@todo send the hashes stored for every proofvaltype to rust server
	txID ids.ID,
	imageID ids.ID,
	proofValType uint16,
	RiscZeroImageID string,
	baseDir string,
	endPointRequester *requester.EndpointRequester) error {

	proofKey := storage.DeployKey(imageID, proofValType)
	proofFilePath := baseDir + "/" + proofKey

	cli := endPointRequester.Cli
	uri := endPointRequester.Uri + requester.RISCZEROENDPOINT
	args := RiscZeroArgs{
		TxID:            txID.String(),
		RiscZeroImageID: RiscZeroImageID,
		ProofFilePath:   proofFilePath,
	}

	jsonData, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal risczero request args: %w", err)
	}
	req, err := requester.NewRequest(uri, jsonData)
	if err != nil {
		return fmt.Errorf("failed to create new request in HandleRiscZero: %w", err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request in HandleRiscZero: %w", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body in HandleRiscZero: %w", err)
	}
	reply := new(RiscZeroReplyArgs)
	err = json.Unmarshal(body, reply)
	if err != nil {
		return fmt.Errorf("failed to unmarshal risczero reply: %w", err)
	}

	if reply.IsSubmitted {
		// call the submit-verify endpoint with txID
		vargs := VerifyRequestArgs{
			TxID:       txID.String(),
			VerifyType: SP1VERIFY,
		}
		uri := endPointRequester.Uri + requester.VERIFYENDPOINT
		vjsonData, err := json.Marshal(vargs)
		if err != nil {
			return fmt.Errorf("failed to marshal verify request args in HandleRiscZero: %w", err)
		}
		req, err := requester.NewRequest(uri, vjsonData)
		if err != nil {
			return fmt.Errorf("failed to create new verify request in HandleRiscZero: %w", err)
		}
		resp, err := cli.Do(req)
		if err != nil {
			return fmt.Errorf("failed to do verify request in HandleRiscZero: %w", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read verify response body in HandleRiscZero: %w", err)
		}
		reply := new(VerifyReplyArgs)
		err = json.Unmarshal(body, reply)
		if err != nil {
			return fmt.Errorf("failed to unmarshal verify reply in HandleRiscZero: %w", err)
		}
	}
	return nil
}
