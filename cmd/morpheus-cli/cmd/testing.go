package cmd

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sausaging/hyper-pvzk/actions"
	"github.com/spf13/cobra"
)

type txData struct {
	ImageID      ids.ID
	ProofValType uint64
	ChunkIndex   uint64
	Data         []byte
}

var chunkSize = 10 * 1024

var testingCmd = &cobra.Command{
	Use: "testing",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}
var registerCmd = &cobra.Command{
	Use: "register",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, bcli, ws, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		imageID, err := handler.Root().PromptID("image id")
		if err != nil {
			return err
		}
		// chunkSize, err := handler.Root().PromptInt("chunk size", int(consts.MaxUint16))
		// if err != nil {
		// 	return err
		// }
		valType, err := handler.Root().PromptInt("val type(1 for ELF, rest for proofs)", int(consts.MaxUint16))
		if err != nil {
			return err
		}
		filePath, err := handler.Root().PromptString("file path to register total bytes", 1, consts.MaxInt)
		if err != nil {
			return err
		}
		code, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}
		totalBytes := uint64(len(code))
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}
		_, _, err = sendAndWait(ctx, nil, &actions.Register{
			ImageID:    imageID,
			ChunkSize:  uint64(chunkSize),
			ValType:    uint64(valType),
			TotalBytes: totalBytes,
		}, cli, bcli, ws, factory, true)
		return err
	},
}

var deployCmd = &cobra.Command{
	Use: "deploy",
	RunE: func(*cobra.Command, []string) error {

		ctx := context.Background()
		_, _, factory, cli, bcli, ws, err := handler.DefaultActor()
		// go listenAndRetry(ws, bcli, cli, factory)
		if err != nil {
			return err
		}
		imageID, err := handler.Root().PromptID("image id")
		if err != nil {
			return err
		}
		valType, err := handler.Root().PromptInt("proof val type(1 for ELF, rest for proofs)", int(consts.MaxUint16))
		if err != nil {
			return err
		}
		filePath, err := handler.Root().PromptString("file path to deploy", 1, consts.MaxInt)
		if err != nil {
			return err
		}
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}
		code, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}
		totalChunks := (len(code) + chunkSize - 1) / chunkSize // Include the last chunk
		for chunkIndex := 0; chunkIndex < totalChunks; chunkIndex++ {
			start := uint64(chunkIndex) * uint64(chunkSize)
			end := (chunkIndex + 1) * chunkSize
			if end > len(code) {
				end = len(code)
			}
			txID, err := send(ctx, nil, &actions.Deploy{
				ImageID:      imageID,
				ProofvalType: uint64(valType),
				Data:         code[start:end],
				ChunkIndex:   uint64(chunkIndex + 1), // figure out why parser fails when chunkIndex is 0?
			}, cli, bcli, ws, factory)
			if err != nil {
				return err
			}
			WriteToJson(txData{
				ImageID:      imageID,
				ProofValType: uint64(valType),
				ChunkIndex:   uint64(chunkIndex),
				Data:         code[start:end],
			}, txID)

			utils.Outf("{{yellow}} sent chunk %d{{/}}\n{{green}} offset: %d, file size: %d{{/}}\n", chunkIndex, end, len(code))
			if chunkIndex%5 == 0 && chunkIndex != 0 {
				time.Sleep(15 * time.Second)
			}
		}

		return err
	},
}

var retryDeployCmd = &cobra.Command{
	Use: "retry",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		fileName := "./data.json"
		_, _, factory, cli, bcli, ws, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(fileName)
		if err != nil {
			log.Fatal("Error reading file:", err)
		}
		arr := make([]ids.ID, 0)
		var jsonData map[string]txData
		if err := json.Unmarshal(data, &jsonData); err != nil {
			// Handle potential errors if the file is empty or invalid JSON
			jsonData = make(map[string]txData)
		}
		for txID, txDa := range jsonData {
			txID, err := ids.FromString(txID)
			if err != nil {
				return err
			}
			// txDa2 := txDa.(txData)
			contains, success, _, _, err := bcli.Tx(ctx, txID)
			if err != nil {
				return err
			}

			if contains && success {
				arr = append(arr, txID)
			} else {

				utils.Outf("{{yellow}}retrying tx:{{/}} %s, contains: %t, success: %t, {{cyan}}chunk: %d{{/}}\n", txID, contains, success, txDa.ChunkIndex)
				txID2, _ := send(ctx, nil, &actions.Deploy{
					ImageID:      txDa.ImageID,
					ProofvalType: txDa.ProofValType,
					ChunkIndex:   txDa.ChunkIndex + 1,
					Data:         txDa.Data,
				}, cli, bcli, ws, factory)
				arr = append(arr, txID)
				jsonData[txID2.String()] = txDa
			}
		}
		for _, i := range arr {
			delete(jsonData, i.String())
		}

		newData, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			utils.Outf("{{red}}error marshalling data: %s{{/}}\n", err)
		}

		// Write the updated JSON data back to the file
		err = os.WriteFile(fileName, newData, 0644)
		if err != nil {
			utils.Outf("{{red}}error writing to file: %s{{/}}\n", err)
		}
		return err
	},
}

