package poweradmin

import (
	"encoding/json"
	"fmt"
)

// RecordID identifies a DNS record. SQL-backed PowerAdmin instances return
// numeric IDs while the PowerDNS API backend returns encoded string IDs, so
// it unmarshals from both JSON numbers and strings.
type RecordID string

func (r *RecordID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*r = RecordID(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*r = RecordID(n.String())
		return nil
	}
	return fmt.Errorf("RecordID: cannot unmarshal %s", string(data))
}

// FlexBool is a bool type that can unmarshal from both JSON booleans and integers (0/1).
// PowerAdmin API returns the disabled field as bool in some versions and int in others.
type FlexBool bool

func (b *FlexBool) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "true", "1":
		*b = true
	case "false", "0":
		*b = false
	default:
		return fmt.Errorf("FlexBool: cannot unmarshal %s", string(data))
	}
	return nil
}

func (b FlexBool) MarshalJSON() ([]byte, error) {
	if b {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}

// Zone represents a DNS zone in PowerAdmin.
type Zone struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Record represents a DNS record in PowerAdmin.
type Record struct {
	ID       RecordID `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Content  string   `json:"content"`
	TTL      int      `json:"ttl"`
	Priority int      `json:"priority"`
	Disabled FlexBool `json:"disabled"`
}
