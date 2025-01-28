package compactor

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"sort"
	"strings"

	"github.com/grafana/dskit/middleware"

	"github.com/grafana/loki/v3/pkg/compactor/deletion"
	"github.com/grafana/loki/v3/pkg/util/handler"
)

//go:embed ui/dist
var uiFS embed.FS

func (c *Compactor) initUIFs() error {
	var err error
	c.uiFS, err = fs.Sub(uiFS, "dist")
	if err != nil {
		return err
	}
	return nil
}

func (c *Compactor) Handler() http.Handler {
	mux := http.NewServeMux()

	mw := middleware.Merge(
		middleware.AuthenticateUser,
		deletion.TenantMiddleware(c.limits),
		// Automatically parse form data
		middleware.Func(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseForm(); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				next.ServeHTTP(w, r)
			})
		}),
	)

	// API endpoints
	mux.Handle("/compactor/ring", c.ring)
	mux.Handle("/compactor/ui/api/ring", c.ring)
	mux.Handle("/compactor/ui/api/format_query", handler.NewFormatQuery())
	mux.HandleFunc("/compactor/ui/api/deletes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			c.handleListDeleteRequests(w, r)
		case http.MethodPost:
			mw.Wrap(http.HandlerFunc(c.DeleteRequestsHandler.AddDeleteRequestHandler)).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	fsHandler := http.FileServer(http.FS(c.uiFS))
	mux.Handle("/compactor/ui/", http.StripPrefix("/compactor/ui/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := c.uiFS.Open(strings.TrimPrefix(r.URL.Path, "/")); err != nil {
			r.URL.Path = "/"
		}
		fsHandler.ServeHTTP(w, r)
	})))

	return mux
}

type DeleteRequestResponse struct {
	RequestID    string `json:"request_id"`
	StartTime    int64  `json:"start_time"`
	EndTime      int64  `json:"end_time"`
	Query        string `json:"query"`
	Status       string `json:"status"`
	CreatedAt    int64  `json:"created_at"`
	UserID       string `json:"user_id"`
	DeletedLines int32  `json:"deleted_lines"`
}

func (c *Compactor) handleListDeleteRequests(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = string(deletion.StatusReceived)
	}

	ctx := r.Context()
	requests, err := c.deleteRequestsStore.GetDeleteRequestsByStatus(ctx, deletion.DeleteRequestStatus(status))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sort by creation time descending
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].CreatedAt > requests[j].CreatedAt
	})

	// Take only last 20
	if len(requests) > 20 {
		requests = requests[:20]
	}

	response := make([]DeleteRequestResponse, 0, len(requests))
	for _, req := range requests {
		response = append(response, DeleteRequestResponse{
			RequestID:    req.RequestID,
			StartTime:    int64(req.StartTime),
			EndTime:      int64(req.EndTime),
			Query:        req.Query,
			Status:       string(req.Status),
			CreatedAt:    int64(req.CreatedAt),
			UserID:       req.UserID,
			DeletedLines: req.DeletedLines,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
