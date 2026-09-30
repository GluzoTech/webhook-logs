package webhook

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*.html
var templateFS embed.FS

// Handler exposes the webhook logger over HTTP.
type Handler struct {
	svc  *Service
	tmpl *template.Template
}

// NewHandler parses the viewer template and returns a handler.
func NewHandler(svc *Service) (*Handler, error) {
	tmpl, err := template.New("viewer").Funcs(templateFuncs()).
		ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Handler{svc: svc, tmpl: tmpl}, nil
}

// Log records an incoming webhook request.
// POST /api/v1/webhook/log
func (h *Handler) Log(c *gin.Context) {
	entry, err := h.svc.Record(c.Request.Context(), c.Request, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not record request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "logged",
		"id":        entry.ID,
		"date":      entry.Date(),
		"timestamp": entry.Timestamp.Format(time.RFC3339),
	})
}

// viewData is the model behind the viewer template.
type viewData struct {
	Page    Page
	Dates   []string
	Key     string
	Today   string
	PrevURL string
	NextURL string
	Error   string
}

// View renders the paginated log viewer.
// GET /api/v1/webhook/view?key=...&date=YYYY-MM-DD&q=...&page=1
func (h *Handler) View(c *gin.Context) {
	key := c.Query("key")
	date := c.Query("date")
	search := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	if date == "" {
		date = h.svc.Today()
	}

	data := viewData{Key: key, Today: h.svc.Today()}

	if dates, err := h.svc.Dates(c.Request.Context()); err == nil {
		data.Dates = dates
	}

	result, err := h.svc.List(c.Request.Context(), date, search, page)
	if err != nil {
		if errors.Is(err, ErrInvalidDate) {
			data.Error = ErrInvalidDate.Error()
			data.Page = Page{Date: date, Search: search, Page: 1, PageSize: h.svc.PageSize(), TotalPages: 1}
		} else {
			c.String(http.StatusInternalServerError, "could not read logs")
			return
		}
	} else {
		data.Page = result
	}

	data.PrevURL = viewURL(key, data.Page.Date, search, data.Page.Page-1)
	data.NextURL = viewURL(key, data.Page.Date, search, data.Page.Page+1)

	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(c.Writer, "viewer.html", data); err != nil {
		c.Error(err) //nolint:errcheck // response already started
	}
}

// viewURL builds a link back to the viewer keeping the current filters.
func viewURL(key, date, search string, page int) string {
	q := url.Values{}
	q.Set("key", key)
	q.Set("date", date)
	if search != "" {
		q.Set("q", search)
	}
	q.Set("page", strconv.Itoa(page))
	return "/api/v1/webhook/view?" + q.Encode()
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		// prettyBody indents the body when it is JSON, otherwise returns it as is.
		// json.Indent reformats the raw bytes rather than re-encoding them, so
		// characters like & < > survive as themselves; the template escapes them.
		"prettyBody": func(body string) string {
			if !json.Valid([]byte(body)) {
				return body
			}
			var buf bytes.Buffer
			if err := json.Indent(&buf, []byte(body), "", "  "); err != nil {
				return body
			}
			return buf.String()
		},
		"localTime": func(t time.Time) string {
			return t.Local().Format("15:04:05.000")
		},
		"fullTime": func(t time.Time) string {
			return t.Local().Format("2006-01-02 15:04:05 MST")
		},
		"sortedHeaders": func(h map[string][]string) []string {
			keys := make([]string, 0, len(h))
			for k := range h {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			return keys
		},
		"joinValues": func(v []string) string {
			out := ""
			for i, s := range v {
				if i > 0 {
					out += ", "
				}
				out += s
			}
			return out
		},
		"add": func(a, b int) int { return a + b },
	}
}
