package server

import (
	"cloudsave/cmd/web/server/config"
	"cloudsave/pkg/constants"
	"cloudsave/pkg/remote/client"
	"cloudsave/pkg/repository"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"runtime"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	_ "embed"
)

type (
	HTTPServer struct {
		Server    *http.Server
		Config    config.Configuration
		Templates Templates
	}

	Templates struct {
		Dashboard *template.Template
		Detailled *template.Template
		System    *template.Template
	}
)

type (
	DetaillePayload struct {
		Version        string
		Save           repository.Metadata
		BackupMetadata []repository.Backup
		Hash           string
	}

	DashboardPayload struct {
		Version string
		Saves   []repository.Metadata
	}

	SystemPayload struct {
		Version string
		Client  client.Information
		Server  client.Information
	}
)

var (
	//go:embed templates/500.html
	InternalServerErrorHTMLPage string

	//go:embed templates/401.html
	UnauthorizedErrorHTMLPage string

	//go:embed templates/dashboard.html
	DashboardHTMLPage string

	//go:embed templates/detailled.html
	DetailledHTMLPage string

	//go:embed templates/information.html
	SystemHTMLPage string
)

// NewServer start the http server
func NewServer(c config.Configuration) *HTTPServer {
	dashboardTemplate := template.New("dashboard")
	dashboardTemplate.Parse(DashboardHTMLPage)

	detailledTemplate := template.New("detailled")
	detailledTemplate.Parse(DetailledHTMLPage)

	systemTemplate := template.New("system")
	systemTemplate.Parse(SystemHTMLPage)

	s := &HTTPServer{
		Config: c,
		Templates: Templates{
			Dashboard: dashboardTemplate,
			Detailled: detailledTemplate,
			System:    systemTemplate,
		},
	}
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(recoverMiddleware)
	router.Route("/web", func(routerAPI chi.Router) {
		routerAPI.Get("/", s.dashboard)
		routerAPI.Get("/{id}", s.detailled)
		routerAPI.Get("/system", s.system)
	})
	s.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", c.Server.Port),
		Handler: router,
	}
	return s
}

func (s *HTTPServer) dashboard(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		basicAuthFailed(w, r, "realm")
		return
	}

	cli := client.New(s.Config.Remote.URL, user, pass)

	if err := cli.Ping(); err != nil {
		slog.Error("unable to connect to the remote", "err", err)
		return
	}

	saves, err := cli.All()
	if err != nil {
		if errors.Is(err, client.ErrUnauthorized) {
			unauthorized("Unable to access resources", w, r)
			return
		}
		slog.Error("unable to connect to the remote", "err", err)
		return
	}

	slices.SortFunc(saves, func(a, b repository.Metadata) int {
		return a.Date.Compare(b.Date)
	})

	slices.Reverse(saves)

	payload := DashboardPayload{
		Version: constants.Version,
		Saves:   saves,
	}

	if err := s.Templates.Dashboard.Execute(w, payload); err != nil {
		slog.Error("failed to render the html pages", "err", err)
		return
	}
}

func (s *HTTPServer) detailled(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		basicAuthFailed(w, r, "realm")
		return
	}

	id := chi.URLParam(r, "id")
	cli := client.New(s.Config.Remote.URL, user, pass)

	if err := cli.Ping(); err != nil {
		slog.Error("unable to connect to the remote", "err", err)
		return
	}

	save, err := cli.Metadata(id)
	if err != nil {
		if errors.Is(err, client.ErrUnauthorized) {
			unauthorized("Unable to access resources", w, r)
			return
		}
		slog.Error("unable to connect to the remote", "err", err)
		return
	}

	h, err := cli.Hash(id)
	if err != nil {
		slog.Error("unable to connect to the remote", "err", err)
		return
	}

	ids, err := cli.ListArchives(id)
	if err != nil {
		slog.Error("unable to connect to the remote", "err", err)
		return
	}

	var bm []repository.Backup
	for _, i := range ids {
		b, err := cli.ArchiveInfo(id, i)
		if err != nil {
			slog.Error("unable to connect to the remote", "err", err)
			return
		}
		bm = append(bm, b)
	}

	payload := DetaillePayload{
		Save:           save,
		Hash:           h,
		BackupMetadata: bm,
		Version:        constants.Version,
	}

	if err := s.Templates.Detailled.Execute(w, payload); err != nil {
		slog.Error("failed to render the html pages", "err", err)
		return
	}
}

func (s *HTTPServer) system(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		basicAuthFailed(w, r, "realm")
		return
	}
	cli := client.New(s.Config.Remote.URL, user, pass)

	if err := cli.Ping(); err != nil {
		slog.Error("unable to connect to the remote", "err", err)
		return
	}

	clientInfo := client.Information{
		Version:        constants.Version,
		APIVersion:     constants.ApiVersion,
		GoVersion:      runtime.Version(),
		OSName:         runtime.GOOS,
		OSArchitecture: runtime.GOARCH,
	}
	serverInfo, err := cli.Version()
	if err != nil {
		if errors.Is(err, client.ErrUnauthorized) {
			unauthorized("Unable to access resources", w, r)
			return
		}
		slog.Error("unable to connect to the remote", "err", err)
		return
	}

	payload := SystemPayload{
		Version: constants.Version,
		Client:  clientInfo,
		Server:  serverInfo,
	}

	if err := s.Templates.System.Execute(w, payload); err != nil {
		slog.Error("failed to render the html pages", "err", err)
		return
	}
}
