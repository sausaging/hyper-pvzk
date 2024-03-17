package main

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// 	"os"
// 	"time"

// 	"github.com/ava-labs/avalanche-network-runner/client"
// 	"github.com/ava-labs/avalanchego/ids"
// 	"github.com/ava-labs/avalanchego/utils/logging"
// 	"github.com/ava-labs/hypersdk/chain"
// 	"github.com/ava-labs/hypersdk/crypto/ed25519"

// 	"github.com/ava-labs/hypersdk/rpc"
// 	"github.com/ava-labs/hypersdk/utils"
// 	"github.com/sausaging/hyper-pvzk/actions"
// 	"github.com/sausaging/hyper-pvzk/auth"
// 	brpc "github.com/sausaging/hyper-pvzk/rpc"
// )

// var chunkSize = 10 * 1024
// var elfFilePath = "./demo.elf"
// var sp1ProofFilePath = "./demo.sp1"
// var midenProofFilePath = "./demo.miden"
// var risc0ProofFilePath = "./demo.risc0"

// type txData struct {
// 	ImageID      ids.ID
// 	ProofValType uint64
// 	ChunkIndex   uint64
// 	Data         []byte
// }

// func register(
// 	action chain.Action,
// 	bcli *brpc.JSONRPCClient,
// 	cli *rpc.JSONRPCClient,
// 	ws *rpc.WebSocketClient,
// 	factory chain.AuthFactory,
// 	parser *chain.Parser,
// ) (bool, ids.ID) {
// 	ctx := context.Background()
// 	_, tx, _, err := cli.GenerateTransaction(ctx, *parser, nil, action, factory)
// 	if err != nil {
// 		return false, ids.Empty
// 	}
// 	if err := ws.RegisterTx(tx); err != nil {
// 		return false, ids.Empty
// 	}
// 	var result *chain.Result
// 	for {
// 		txID, txErr, txResult, err := ws.ListenTx(ctx)
// 		if err != nil {
// 			return false, ids.Empty
// 		}
// 		if txErr != nil {
// 			return false, ids.Empty
// 		}
// 		if txID == tx.ID() {
// 			result = txResult
// 			break
// 		}
// 		utils.Outf("{{yellow}}skipping unexpected transaction:{{/}} %s\n", tx.ID())
// 	}
// 	return result.Success, tx.ID()
// }

// func deploy(
// 	filePath string,
// 	valType uint64,
// 	imageID ids.ID,
// 	bcli *brpc.JSONRPCClient,
// 	cli *rpc.JSONRPCClient,
// 	factory chain.AuthFactory,
// 	parser *chain.Parser,
// ) error {
// 	ctx := context.Background()
// 	code, err := ioutil.ReadFile(elfFilePath)
// 	if err != nil {
// 		panic(err)
// 	}
// 	totalChunks := (len(code) + chunkSize - 1) / chunkSize // Include the last chunk
// 	for chunkIndex := 0; chunkIndex < totalChunks; chunkIndex++ {
// 		start := uint64(chunkIndex) * uint64(chunkSize)
// 		end := (chunkIndex + 1) * chunkSize
// 		if end > len(code) {
// 			end = len(code)
// 		}
// 		action := &actions.Deploy{
// 			ImageID:      imageID,
// 			ProofvalType: valType,
// 			Data:         code[start:end],
// 			ChunkIndex:   uint64(chunkIndex + 1),
// 		}
// 		f, tx, _, err := cli.GenerateTransaction(ctx, *parser, nil, action, factory)
// 		if err != nil {
// 			return err
// 		}
// 		if err := f(ctx); err != nil {
// 			utils.Outf("Error submitting Tx")
// 			return nil
// 		}
// 		utils.Outf("{{yellow}}Submited txID:{{/}} %s {{yellow}}to mempool{{/}}\n", tx.ID())
// 		WriteToJson(txData{
// 			ImageID:      imageID,
// 			ProofValType: valType,
// 			ChunkIndex:   uint64(chunkIndex),
// 			Data:         code[start:end],
// 		}, tx.ID())

// 		utils.Outf("{{yellow}} sent chunk{{/}} %d\n{{green}} offset:{{/}} %d, {{green}}file size:{{/}} %d\n", chunkIndex+1, end, len(code))
// 		if chunkIndex%25 == 0 && chunkIndex != 0 {
// 			time.Sleep(15 * time.Second)
// 		}
// 	}
// 	return nil
// }

// func retry(
// 	bcli *brpc.JSONRPCClient,
// 	cli *rpc.JSONRPCClient,
// 	factory chain.AuthFactory,
// 	parser *chain.Parser,
// ) error {
// 	ctx := context.Background()
// 	fileName := "./data.json"
// 	data, err := os.ReadFile(fileName)
// 	if err != nil {
// 		log.Fatal("Error reading file:", err)
// 	}
// 	arr := make([]ids.ID, 0)
// 	var jsonData map[string]txData
// 	if err := json.Unmarshal(data, &jsonData); err != nil {
// 		// Handle potential errors if the file is empty or invalid JSON
// 		jsonData = make(map[string]txData)
// 	}
// 	for txID, txDa := range jsonData {
// 		txID, err := ids.FromString(txID)
// 		if err != nil {
// 			return err
// 		}
// 		contains, success, _, _, err := bcli.Tx(ctx, txID)
// 		if err != nil {
// 			return err
// 		}

