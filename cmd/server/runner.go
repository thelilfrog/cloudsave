package main

import (
	"cloudsave/cmd/server/api"
	"cloudsave/cmd/server/security/htpasswd"
	"cloudsave/pkg/constants"
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
)

func run() {
	fmt.Printf("CloudSave server -- v%s.%s.%s\n\n", constants.Version, runtime.GOOS, runtime.GOARCH)

	var documentRoot string
	var port int
	flag.StringVar(&documentRoot, "document-root", defaultDocumentRoot, "Define the path to the document root")
	flag.IntVar(&port, "port", 8080, "Define the port of the server")
	flag.Parse()

	h, err := htpasswd.Open(filepath.Join(documentRoot, ".htpasswd"))
	if err != nil {
		fatal("failed to load .htpasswd: "+err.Error(), 1)
	}

	server := api.NewServer(documentRoot, h.Content(), port)

	fmt.Println("starting server at :" + strconv.Itoa(port))
	if err := server.Server.ListenAndServe(); err != nil {
		fatal("failed to start server: "+err.Error(), 1)
	}
}
