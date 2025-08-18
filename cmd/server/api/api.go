package api

import (
	"cloudsave/pkg/data"
	"cloudsave/pkg/repository"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type (
	HTTPServer struct {
		Server       *http.Server
		Service      *data.Service
		documentRoot string
	}
)

// NewServer start the http server
func NewServer(documentRoot string, srv *data.Service, creds map[string]string, port int) *HTTPServer {
	if !filepath.IsAbs(documentRoot) {
		panic("the document root is not an absolute path")
	}
	s := &HTTPServer{
		Service:      srv,
		documentRoot: documentRoot,
	}
	router := chi.NewRouter()
	router.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		notFound("id not found", writer, request)
	})
	router.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		methodNotAllowed(writer, request)
	})
	router.Use(middleware.Logger)
	router.Use(recoverMiddleware)
	router.Use(middleware.GetHead)
	router.Use(middleware.Compress(5, "application/gzip"))
	router.Use(middleware.Heartbeat("/heartbeat"))
	router.Route("/api", func(routerAPI chi.Router) {
		routerAPI.Use(BasicAuth("cloudsave", creds))
		routerAPI.Route("/v1", func(r chi.Router) {
			// Get information about the server
			r.Get("/version", s.Information)
			// Secured routes
			r.Group(func(secureRouter chi.Router) {
				// Save files routes
				secureRouter.Route("/games", func(gamesRouter chi.Router) {
					// List all available saves
					gamesRouter.Get("/", s.all)
					// Data routes
					gamesRouter.Group(func(saveRouter chi.Router) {
						saveRouter.Post("/{id}/data", s.upload)
						saveRouter.Get("/{id}/data", s.download)
						saveRouter.Get("/{id}/metadata", s.metadata)

						saveRouter.Get("/{id}/hist", s.allHist)
						saveRouter.Post("/{id}/hist/{uuid}/data", s.histUpload)
						saveRouter.Get("/{id}/hist/{uuid}/data", s.histDownload)
						saveRouter.Get("/{id}/hist/{uuid}/info", s.histExists)
					})
				})
			})
		})
	})
	s.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}
	return s
}

func (s HTTPServer) all(w http.ResponseWriter, r *http.Request) {
	datastore, err := s.Service.AllGames()
	if err != nil {
		slog.Error(err.Error())
		internalServerError(w, r)
		return
	}

	ok(datastore, w, r)
}

func (s HTTPServer) download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	path := filepath.Clean(filepath.Join(s.documentRoot, "data", id))

	fi, err := os.Stat(filepath.Join(path, "data.tar.gz"))
	if err != nil {
		notFound("id not found", w, r)
		return
	}

	f, err := s.Service.Repository().ReadBlob(repository.NewGameIdentifier(id))
	if err != nil {
		slog.Error(err.Error())
		internalServerError(w, r)
		return
	}
	defer f.Close()

	// Set headers
	w.Header().Set("Content-Disposition", "attachment; filename=\"data.tar.gz\"")
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(200)

	// Stream the file content
	http.ServeContent(w, r, "data.tar.gz", fi.ModTime(), f)
}

func (s HTTPServer) upload(w http.ResponseWriter, r *http.Request) {
	const (
		sizeLimit int64 = 500 << 20 // 500 MB
	)

	id := chi.URLParam(r, "id")

	// Limit max upload size
	r.Body = http.MaxBytesReader(w, r.Body, sizeLimit)

	// Parse multipart form
	err := r.ParseMultipartForm(sizeLimit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to load payload:", err)
		badRequest("bad payload", w, r)
		return
	}

	m, err := parseFormMetadata(id, r.MultipartForm.Value)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot find metadata in the form:", err)
		badRequest("metadata not found", w, r)
		return
	}

	// Retrieve file
	file, _, err := r.FormFile("payload")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot find payload in the form:", err)
		badRequest("payload not found", w, r)
		return
	}
	defer file.Close()

	//TODO make a transaction
	if err := s.Service.UpdateMetadata(id, m); err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to write metadata to disk:", err)
		internalServerError(w, r)
		return
	}

	if err := s.Service.Copy(id, file); err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to write data to disk:", err)
		internalServerError(w, r)
		return
	}

	// Respond success
	w.WriteHeader(http.StatusCreated)
}