// 		if contains && success {
// 			arr = append(arr, txID)
// 		} else {
// 			utils.Outf("{{yellow}}retrying tx:{{/}} %s, contains: %t, success: %t, {{cyan}}chunk: %d{{/}}\n", txID, contains, success, txDa.ChunkIndex)

// 		}
// 	}
// 	return nil
// }
// func verify() {

// }

// func main() {
// 	//load private key and prepare auth factory
// 	// should call from root, key is ed25519
// 	ctx := context.Background()
// 	p, err := utils.LoadBytes("./demo.pk", ed25519.PrivateKeyLen)
// 	if err != nil {
// 		panic(err)
// 	}
// 	authFactory := auth.NewED25519Factory(ed25519.PrivateKey(p))

// 	// Load new items from ANR
// 	anrCli, err := client.New(client.Config{
// 		Endpoint:    "0.0.0.0:12352",
// 		DialTimeout: 10 * time.Second,
// 	}, logging.NoLog{})
// 	if err != nil {
// 		panic(err)
// 	}
// 	status, err := anrCli.Status(ctx)
// 	if err != nil {
// 		panic(err)
// 	}
// 	chainIDs := make([]ids.ID, 0)
// 	for chain := range status.ClusterInfo.CustomChains {
// 		chainID, err := ids.FromString(chain)
// 		if err != nil {
// 			panic(err)
// 		}
// 		chainIDs = append(chainIDs, chainID)
// 	}

// 	// go through all nodes and get their rpc endpoints
// 	var uris []string
// 	for _, nodeInfo := range status.ClusterInfo.NodeInfos {
// 		if len(nodeInfo.WhitelistedSubnets) == 0 {
// 			continue
// 		}
// 		chainID := chainIDs[0]
// 		uris = append(uris, fmt.Sprintf("%s/ext/bc/%s", nodeInfo.Uri, chainID))
// 	}
// 	cli := rpc.NewJSONRPCClient(uris[0])
// 	ws, err := rpc.NewWebSocketClient(uris[0], rpc.DefaultHandshakeTimeout, 1024, 1024)
// 	if err != nil {
// 		panic(err)
// 	}
// 	networkID, _, _, err := cli.Network(context.TODO())
// 	if err != nil {
// 		panic(err)
// 	}
// 	bcli := brpc.NewJSONRPCClient(uris[0], networkID, chainIDs[0])
// 	parser, err := bcli.Parser(ctx)
// 	if err != nil {
// 		panic(err)
// 	}
// 	sp1ImageID, _ := ids.FromString("2w")
// 	code, err := ioutil.ReadFile(elfFilePath)
// 	if err != nil {
// 		panic(err)
// 	}
// 	totalBytes := uint64(len(code))

// 	regActionSP1ELF := &actions.Register{
// 		ImageID:    sp1ImageID,
// 		ChunkSize:  uint64(chunkSize),
// 		ValType:    1,
// 		TotalBytes: totalBytes,
// 	}
// 	utils.Outf("registering SP1 ELF\n")
// 	result, txID := register(regActionSP1ELF, bcli, cli, ws, authFactory, &parser)
// 	if !result {
// 		panic("register failed")
// 	} else {
// 		utils.Outf("register success, txID: %s\n", txID)
// 	}
// 	utils.Outf("deploying SP1 ELF\n")
// 	err = deploy(elfFilePath, 1, sp1ImageID, bcli, cli, authFactory, &parser)
// 	if err != nil {
// 		panic(err)
// 	}
// 	retry()
// 	utils.Outf("registering SP1 proof\n")

// }

// func WriteToJson(obj txData, txID ids.ID) {
// 	fileName := "./data.json"

// 	// Read the existing data from the file
// 	data, err := os.ReadFile(fileName)
// 	if err != nil {
// 		log.Fatal("Error reading file:", err)
// 	}
// 	var jsonData map[string]interface{}
// 	if err := json.Unmarshal(data, &jsonData); err != nil {
// 		// Handle potential errors if the file is empty or invalid JSON
// 		jsonData = make(map[string]interface{})
// 	}
// 	jsonData[txID.String()] = obj
// 	newData, err := json.MarshalIndent(jsonData, "", "  ")
// 	if err != nil {
// 		log.Fatal("Error marshalling data:", err)
// 	}

// 	err = ioutil.WriteFile(fileName, newData, 0644)
// 	if err != nil {
// 		log.Fatal("Error writing to file:", err)
// 	}

// }
