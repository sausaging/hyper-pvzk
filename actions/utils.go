package actions

import (
	"io"
	"os"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/hypersdk/consts"
	rpc "github.com/gorilla/rpc/v2/json2"
)

func WriteFile(filePath string, data []byte) error {
	// Write the data to the file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close() // Ensure file is closed even if errors occur
	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func DecodeClientResponse(r io.Reader, reply interface{}) (interface{}, error) {
	err := rpc.DecodeClientResponse(r, reply)
	return reply, err
}

func GetMessage(txID ids.ID, vote bool) (msg []byte) {
	msg = make([]byte, consts.IDLen+consts.BoolLen)
	copy(msg[:], txID[:])
	if vote {
		copy(msg[consts.IDLen:], []byte{1})
	} else {
		copy(msg[consts.IDLen:], []byte{0})
	}
	return msg
}
