package poweradmin

import "fmt"

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
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Content  string   `json:"content"`
	TTL      int      `json:"ttl"`
	Priority int      `json:"priority"`
	Disabled FlexBool `json:"disabled"`
}
