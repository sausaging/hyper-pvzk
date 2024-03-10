// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	smath "github.com/ava-labs/avalanchego/utils/math"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/fees"
	"github.com/ava-labs/hypersdk/state"

	mconsts "github.com/sausaging/hyper-pvzk/consts"
)

type ReadState func(context.Context, [][]byte) ([][]byte, []error)

// Metadata
// 0x0/ (tx)
//   -> [txID] => timestamp
//
// State
// / (height) => store in root
//   -> [heightPrefix] => height
// 0x0/ (balance)
//   -> [owner] => balance
// 0x1/ (hypersdk-height)
// 0x2/ (hypersdk-timestamp)
// 0x3/ (hypersdk-fee)
// 0x4/ (hypersdk-incoming warp)
// 0x5/ (hypersdk-outgoing warp)

const (
	// metaDB
	txPrefix = 0x0

	// stateDB
	balancePrefix      = 0x0
	heightPrefix       = 0x1
	timestampPrefix    = 0x2
	feePrefix          = 0x3
	incomingWarpPrefix = 0x4
	outgoingWarpPrefix = 0x5
	registerPrefix     = 0x6
	deployPrefix       = 0x7
)

const BalanceChunks uint16 = 1

// const registerChunks uint16 = consts.MaxUint16

var (
	failureByte  = byte(0x0)
	successByte  = byte(0x1)
	heightKey    = []byte{heightPrefix}
	timestampKey = []byte{timestampPrefix}
	feeKey       = []byte{feePrefix}
)

// [txPrefix] + [txID]
func TxKey(id ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen)
	k[0] = txPrefix
	copy(k[1:], id[:])
	return
}

func StoreTransaction(
	_ context.Context,
	db database.KeyValueWriter,
	id ids.ID,
	t int64,
	success bool,
	units fees.Dimensions,
	fee uint64,
) error {
	k := TxKey(id)
	v := make([]byte, consts.Uint64Len+1+fees.DimensionsLen+consts.Uint64Len)
	binary.BigEndian.PutUint64(v, uint64(t))
	if success {
		v[consts.Uint64Len] = successByte
	} else {
		v[consts.Uint64Len] = failureByte
	}
	copy(v[consts.Uint64Len+1:], units.Bytes())
	binary.BigEndian.PutUint64(v[consts.Uint64Len+1+fees.DimensionsLen:], fee)
	return db.Put(k, v)
}

func GetTransaction(
	_ context.Context,
	db database.KeyValueReader,
	id ids.ID,
) (bool, int64, bool, fees.Dimensions, uint64, error) {
	k := TxKey(id)
	v, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return false, 0, false, fees.Dimensions{}, 0, nil
	}
	if err != nil {
		return false, 0, false, fees.Dimensions{}, 0, err
	}
	t := int64(binary.BigEndian.Uint64(v))
	success := true
	if v[consts.Uint64Len] == failureByte {
		success = false
	}
	d, err := fees.UnpackDimensions(v[consts.Uint64Len+1 : consts.Uint64Len+1+fees.DimensionsLen])
	if err != nil {
		return false, 0, false, fees.Dimensions{}, 0, err
	}
	fee := binary.BigEndian.Uint64(v[consts.Uint64Len+1+fees.DimensionsLen:])
	return true, t, success, d, fee, nil
}

// [balancePrefix] + [address]
func BalanceKey(addr codec.Address) (k []byte) {
	k = make([]byte, 1+codec.AddressLen+consts.Uint16Len)
	k[0] = balancePrefix
	copy(k[1:], addr[:])
	binary.BigEndian.PutUint16(k[1+codec.AddressLen:], BalanceChunks)
	return
}

// If locked is 0, then account does not exist
func GetBalance(
	ctx context.Context,
	im state.Immutable,
	addr codec.Address,
) (uint64, error) {
	_, bal, _, err := getBalance(ctx, im, addr)
	return bal, err
}

func getBalance(
	ctx context.Context,
	im state.Immutable,
	addr codec.Address,
) ([]byte, uint64, bool, error) {
	k := BalanceKey(addr)
	bal, exists, err := innerGetBalance(im.GetValue(ctx, k))
	return k, bal, exists, err
}

// Used to serve RPC queries
func GetBalanceFromState(
	ctx context.Context,
	f ReadState,
	addr codec.Address,
) (uint64, error) {
	k := BalanceKey(addr)
	values, errs := f(ctx, [][]byte{k})
	bal, _, err := innerGetBalance(values[0], errs[0])
	return bal, err
}

func innerGetBalance(
	v []byte,
	err error,
) (uint64, bool, error) {
	if errors.Is(err, database.ErrNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return binary.BigEndian.Uint64(v), true, nil
}

func SetBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	balance uint64,
) error {
	k := BalanceKey(addr)
	return setBalance(ctx, mu, k, balance)
}

