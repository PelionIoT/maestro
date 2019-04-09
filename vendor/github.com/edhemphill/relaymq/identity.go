package relaymq

import (
	"encoding/json"
	"net/http"
)

type IdentityHeader struct {
	ClientType string `json:"clientType"`
	RelayID string `json:"relayID"`
	AccountID string `json:"accountID"`
}

func (identity *IdentityHeader) Header() http.Header {
	if identity.ClientType == "" {
		return http.Header{ }
	}

	header := http.Header{ }
	encoded, _ := json.Marshal(identity)

	header.Add("X-WigWag-Identity", string(encoded))

	return header
}

func (identity *IdentityHeader) FromHeader(header http.Header) error {
	err := json.Unmarshal([]byte(header.Get("X-WigWag-Identity")), identity)

	if err != nil {
		return err
	}

	return nil
}
