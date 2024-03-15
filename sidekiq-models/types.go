package model

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// NullTime - an alias for mysql.NullTime data type
type NullTime mysql.NullTime

// NullInt64 - an alias for sql.NullInt64 data type
type NullInt64 sql.NullInt64

// NullString - an alias for sql.NullString data type
type NullString sql.NullString

// IDList - used to hold and convert an csv string of IDs
type IDList []int

// Scan implements the Scanner interface for NullTime
func (nt *NullTime) Scan(value interface{}) error {
	var t mysql.NullTime
	if err := t.Scan(value); err != nil {
		return err
	}

	// if nil then make Valid false
	if reflect.TypeOf(value) == nil {
		*nt = NullTime{t.Time, false}
	} else {
		*nt = NullTime{t.Time, true}
	}

	return nil
}

// Value implements the Valuer interface for NullTime
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return driver.Value(nil), nil
	}
	return driver.Value(nt.Time), nil
}

// MarshalJSON for NullTime
func (nt *NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	val := fmt.Sprintf("\"%s\"", nt.Time.Format(time.RFC3339))
	return []byte(val), nil
}

// UnmarshalJSON for NullTime
func (nt *NullTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		nt.Valid = false
		return nil
	}

	s = strings.ReplaceAll(s, "\"", "")
	x, err := time.Parse(time.RFC3339, s)
	if err != nil {
		nt.Valid = false
		return err
	}

	nt.Time = x
	nt.Valid = true
	return nil
}

// Scan implements the Scanner interface for NullInt64
func (ni *NullInt64) Scan(value interface{}) error {
	var t sql.NullInt64
	if err := t.Scan(value); err != nil {
		return err
	}

	// if nil then make Valid false
	if reflect.TypeOf(value) == nil {
		*ni = NullInt64{t.Int64, false}
	} else {
		*ni = NullInt64{t.Int64, true}
	}

	return nil
}

// Value implements the Valuer interface for NullInt64
func (ni *NullInt64) Value() (driver.Value, error) {
	if !ni.Valid {
		return driver.Value(nil), nil
	}
	return driver.Value(ni.Int64), nil
}

// MarshalJSON for NullInt64
func (ni *NullInt64) MarshalJSON() ([]byte, error) {
	if !ni.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ni.Int64)
}

// UnmarshalJSON for NullTime
func (ni *NullInt64) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &ni.Int64)
	ni.Valid = (err == nil)
	return err
}

// Value implements the Valuer interface for NullString
func (ns *NullString) Value() (driver.Value, error) {
	if !ns.Valid {
		return driver.Value(nil), nil
	}
	return driver.Value(ns.String), nil
}

// Scan implements the Scanner interface for NullString
func (ns *NullString) Scan(value interface{}) error {
	var s sql.NullString
	if err := s.Scan(value); err != nil {
		return err
	}

	// if nil then make Valid false
	if reflect.TypeOf(value) == nil {
		*ns = NullString{s.String, false}
	} else {
		*ns = NullString{s.String, true}
	}

	return nil
}

// MarshalJSON for NullString
func (ns *NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}

// UnmarshalJSON for NullString
func (ns *NullString) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &ns.String)
	ns.Valid = (err == nil)
	return err
}

// Scan implements the Scanner interface for NullTime
func (idl *IDList) Scan(value interface{}) error {
	var s sql.NullString
	if err := s.Scan(value); err != nil {
		return err
	}

	// if nil then make Valid false
	if reflect.TypeOf(value) == nil {
		*idl = IDList{}
	} else {
		members := IDList{}
		vals := strings.Split(s.String, ",")
		for i := 0; i < len(vals); i++ {
			iVal, err := strconv.Atoi(vals[i])
			if err == nil {
				members = append(members, iVal)
			}
		}
		*idl = members
	}

	return nil
}
