package actions

import (
	"io"
	"os"

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
