package model

import (
	"encoding/json"
	"io"
)

// Discussion model
type Discussion struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	IsPublic    bool   `json:"isPublic" db:"isPublic"`
	CreatedByID int    `json:"createdByID" db:"createdByID"`
	Members     IDList `json:"members" db:"members"`
}

// ToJSON converts discussion to json string
func (a *Discussion) ToJSON() string {
	json, _ := json.Marshal(a)
	return string(json)
}

//WriteToJSON encode model directly to writer
func (a *Discussion) WriteToJSON(w io.Writer) {
	json.NewEncoder(w).Encode(a)
}

// ReadDiscussionFromJSON create discussion from io.Reader
func ReadDiscussionFromJSON(data io.Reader) *Discussion {
	var discussion *Discussion
	err := json.NewDecoder(data).Decode(&discussion)
	if err != nil {
		return nil
	}
	return discussion
}

// DiscussionFromJSON create discussion from string
func DiscussionFromJSON(data string) *Discussion {
	var discussion *Discussion
	json.Unmarshal([]byte(data), &discussion)
	return discussion
}
