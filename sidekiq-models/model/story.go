package model

import "time"

type Story struct {
	PublicStartDate time.Time `json:"publicStartDate" bson:"publicStartDate"`
	PublicEndDate   time.Time `json:"publicEndDate" bson:"publicEndDate"`
}
