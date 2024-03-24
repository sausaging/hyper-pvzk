package handle

type VerifyRequestArgs struct {
	TxID       string `json:"tx_id"`
	VerifyType uint32 `json:"verify_type"`
}

type VerifyReplyArgs struct {
	IsSubmitted bool `json:"is_submitted"`
}
