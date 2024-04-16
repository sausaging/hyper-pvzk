package cmd

import (
	"context"
	"log"
	"os"

	"github.com/sausaging/hyper-pvzk/actions"
	"github.com/sausaging/hypersdk/chain"
	"github.com/sausaging/hypersdk/consts"
	"github.com/sausaging/hypersdk/utils"
	"github.com/spf13/cobra"
)

// var chunkSize = 10 * 1024

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
		ps, err := handler.Root().PromptInt("proving sytem: sp1 -> 1, miden -> 2, risc0 -> 3, gnark -> 4, jolt -> 5 ", consts.MaxInt)
		if err != nil {
			return err
		}
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}
		_, _, err = sendAndWait(ctx, nil, &actions.Register{
			ProovingSystem: uint64(ps),
		}, cli, bcli, ws, factory, true)
		return err
	},
}

var registerImageCmd = &cobra.Command{
	Use: "register-image",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, bcli, ws, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		imageID, err := handler.Root().PromptID("image id")
		if err != nil {
			return nil
		}
		valType, err := handler.Root().PromptInt("proof val type(1 for ELF, rest for proofs)", int(consts.MaxUint16))
		if err != nil {
			return err
		}
		rootHash, err := handler.Root().PromptString("root hash", 1, consts.MaxInt)
		if err != nil {
			return err
		}
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}
		_, _, err = sendAndWait(ctx, nil, &actions.RegisterImage{
			ImageID:  imageID,
			ValType:  uint64(valType),
			RootHash: rootHash,
		}, cli, bcli, ws, factory, true)
		return err
	},
}

var broadcastCmd = &cobra.Command{
	Use: "broadcast",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, _, cli, _, _, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		fileName, err := handler.Root().PromptString("file name", 1, consts.MaxInt)
		if err != nil {
			return err
		}
		// Read the existing data from the file
		data, err := os.ReadFile(fileName)
		if err != nil {
			log.Fatal("Error reading file:", err)
		}

		imageID, err := handler.Root().PromptID("image id")
		if err != nil {
			return nil
		}
		valType, err := handler.Root().PromptInt("proof val type(1 for ELF, rest for proofs)", int(consts.MaxUint16))
		if err != nil {
			return err
		}
		chunkIndex, err := handler.Root().PromptInt("chunk index", int(consts.MaxUint16))
		if err != nil {
			return err
		}
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// @todo we are naive broadcasting full file. change this to 100kib blob based broadcast
		cli.SubmitChunks(ctx, imageID, uint16(valType), uint16(chunkIndex), data)
		return nil
	},
}

// the data refered for get verification will be associated with this txID
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
		verifyType, err := handler.Root().PromptInt("verification type: 1 -> SP1, 2 -> Miden, 3 -> Risc0, 4 -> Gnark, 5 -> Jolt", 10)
		if err != nil {
			return err
		}
		timeOutBlocks, err := handler.Root().PromptInt("time out blocks", int(consts.MaxUint16))
		if err != nil {
			return err
		}
		var action chain.Action
		if verifyType == 1 {
			action = &actions.SP1{
				ImageID:       imageID,
				ProofValType:  uint64(valType),
				TimeOutBlocks: uint64(timeOutBlocks),
			}
		} else if verifyType == 2 {
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
				TimeOutBlocks:   uint64(timeOutBlocks),
			}

		} else if verifyType == 3 {
			riscZeroImageID, err := handler.Root().PromptString("risc zero image id", 1, consts.MaxInt)
			if err != nil {
				return err
			}

			action = &actions.RiscZero{
				ImageID:         imageID,
				ProofValType:    uint64(valType),
				RiscZeroImageID: riscZeroImageID,
				TimeOutBlocks:   uint64(timeOutBlocks),
			}
		} else if verifyType == 4 {

		} else if verifyType == 5 {
			action = &actions.Jolt{
				ImageID:       imageID,
				ProofValType:  uint64(valType),
				TimeOutBlocks: uint64(timeOutBlocks),
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

var verifyStatusCmd = &cobra.Command{
	Use: "verify-status",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, _, _, bcli, _, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		txID, err := handler.Root().PromptID("tx id of verify")
		if err != nil {
			return err
		}
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}
		status, err := bcli.VerifyStatus(ctx, txID)
		if err != nil {
			return err
		}
		utils.Outf("status: %s", status)
		return nil
	},
}
