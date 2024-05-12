package handle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/sausaging/hyper-pvzk/requester"
	"github.com/sausaging/hyper-pvzk/storage"
)

type Plonky2RequestArgs struct {
	TxID                 string `json:"tx_id"`
	ProofFilePath        string `json:"proof_file_path"`
	CommonDataFilePath   string `json:"common_data_file_path"`
	VerifierDataFilePath string `json:"verifier_data_file_path"`
}

type Plonky2ReplyArgs struct {
	IsSubmitted bool `json:"is_submitted"`
}

func HandlePlonky2(
	txID ids.ID,
	imageID ids.ID,
	proofValType uint16,
	baseDir string,
	endPointRequester *requester.EndpointRequester,
) error { //@todo send the hashes stored for every proofvaltype to rust server
	commonDataKey := storage.DeployKey(imageID, commonDataValType)
	verifierDataValType := storage.DeployKey(imageID, verifierDataValType)
	proofKey := storage.DeployKey(imageID, proofValType)
	commonDataFilePath := baseDir + "/" + commonDataKey
	verifierDataFilePath := baseDir + "/" + verifierDataValType
	proofFilePath := baseDir + "/" + proofKey
	// call the plonky2 endpoint with elfFilePath, proofFilePath, txID
	cli := endPointRequester.Cli
	uri := endPointRequester.Uri + requester.PLONKY2ENDPOINT
	args := Plonky2RequestArgs{
		TxID:                 txID.String(),
		CommonDataFilePath:   commonDataFilePath,
		VerifierDataFilePath: verifierDataFilePath,
		ProofFilePath:        proofFilePath,
	}

	jsonData, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal plonky2 request args: %w", err)
	}
	req, err := requester.NewRequest(uri, jsonData)
	if err != nil {
		return fmt.Errorf("failed to create new request in HandlePlonky2: %w", err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request in HandlePlonky2: %w", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body in HandlePlonky2: %w", err)
	}
	reply := new(Plonky2ReplyArgs)
	err = json.Unmarshal(body, reply)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json in HandlePlonky2: %w", err)
	}

	if reply.IsSubmitted {
		// call the submit-verify endpoint with txID
		vargs := VerifyRequestArgs{
			TxID:       txID.String(),
			VerifyType: PLONKY2VERIFY,
		}
		uri := endPointRequester.Uri + requester.VERIFYENDPOINT
		vjsonData, err := json.Marshal(vargs)
		if err != nil {
			return fmt.Errorf("failed to marshal verify request args in HandlePlonky2: %w", err)
		}
		req, err := requester.NewRequest(uri, vjsonData)
		if err != nil {
			return fmt.Errorf("failed to create new verify request in HandlePlonky2: %w", err)
		}
		resp, err := cli.Do(req)
		if err != nil {
			return fmt.Errorf("failed to do verify request in HandlePlonky2: %w", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read verify response body in HandlePlonky2: %w", err)
		}
		reply := new(VerifyReplyArgs)
		err = json.Unmarshal(body, reply)
		if err != nil {
			return fmt.Errorf("failed to unmarshal verify reply in HandlePlonky2: %w", err)
		}
	}
	return nil
}
