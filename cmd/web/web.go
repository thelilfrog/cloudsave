package main

import (
	"cloudsave/cmd/web/server"
	"cloudsave/cmd/web/server/config"
	"cloudsave/pkg/constants"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
)

func main() {
	fmt.Printf("CloudSave web -- v%s.%s.%s\n\n", constants.Version, runtime.GOOS, runtime.GOARCH)

	var configPath string
	flag.StringVar(&configPath, "config", "/var/lib/cloudsave/config.json", "Define the path to the configuration file")
	flag.Parse()

	c, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load configuration:", err)
		os.Exit(1)
	}

	s := server.NewServer(c)

	fmt.Println("starting server at :" + strconv.Itoa(c.Server.Port))
	if err := s.Server.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to start web server:", err)
		os.Exit(1)
	}
}
