package webhook

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Service captures incoming webhook requests and serves them back for viewing.
type Service struct {
	repo         Repository
	maxBodyBytes int64
	pageSize     int
	now          func() time.Time
}

// NewService wires a service onto a repository.
func NewService(repo Repository, maxBodyBytes int64, pageSize int) *Service {
	if pageSize <= 0 {
		pageSize = 20
	}
	return &Service{
		repo:         repo,
		maxBodyBytes: maxBodyBytes,
		pageSize:     pageSize,
		now:          func() time.Time { return time.Now().In(istLocation) },
	}
}

// PageSize is the number of entries served per page.
func (s *Service) PageSize() int { return s.pageSize }

// Record reads the request and appends it to the current day's log.
func (s *Service) Record(ctx context.Context, r *http.Request, clientIP string) (LogEntry, error) {
	// One byte over the limit tells us the body was cut short.
	body, err := io.ReadAll(io.LimitReader(r.Body, s.maxBodyBytes+1))
	if err != nil {
		return LogEntry{}, fmt.Errorf("read body: %w", err)
	}

	truncated := int64(len(body)) > s.maxBodyBytes
	if truncated {
		body = body[:s.maxBodyBytes]
	}

	entry := LogEntry{
		ID:        newID(),
		Timestamp: s.now(),
		Method:    r.Method,
		Path:      r.URL.Path,
		Query:     r.URL.RawQuery,
		RemoteIP:  clientIP,
		Headers:   r.Header.Clone(),
		BodySize:  len(body),
		Body:      string(body),
		Truncated: truncated,
	}

	if err := s.repo.Save(ctx, entry); err != nil {
		return LogEntry{}, err
	}
	return entry, nil
}

// Dates lists the days that have logs, newest first.
func (s *Service) Dates(ctx context.Context) ([]string, error) {
	return s.repo.Dates(ctx)
}

// List returns one page of a day's entries, newest first. An empty date falls
// back to today.
func (s *Service) List(ctx context.Context, date, search string, page int) (Page, error) {
	if date == "" {
		date = s.now().Format(dateLayout)
	}
	if page < 1 {
		page = 1
	}

	return s.repo.Query(ctx, QueryParams{
		Date:     date,
		Search:   search,
		Page:     page,
		PageSize: s.pageSize,
	})
}

// Today is the current log bucket.
func (s *Service) Today() string { return s.now().Format(dateLayout) }

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
