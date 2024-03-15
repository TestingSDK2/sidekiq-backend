package model

import (
	"encoding/json"
	"io"
)

// UserGroup - group of users
type UserGroup struct {
	ID               int        `json:"id" db:"id"`
	DiscussionID     int        `json:"discussionID" db:"discussionID"`
	Name             string     `json:"name" db:"name"`
	LastModifiedDate NullTime   `json:"lastModifiedDate" db:"lastModifiedDate"`
	Users            []*Account `json:"users"`
}

// ToJSON converts user to json string
func (u *UserGroup) ToJSON() string {
	json, _ := json.Marshal(u)
	return string(json)
}

//WriteToJSON encode model directly to writer
func (u *UserGroup) WriteToJSON(w io.Writer) {
	json.NewEncoder(w).Encode(u)
}
