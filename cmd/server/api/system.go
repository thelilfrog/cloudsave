package api

import (
	"cloudsave/pkg/constants"
	"net/http"
	"runtime"
)

type information struct {
	Version        string `json:"version"`
	APIVersion     int    `json:"api_version"`
	GoVersion      string `json:"go_version"`
	OSName         string `json:"os_name"`
	OSArchitecture string `json:"os_architecture"`
}

func (s *HTTPServer) Information(w http.ResponseWriter, r *http.Request) {
	info := information{
		Version:        constants.Version,
		APIVersion:     constants.ApiVersion,
		GoVersion:      runtime.Version(),
		OSName:         runtime.GOOS,
		OSArchitecture: runtime.GOARCH,
	}
	ok(info, w, r)
}
