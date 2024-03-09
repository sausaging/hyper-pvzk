package actions

import (
	"io"
	"os"

	rpc "github.com/gorilla/rpc/v2/json2"
)

func WriteFile(filePath string, data []byte) error {
	// Write the data to the file
	err := os.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func DecodeClientResponse(r io.Reader, reply interface{}) (interface{}, error) {
	err := rpc.DecodeClientResponse(r, reply)
	return reply, err
}
