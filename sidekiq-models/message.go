package model

import (
	"encoding/json"
	"io"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PrevMessage model
type PrevMessage struct {
	ID           int      `json:"id" db:"id"`
	DiscussionID int      `json:"discussionID" db:"discussionID"`
	UserID       int      `json:"userID" db:"userID"`
	Content      string   `json:"content" db:"content"`
	Timestamp    NullTime `json:"timestamp" db:"timestamp"`
}

// Chat model
type Chat struct {
	Id      primitive.ObjectID `json:"_id" bson:"_id"`
	BoardID primitive.ObjectID `json:"boardID" bson:"boardID"`
	Type    string             `json:"type" bson:"type"`
	GroupID primitive.ObjectID `json:"groupID" bson:"groupID"`
	// Receiver         []string           `json:"receiver" bson:"receiver"`
	Sender           string    `json:"sender" bson:"sender"`
	Message          string    `json:"message" bson:"message"`
	CreateDate       time.Time `json:"createDate" bson:"createDate"`
	LastModifiedDate time.Time `json:"lastModifiedDate" bson:"lastModifiedDate"`
	DeletedDate      time.Time `json:"deleteDate" bson:"deleteDate"`
	IsActive         bool      `json:"isActive" bson:"isActive"`
}

// MessageCollection collection to hold a list of messages
type MessageCollection struct {
	Size  int            `json:"size"`
	Index int            `json:"index"`
	Items []*PrevMessage `json:"items"`
}

// ToJSON converts message to json string
func (a *PrevMessage) ToJSON() string {
	json, _ := json.Marshal(a)
	return string(json)
}

// ToJSON converts discussion to json string
func (a *Chat) ToJSON() string {
	json, _ := json.Marshal(a)
	return string(json)
}

// ToJSON converts MessageCollection to json string
func (t *MessageCollection) ToJSON() string {
	json, _ := json.Marshal(t)
	return string(json)
}

// WriteToJSON encode model directly to writer
func (a *PrevMessage) WriteToJSON(w io.Writer) {
	json.NewEncoder(w).Encode(a)
}

// ReadMessageFromJSON create message from io.Reader
func ReadMessageFromJSON(data io.Reader) *PrevMessage {
	var message *PrevMessage
	err := json.NewDecoder(data).Decode(&message)
	if err != nil {
		return nil
	}
	return message
}

// MessageFromJSON create message from string
func MessageFromJSON(data string) *PrevMessage {
	var message *PrevMessage
	json.Unmarshal([]byte(data), &message)
	return message
}
