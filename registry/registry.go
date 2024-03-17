// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package registry

import (
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"

	"github.com/sausaging/hyper-pvzk/actions"
	"github.com/sausaging/hyper-pvzk/auth"
	"github.com/sausaging/hyper-pvzk/consts"
)

// Setup types
func init() {
	consts.ActionRegistry = codec.NewTypeParser[chain.Action, *warp.Message]()
	consts.AuthRegistry = codec.NewTypeParser[chain.Auth, *warp.Message]()

	errs := &wrappers.Errs{}
	errs.Add(
		// When registering new actions, ALWAYS make sure to append at the end.
		consts.ActionRegistry.Register((&actions.Transfer{}).GetTypeID(), actions.UnmarshalTransfer, false),
		consts.ActionRegistry.Register((&actions.Miden{}).GetTypeID(), actions.UnmarshalMiden, false),
		consts.ActionRegistry.Register((&actions.SP1{}).GetTypeID(), actions.UnmarshalSP1, false),
		consts.ActionRegistry.Register((&actions.Register{}).GetTypeID(), actions.UnmarshalRegister, false),
		consts.ActionRegistry.Register((&actions.Deploy{}).GetTypeID(), actions.UnmarshalDeploy, false),
		consts.ActionRegistry.Register((&actions.RiscZero{}).GetTypeID(), actions.UnmarshalRiscZero, false),
		consts.ActionRegistry.Register((&actions.Gnark{}).GetTypeID(), actions.UnmarshalGnark, false),
		// When registering new auth, ALWAYS make sure to append at the end.
		consts.AuthRegistry.Register((&auth.ED25519{}).GetTypeID(), auth.UnmarshalED25519, false),
		consts.AuthRegistry.Register((&auth.SECP256R1{}).GetTypeID(), auth.UnmarshalSECP256R1, false),
		consts.AuthRegistry.Register((&auth.BLS{}).GetTypeID(), auth.UnmarshalBLS, false),
	)
	if errs.Errored() {
		panic(errs.Err)
	}
}
