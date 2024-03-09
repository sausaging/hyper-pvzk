package requester

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/ava-labs/hypersdk/chain"
	rpc "github.com/gorilla/rpc/v2/json2"
)

type EndpointRequester struct {
	Cli *http.Client
	Uri string
}

type PingReply struct {
	Success bool `json:"success"`
}

func New(uri string) *EndpointRequester {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 1_000
	t.MaxConnsPerHost = 1_000
	t.MaxIdleConnsPerHost = 1_000

	return &EndpointRequester{
		Cli: &http.Client{
			Timeout:   8 * time.Second,
			Transport: t,
		},
		Uri: uri,
	}
}

func GetRequesterInstance(rules chain.Rules) (*http.Client, string) {
	data, _ := rules.FetchCustom("")
	endpoint := data.(*EndpointRequester)
	return endpoint.Cli, endpoint.Uri
}

func NewRequest(endPoint string, data []byte) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, endPoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func Ping(client *EndpointRequester) (bool, error) {
	endPointUri := client.Uri + PINGENDPOINT
	req, err := NewRequest(endPointUri, []byte{})
	if err != nil {
		return false, fmt.Errorf("%s: can't request http", err)
	}
	resp, err := client.Cli.Do(req)
	if err != nil {
		return false, fmt.Errorf("%s: can't do request", err)
	}
	reply := new(PingReply)
	err = rpc.DecodeClientResponse(resp.Body, reply)
	if err != nil {
		return false, fmt.Errorf("%s: can't decode client response", err)
	}
	return reply.Success, nil
}
