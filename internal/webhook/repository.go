package webhook

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const (
	dateLayout = "2006-01-02"
	fileSuffix = ".jsonl"
)

// ErrInvalidDate is returned when a date is not a well formed YYYY-MM-DD value.
var ErrInvalidDate = errors.New("date must be in YYYY-MM-DD format")

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// QueryParams selects a page of entries from a single day's log.
type QueryParams struct {
	Date     string
	Search   string
	Page     int
	PageSize int
}

// Repository persists and retrieves webhook request logs.
type Repository interface {
	Save(ctx context.Context, entry LogEntry) error
	Dates(ctx context.Context) ([]string, error)
	Query(ctx context.Context, params QueryParams) (Page, error)
}

// FileRepository stores one newline delimited JSON file per day under Dir.
type FileRepository struct {
	dir     string
	maxLine int

	mu sync.Mutex // serialises appends so concurrent writes stay line aligned
}

// NewFileRepository creates the log directory if needed and returns a repository
// rooted at it. maxBodyBytes sizes the read buffer so long entries still scan.
func NewFileRepository(dir string, maxBodyBytes int64) (*FileRepository, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	return &FileRepository{
		dir:     dir,
		maxLine: int(maxBodyBytes) + 64<<10,
	}, nil
}

// Save appends the entry to the file for the day it was received.
func (r *FileRepository) Save(_ context.Context, entry LogEntry) error {
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode entry: %w", err)
	}
	line = append(line, '\n')

	path, err := r.pathFor(entry.Date())
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("write entry: %w", err)
	}
	return nil
}

// Dates lists the days that have logs, newest first.
func (r *FileRepository) Dates(_ context.Context) ([]string, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read log dir: %w", err)
	}

	dates := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), fileSuffix)
		if name == e.Name() || !datePattern.MatchString(name) {
			continue
		}
		dates = append(dates, name)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	return dates, nil
}

// Query returns one page of the given day's entries, newest first, keeping only
// the entries that match the search term.
func (r *FileRepository) Query(_ context.Context, params QueryParams) (Page, error) {
	page := Page{
		Date:     params.Date,
		Search:   params.Search,
		Page:     params.Page,
		PageSize: params.PageSize,
		Entries:  []LogEntry{},
	}

	matched, err := r.load(params.Date, params.Search)
	if err != nil {
		return page, err
	}

	page.TotalCount = len(matched)
	page.TotalPages = (page.TotalCount + params.PageSize - 1) / params.PageSize
	if page.TotalPages == 0 {
		page.TotalPages = 1
	}
	if page.Page > page.TotalPages {
		page.Page = page.TotalPages
	}

	// Entries are appended chronologically, so the newest page is the tail.
	end := page.TotalCount - (page.Page-1)*params.PageSize
	start := end - params.PageSize
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}

	for i := end - 1; i >= start; i-- {
		page.Entries = append(page.Entries, matched[i])
	}

	return page, nil
}

// load reads a day's file and returns the entries matching search, oldest first.
func (r *FileRepository) load(date, search string) ([]LogEntry, error) {
	path, err := r.pathFor(date)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	needle := strings.ToLower(strings.TrimSpace(search))

	var matched []LogEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64<<10), r.maxLine)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// A partially written or corrupt line should not hide the rest.
			continue
		}
		if needle != "" && !entry.matches(needle) {
			continue
		}
		matched = append(matched, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan log file: %w", err)
	}

	return matched, nil
}

// pathFor maps a date to its log file, rejecting anything that is not a plain
// YYYY-MM-DD value so the date can never escape the log directory.
func (r *FileRepository) pathFor(date string) (string, error) {
	if !datePattern.MatchString(date) {
		return "", ErrInvalidDate
	}
	return filepath.Join(r.dir, date+fileSuffix), nil
}

// matches reports whether needle (already lowercased) appears anywhere in the
// parts of the entry that are worth searching.
func (e LogEntry) matches(needle string) bool {
	fields := []string{e.ID, e.Method, e.Path, e.Query, e.RemoteIP, e.Body}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), needle) {
			return true
		}
	}
	for name, values := range e.Headers {
		if strings.Contains(strings.ToLower(name), needle) {
			return true
		}
		for _, v := range values {
			if strings.Contains(strings.ToLower(v), needle) {
				return true
			}
		}
	}
	return false
}
