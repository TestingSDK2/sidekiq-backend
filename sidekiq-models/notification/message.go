package notification

import "encoding/json"

type Message struct {
	Type           string `json:"type"`
	Content        string `json:"content"`
	GroupID        string `json:"groupID,omitempty"`
	ListReceiverID []int  `json:"listReceiverID,omitempty"`
}

// ToJSON converts message to json string
func (a *Message) ToJSON() string {
	json, _ := json.Marshal(a)
	return string(json)
}
