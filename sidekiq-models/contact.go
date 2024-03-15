package model

import (
	"encoding/json"
	"io"
)

// Contact model
type Contact struct {
	ID               int        `json:"id" db:"id"`
	UserID           NullInt64  `json:"userID" db:"userID"`
	FirstName        string     `json:"firstName" db:"firstName"`
	LastName         string     `json:"lastName" db:"lastName"`
	Address          NullString `json:"address" db:"address"`
	City             NullString `json:"city" db:"city"`
	State            NullString `json:"state" db:"state"`
	Zip              NullString `json:"zip" db:"zip"`
	Country          NullString `json:"country" db:"country"`
	Phone            NullString `json:"phone" db:"phone"`
	Fax              NullString `json:"fax" db:"fax"`
	Email            NullString `json:"email" db:"email"`
	LastModifiedDate NullTime   `json:"lastModifiedDate" db:"lastModifiedDate"`
}

// ToJSON converts contact to json string
func (a *Contact) ToJSON() string {
	json, _ := json.Marshal(a)
	return string(json)
}

//WriteToJSON encode model directly to writer
func (a *Contact) WriteToJSON(w io.Writer) {
	json.NewEncoder(w).Encode(a)
}
