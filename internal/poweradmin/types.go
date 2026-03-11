package poweradmin

// Zone represents a DNS zone in PowerAdmin.
type Zone struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Record represents a DNS record in PowerAdmin.
type Record struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
	Disabled int    `json:"disabled"`
}