var verifyCmd = &cobra.Command{
	Use: "verify",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, bcli, ws, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		imageID, err := handler.Root().PromptID("image id")
		if err != nil {
			return err
		}
		valType, err := handler.Root().PromptInt("proof val type(1 + for proofs)", int(consts.MaxUint16))
		if err != nil {
			return err
		}
		// @todo verifying system type
		verifyType, err := handler.Root().PromptInt("verification type: 1 -> SP1, 2 -> Miden, 3 -> Risc0", 10)
		if err != nil {
			return err
		}
		var action chain.Action
		if verifyType == 1 {
			action = &actions.SP1{
				ImageID:      imageID,
				ProofValType: uint64(valType),
			}
		} else if verifyType == 2 {
			//@todo do ask next questions
			codeFrontEnd, err := handler.Root().PromptString("codeFrontEnd", 1, consts.MaxInt)
			if err != nil {
				return err
			}
			inputsFrontEnd, err := handler.Root().PromptString("inputsFrontEnd", 1, consts.MaxInt)
			if err != nil {
				return err
			}
			outputsFrontEnd, err := handler.Root().PromptString("outputsForntEnd", 1, consts.MaxInt)
			if err != nil {
				return err
			}
			action = &actions.Miden{
				ImageID:         imageID,
				ProofValType:    uint64(valType),
				CodeFrontEnd:    codeFrontEnd,
				InputsFrontEnd:  inputsFrontEnd,
				OutputsFrontEnd: outputsFrontEnd,
			}

		} else if verifyType == 3 {
			input, err := handler.Root().PromptString("risc zero image id", 1, 128)
			if err != nil {
				return err
			}
			riscZeroImageID := common.FromHex(input)
			action = &actions.RiscZero{
				ImageID:         imageID,
				ProofValType:    uint64(valType),
				RiscZeroImageID: [32]byte(riscZeroImageID),
			}
		} else {
			return ErrInvalidVerificationType
		}
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}
		_, _, err = sendAndWait(ctx, nil, action, cli, bcli, ws, factory, true)
		return err

	},
}

// func listenAndRetry(ws *rpc.WebSocketClient, bcli *brpc.JSONRPCClient, cli *rpc.JSONRPCClient, factory chain.AuthFactory) {
// 	ctx := context.Background()
// 	ws.RegisterBlocks()
// 	pendingTxs := make(map[ids.ID]txData)
// 	p, _ := bcli.Parser(context.TODO())
// 	utils.Outf("{{yellow}}listening for blocks{{/}}\n")
// 	for {
// 		txD := <-status
// 		pendingTxs[txD.txID] = txD.txD
// 		utils.Outf("{{yellow}}pending txs:{{/}} %d\n", len(pendingTxs))
// 		blk, result, _, err := ws.ListenBlock(ctx, p)
// 		if err != nil {
// 			utils.Outf("{{red}} error listening block: %s{{/}}\n", err)
// 		}
// 		for i, tx := range blk.Txs {
// 			for txID, txdata := range pendingTxs {
// 				if txID == tx.ID() {
// 					if result[i].Success {
// 						utils.Outf("{{green}}tx %s succeeded{{/}}\n", txID)
// 						delete(pendingTxs, txID)
// 					} else {
// 						utils.Outf("{{red}}tx %s failed, retrying{{/}}\n", txID)
// 						delete(pendingTxs, txID)
// 						txID, err := send(ctx, nil, &actions.Deploy{
// 							ImageID:      txdata.ImageID,
// 							ProofvalType: txdata.ProofValType,
// 							ChunkIndex:   txdata.ChunkIndex,
// 							Data:         txdata.Data,
// 						}, cli, bcli, ws, factory)
// 						if err != nil {
// 							utils.Outf("{{red}}%s{{/}}\n", err)
// 						} else {
// 							pendingTxs[txID] = txData{
// 								ImageID:      txdata.ImageID,
// 								ProofValType: txdata.ProofValType,
// 								ChunkIndex:   txdata.ChunkIndex,
// 								Data:         txdata.Data,
// 							}

// 						}
// 					}
// 				}
// 			}

// 		}

// 		// utils.Outf("{{yellow}}skipping unexpected transaction:{{/}} %s\n", tx.ID())
// 	}
// }

func WriteToJson(obj txData, txID ids.ID) {
	fileName := "./data.json"

	// Read the existing data from the file
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Fatal("Error reading file:", err)
	}
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		// Handle potential errors if the file is empty or invalid JSON
		jsonData = make(map[string]interface{})
	}
	jsonData[txID.String()] = obj
	newData, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		log.Fatal("Error marshalling data:", err)
	}

	// Write the updated JSON data back to the file
	err = ioutil.WriteFile(fileName, newData, 0644)
	if err != nil {
		log.Fatal("Error writing to file:", err)
	}

	// log.Println("Successfully added object to", fileName)
}
