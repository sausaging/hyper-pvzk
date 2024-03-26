package handle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/sausaging/hyper-pvzk/requester"
	"github.com/sausaging/hyper-pvzk/storage"
)

type MidenRequestArgs struct {
	TxID            string `json:"tx_id"`
	CodeFrontEnd    string `json:"code_front_end"`
	InputsFrontEnd  string `json:"inputs_front_end"`
	OutputsFrontEnd string `json:"outputs_front_end"`
	ProofFilePath   string `json:"proof_file_path"`
}

type MidenReplyArgs struct {
	//@todo intro security field
	IsSubmitted bool `json:"is_submitted"`
}

func HandleMiden( //@todo send the hashes stored for every proofvaltype to rust server
	txID ids.ID,
	imageID ids.ID,
	proofValType uint16,
	codeFrontEnd string,
	inputsFrontEnd string,
	outputsFrontEnd string,
	baseDir string,
	endPointRequester *requester.EndpointRequester) error {
	proofKey := storage.DeployKey(imageID, proofValType)
	proofFilePath := baseDir + "/" + proofKey

	cli := endPointRequester.Cli
	uri := endPointRequester.Uri + requester.MIDENENDPOINT
	args := MidenRequestArgs{
		TxID:            txID.String(),
		CodeFrontEnd:    codeFrontEnd,
		InputsFrontEnd:  inputsFrontEnd,
		OutputsFrontEnd: outputsFrontEnd,
		ProofFilePath:   proofFilePath,
	}

	jsonData, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal miden request args: %w", err)
	}
	req, err := requester.NewRequest(uri, jsonData)
	if err != nil {
		return fmt.Errorf("failed to create new request in HandleMiden: %w", err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request in HandleMiden: %w", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body in HandleMiden: %w", err)
	}
	reply := new(MidenReplyArgs)
	err = json.Unmarshal(body, reply)
	if err != nil {
		return fmt.Errorf("failed to unmarshal miden reply: %w", err)
	}

	if reply.IsSubmitted {
		// call the submit-verify endpoint with txID
		vargs := VerifyRequestArgs{
			TxID:       txID.String(),
			VerifyType: MIDENVERIFY,
		}
		uri := endPointRequester.Uri + requester.VERIFYENDPOINT
		vjsonData, err := json.Marshal(vargs)
		if err != nil {
			return fmt.Errorf("failed to marshal verify request args in HandleMiden: %w", err)
		}
		req, err := requester.NewRequest(uri, vjsonData)
		if err != nil {
			return fmt.Errorf("failed to create new verify request in HandleMiden: %w", err)
		}
		resp, err := cli.Do(req)
		if err != nil {
			return fmt.Errorf("failed to do verify request in HandleMiden: %w", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read verify response body in HandleMiden: %w", err)
		}
		reply := new(VerifyReplyArgs)
		err = json.Unmarshal(body, reply)
		if err != nil {
			return fmt.Errorf("failed to unmarshal verify reply in HandleMiden: %w", err)
		}
	}

	return nil
}
