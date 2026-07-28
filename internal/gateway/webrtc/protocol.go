package webrtc

import (
	"encoding/base64"
	"encoding/json"
)

type RawURLBytes []byte

type Request struct {
	ID string `json:"id"`

	Method string `json:"method"`
	Path   string `json:"path"`

	Headers map[string]string `json:"headers,omitempty"`

	Body RawURLBytes `json:"body,omitempty"`
}

type Response struct {
	ID string `json:"id"`

	// Status int `json:"status"`

	// Headers map[string]string `json:"headers,omitempty"`

	Body RawURLBytes `json:"body,omitempty"`
}

func (b RawURLBytes) MarshalJSON() ([]byte, error) {
	encoded := base64.RawURLEncoding.EncodeToString(b)

	return json.Marshal(encoded)
}

func (b *RawURLBytes) UnmarshalJSON(data []byte) error {
	var s string

	err := json.Unmarshal(data, &s)

	if err != nil {
		return err
	}

	decoded, err := base64.RawURLEncoding.DecodeString(s)

	if err != nil {
		return err
	}

	*b = decoded

	return nil
}
