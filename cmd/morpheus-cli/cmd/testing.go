package cmd

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/sausaging/hyper-pvzk/actions"
	"github.com/spf13/cobra"
)

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
		maxChunks, err := handler.Root().PromptInt("max chunks(feature not yet enabled)", int(consts.MaxUint16))
		if err != nil {
			return err
		}
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}
		_, _, err = sendAndWait(ctx, nil, &actions.Register{
			MaxChunks: uint64(maxChunks),
		}, cli, bcli, ws, factory, true)
		return err
	},
}

var deployCmd = &cobra.Command{
	Use: "deploy",
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
		chunkSize := 100 * 1024
		for i := 0; i < len(code); i += chunkSize {
			end := i + chunkSize
			if end > len(code) {
				end = len(code)
			}
			_, err = send(ctx, nil, &actions.Deploy{
				ImageID:      imageID,
				ProofvalType: uint64(valType),
				Data:         code[i:end],
			}, cli, bcli, ws, factory)
			if (i/chunkSize)%10 == 0 && i != 0 {
				time.Sleep(30 * time.Second)
			}
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
			// action = &actions. //@todo
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
