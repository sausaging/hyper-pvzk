package requester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ava-labs/hypersdk/chain"
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
	req, err := http.NewRequest(http.MethodGet, endPointUri, bytes.NewBuffer([]byte{}))
	if err != nil {
		return false, fmt.Errorf("%s: can't request http", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Cli.Do(req)
	if err != nil {
		return false, fmt.Errorf("%s: can't do request", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("%s: can't decode client response", err)
	}
	reply := new(PingReply)
	err = json.Unmarshal(body, &reply)
	if err != nil {
		return false, fmt.Errorf("%s: can't unmarshal json", err)
	}
	return reply.Success, nil
}
