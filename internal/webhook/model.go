package webhook

import "time"

// LogEntry is a single captured webhook request. It is persisted as one JSON
// object per line in the file for the day the request arrived.
type LogEntry struct {
	ID        string              `json:"id"`
	Timestamp time.Time           `json:"timestamp"`
	Method    string              `json:"method"`
	Path      string              `json:"path"`
	Query     string              `json:"query,omitempty"`
	RemoteIP  string              `json:"remote_ip"`
	Headers   map[string][]string `json:"headers"`
	BodySize  int                 `json:"body_size"`
	Body      string              `json:"body"`
	Truncated bool                `json:"truncated,omitempty"`
}

// Date returns the log bucket the entry belongs to, as YYYY-MM-DD.
func (e LogEntry) Date() string {
	return e.Timestamp.Format(dateLayout)
}

// Page is one slice of query results, newest entry first.
type Page struct {
	Date       string
	Search     string
	Entries    []LogEntry
	Page       int
	PageSize   int
	TotalCount int
	TotalPages int
}

// HasPrev reports whether a newer page exists.
func (p Page) HasPrev() bool { return p.Page > 1 }

// HasNext reports whether an older page exists.
func (p Page) HasNext() bool { return p.Page < p.TotalPages }
