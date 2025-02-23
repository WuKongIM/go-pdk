package pdk

import "encoding/json"

type Payload interface {
	Encode() ([]byte, error)
	Decode([]byte) error
}

type PayloadText struct {
	Content string `json:"content"`
	Type    int    `json:"type"`
}

func (p *PayloadText) Encode() ([]byte, error) {
	return json.Marshal(p)
}

func (p *PayloadText) Decode(data []byte) error {
	return json.Unmarshal(data, p)
}