func setBalance(
	ctx context.Context,
	mu state.Mutable,
	key []byte,
	balance uint64,
) error {
	return mu.Insert(ctx, key, binary.BigEndian.AppendUint64(nil, balance))
}

func AddBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	amount uint64,
	create bool,
) error {
	key, bal, exists, err := getBalance(ctx, mu, addr)
	if err != nil {
		return err
	}
	// Don't add balance if account doesn't exist. This
	// can be useful when processing fee refunds.
	if !exists && !create {
		return nil
	}
	nbal, err := smath.Add64(bal, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not add balance (bal=%d, addr=%v, amount=%d)",
			ErrInvalidBalance,
			bal,
			codec.MustAddressBech32(mconsts.HRP, addr),
			amount,
		)
	}
	return setBalance(ctx, mu, key, nbal)
}

func SubBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	amount uint64,
) error {
	key, bal, _, err := getBalance(ctx, mu, addr)
	if err != nil {
		return err
	}
	nbal, err := smath.Sub(bal, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not subtract balance (bal=%d, addr=%v, amount=%d)",
			ErrInvalidBalance,
			bal,
			codec.MustAddressBech32(mconsts.HRP, addr),
			amount,
		)
	}
	if nbal == 0 {
		// If there is no balance left, we should delete the record instead of
		// setting it to 0.
		return mu.Remove(ctx, key)
	}
	return setBalance(ctx, mu, key, nbal)
}

func HeightKey() (k []byte) {
	return heightKey
}

func TimestampKey() (k []byte) {
	return timestampKey
}

func FeeKey() (k []byte) {
	return feeKey
}

func IncomingWarpKeyPrefix(sourceChainID ids.ID, msgID ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen*2)
	k[0] = incomingWarpPrefix
	copy(k[1:], sourceChainID[:])
	copy(k[1+consts.IDLen:], msgID[:])
	return k
}

func OutgoingWarpKeyPrefix(txID ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen)
	k[0] = outgoingWarpPrefix
	copy(k[1:], txID[:])
	return k
}

func ChunkKey(imageID ids.ID, registerType uint16) (k []byte) {
	k = make([]byte, 1+consts.IDLen+consts.Uint16Len)
	k[0] = registerPrefix
	copy(k[1:], imageID[:])
	binary.BigEndian.PutUint16(k[1+consts.IDLen:], registerType)
	return k
}

func StoreRegistration(
	ctx context.Context,
	mu state.Mutable,
	imageID ids.ID,
	registerType uint16,
	chunkSize uint16,
	totalBytes uint64,
) error {
	key := ChunkKey(imageID, registerType)
	data := binary.BigEndian.AppendUint16([]byte{}, chunkSize)
	data = binary.BigEndian.AppendUint64(data, totalBytes)
	return mu.Insert(ctx, key, data)
}

func GetRegistration(
	ctx context.Context,
	im state.Immutable,
	imageID ids.ID,
	valType uint16,
) (uint16, uint64, error) {
	key := ChunkKey(imageID, valType)
	data, err := im.GetValue(ctx, key)
	if err != nil {
		return 0, 0, err
	}
	chunkSize := binary.BigEndian.Uint16(data[:consts.Uint16Len])
	totalBytes := binary.BigEndian.Uint64(data[consts.Uint16Len:])
	return chunkSize, totalBytes, nil
}

func DeployKey(txID ids.ID, valType uint16) (k []byte) {
	k = make([]byte, 1+consts.IDLen+consts.Uint16Len+consts.Uint16Len)
	k[0] = deployPrefix
	copy(k[1:], txID[:])
	binary.BigEndian.PutUint16(k[1+consts.IDLen:], valType)
	binary.BigEndian.PutUint16(k[1+consts.IDLen+consts.Uint16Len:], consts.MaxUint16)
	return k
}

func StoreDeployType(
	ctx context.Context,
	mu state.Mutable,
	imageID ids.ID,
	valType uint16,
	data []byte,
	chunkIndex uint16,
	chunkSize uint16,
) error {
	k := DeployKey(imageID, valType)
	val, err := mu.GetValue(ctx, k)
	if err != nil {
		return err
	}
	var newVal []byte
	start := uint64(chunkIndex) * uint64(chunkSize)
	if start+uint64(chunkSize) > uint64(len(val)) { // end case
		newVal = append(val[:start], data...) // data len not matches the free end limit, throws an error
	} else {
		newVal = append(val[:start], append(data, val[start+uint64(chunkSize):]...)...)
	}
	return mu.Insert(ctx, k, newVal)
}

func InitiateDeployType(
	ctx context.Context,
	mu state.Mutable,
	imageID ids.ID,
	valType uint16,
	initiationBytes []byte,
) error {
	k := DeployKey(imageID, valType)
	return mu.Insert(ctx, k, initiationBytes)
}

func GetDeployType(
	ctx context.Context,
	im state.Immutable,
	imageID ids.ID,
	valType uint16,
) ([]byte, error) {
	k := DeployKey(imageID, valType)
	return im.GetValue(ctx, k)
}
