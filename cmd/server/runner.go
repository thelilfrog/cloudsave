package main

import (
	"cloudsave/cmd/server/api"
	"cloudsave/cmd/server/security/htpasswd"
	"cloudsave/pkg/constants"
	"cloudsave/pkg/data"
	"cloudsave/pkg/repository"
	"flag"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strconv"
)

func run() {
	fmt.Printf("CloudSave server -- v%s.%s.%s\n\n", constants.Version, runtime.GOOS, runtime.GOARCH)

	var documentRoot string
	var port int
	var noCache, verbose bool
	flag.StringVar(&documentRoot, "document-root", defaultDocumentRoot, "Define the path to the document root")
	flag.IntVar(&port, "port", 8080, "Define the port of the server")
	flag.BoolVar(&noCache, "no-cache", false, "Disable the cache")
	flag.BoolVar(&verbose, "verbose", false, "Show more logs")
	flag.Parse()

	if verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	slog.Info("loading .htpasswd")
	h, err := htpasswd.Open(filepath.Join(documentRoot, ".htpasswd"))
	if err != nil {
		fatal("failed to load .htpasswd: "+err.Error(), 1)
	}
	slog.Info("users loaded: " + strconv.Itoa(len(h.Content())) + " user(s) loaded")

	var repo repository.Repository
	if !noCache {
		slog.Info("loading eager repository...")
		r, err := repository.NewEagerRepository(filepath.Join(documentRoot, "data"))
		if err != nil {
			fatal("failed to load datastore: "+err.Error(), 1)
		}
		if err := r.Preload(); err != nil {
			fatal("failed to load datastore: "+err.Error(), 1)
		}
		repo = r
	} else {
		slog.Info("loading lazy repository...")
		repo, err = repository.NewLazyRepository(filepath.Join(documentRoot, "data"))
		if err != nil {
			fatal("failed to load datastore: "+err.Error(), 1)
		}
	}

	slog.Info("repository loaded")
	s := data.NewService(repo)

	server := api.NewServer(documentRoot, s, h.Content(), port)

	fmt.Println("server started at :" + strconv.Itoa(port))
	if err := server.Server.ListenAndServe(); err != nil {
		fatal("failed to start server: "+err.Error(), 1)
	}
}
