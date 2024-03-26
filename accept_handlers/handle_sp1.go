package handle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/sausaging/hyper-pvzk/requester"
	"github.com/sausaging/hyper-pvzk/storage"
)

type SP1RequestArgs struct {
	TxID          string `json:"tx_id"`
	ELFFilePath   string `json:"elf_file_path"`
	ProofFilePath string `json:"proof_file_path"`
}

type SP1ReplyArgs struct {
	IsSubmitted bool `json:"is_submitted"`
}

const elfValType = 1

func HandleSP1(
	txID ids.ID,
	imageID ids.ID,
	proofValType uint16,
	baseDir string,
	endPointRequester *requester.EndpointRequester,
) error { //@todo send the hashes stored for every proofvaltype to rust server
	elfKey := storage.DeployKey(imageID, elfValType)
	proofKey := storage.DeployKey(imageID, proofValType)
	elfFilePath := baseDir + "/" + elfKey
	proofFilePath := baseDir + "/" + proofKey
	// call the sp1 endpoint with elfFilePath, proofFilePath, txID
	cli := endPointRequester.Cli
	uri := endPointRequester.Uri + requester.SP1ENDPOINT
	args := SP1RequestArgs{
		TxID:          txID.String(),
		ELFFilePath:   elfFilePath,
		ProofFilePath: proofFilePath,
	}

	jsonData, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal sp1 request args: %w", err)
	}
	req, err := requester.NewRequest(uri, jsonData)
	if err != nil {
		return fmt.Errorf("failed to create new request in HandleSP1: %w", err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request in HandleSP1: %w", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body in HandleSP1: %w", err)
	}
	reply := new(SP1ReplyArgs)
	err = json.Unmarshal(body, reply)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json in HandleSP1: %w", err)
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
			return fmt.Errorf("failed to marshal verify request args in HandleSP1: %w", err)
		}
		req, err := requester.NewRequest(uri, vjsonData)
		if err != nil {
			return fmt.Errorf("failed to create new verify request in HandleSP1: %w", err)
		}
		resp, err := cli.Do(req)
		if err != nil {
			return fmt.Errorf("failed to do verify request in HandleSP1: %w", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read verify response body in HandleSP1: %w", err)
		}
		reply := new(VerifyReplyArgs)
		err = json.Unmarshal(body, reply)
		if err != nil {
			return fmt.Errorf("failed to unmarshal verify reply in HandleSP1: %w", err)
		}
	}
	return nil
}