func (s HTTPServer) allHist(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "id")
	datastore := make([]string, 0)

	ds, err := s.Service.AllBackups(gameID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to open datastore (", s.documentRoot, "):", err)
		internalServerError(w, r)
		return
	}

	for _, d := range ds {
		datastore = append(datastore, d.UUID)
	}

	ok(datastore, w, r)
}

func (s HTTPServer) histUpload(w http.ResponseWriter, r *http.Request) {
	const (
		sizeLimit int64 = 500 << 20 // 500 MB
	)

	gameID := chi.URLParam(r, "id")
	uuid := chi.URLParam(r, "uuid")

	// Limit max upload size
	r.Body = http.MaxBytesReader(w, r.Body, sizeLimit)

	// Parse multipart form
	err := r.ParseMultipartForm(sizeLimit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to load payload:", err)
		badRequest("bad payload", w, r)
		return
	}

	// Retrieve file
	file, _, err := r.FormFile("payload")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot find payload in the form:", err)
		badRequest("payload not found", w, r)
		return
	}
	defer file.Close()

	if err := s.Service.CopyBackup(gameID, uuid, file); err != nil {
		fmt.Fprintln(os.Stderr, "error: failed to write data to disk:", err)
		internalServerError(w, r)
		return
	}

	// Respond success
	w.WriteHeader(http.StatusCreated)
}

func (s HTTPServer) histDownload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uuid := chi.URLParam(r, "uuid")
	path := filepath.Clean(filepath.Join(s.documentRoot, "data", id, "hist", uuid))

	fi, err := os.Stat(filepath.Join(path, "data.tar.gz"))
	if err != nil {
		notFound("id not found", w, r)
		return
	}

	f, err := s.Service.Repository().ReadBlob(repository.NewBackupIdentifier(id, uuid))
	if err != nil {
		slog.Error(err.Error())
		internalServerError(w, r)
		return
	}
	defer f.Close()

	// Set headers
	w.Header().Set("Content-Disposition", "attachment; filename=\"data.tar.gz\"")
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(200)

	// Stream the file content
	http.ServeContent(w, r, "data.tar.gz", fi.ModTime(), f)
}

func (s HTTPServer) histExists(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "id")
	uuid := chi.URLParam(r, "uuid")

	finfo, err := s.Service.Backup(gameID, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			notFound("not found", w, r)
			return
		}
		fmt.Fprintln(os.Stderr, "error: failed to read data:", err)
		internalServerError(w, r)
		return
	}

	ok(finfo, w, r)
}


func (s HTTPServer) metadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	metadata, err := s.Service.One(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			notFound("id not found", w, r)
			return
		}
		slog.Error(err.Error())
		internalServerError(w, r)
		return
	}
	ok(metadata, w, r)
}

func parseFormMetadata(gameID string, values map[string][]string) (repository.Metadata, error) {
	var name string
	if v, ok := values["name"]; ok {
		if len(v) == 0 {
			return repository.Metadata{}, fmt.Errorf("error: corrupted metadata")
		}
		name = v[0]
	} else {
		return repository.Metadata{}, fmt.Errorf("error: cannot find metadata in the form")
	}

	var version int
	if v, ok := values["version"]; ok {
		if len(v) == 0 {
			return repository.Metadata{}, fmt.Errorf("error: corrupted metadata")
		}
		if v, err := strconv.Atoi(v[0]); err == nil {
			version = v
		} else {
			return repository.Metadata{}, err
		}
	} else {
		return repository.Metadata{}, fmt.Errorf("error: cannot find metadata in the form")
	}

	var date time.Time
	if v, ok := values["date"]; ok {
		if len(v) == 0 {
			return repository.Metadata{}, fmt.Errorf("error: corrupted metadata")
		}
		if v, err := time.Parse(time.RFC3339, v[0]); err == nil {
			date = v
		} else {
			return repository.Metadata{}, err
		}
	} else {
		return repository.Metadata{}, fmt.Errorf("error: cannot find metadata in the form")
	}

	return repository.Metadata{
		ID:      gameID,
		Version: version,
		Name:    name,
		Date:    date,
	}, nil
}
