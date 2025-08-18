package api

import (
	"cloudsave/pkg/remote/obj"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

func internalServerError(w http.ResponseWriter, r *http.Request) {
	payload := obj.HTTPError{
		HTTPCore: obj.HTTPCore{
			Status:    http.StatusInternalServerError,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Error:   "Internal Server Error",
		Message: "The server encountered an unexpected condition that prevented it from fulfilling the request.",
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	e := json.NewEncoder(w)
	if err := e.Encode(payload); err != nil {
		slog.Error(err.Error())
	}
}

func notFound(message string, w http.ResponseWriter, r *http.Request) {
	payload := obj.HTTPError{
		HTTPCore: obj.HTTPCore{
			Status:    http.StatusNotFound,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Error:   "Not Found",
		Message: message,
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	e := json.NewEncoder(w)
	if err := e.Encode(payload); err != nil {
		slog.Error(err.Error())
	}
}

func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	payload := obj.HTTPError{
		HTTPCore: obj.HTTPCore{
			Status:    http.StatusMethodNotAllowed,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Error:   "Method Not Allowed",
		Message: "The server knows the request method, but the target resource doesn't support this method",
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	e := json.NewEncoder(w)
	if err := e.Encode(payload); err != nil {
		slog.Error(err.Error())
	}
}

func unauthorized(w http.ResponseWriter, r *http.Request) {
	payload := obj.HTTPError{
		HTTPCore: obj.HTTPCore{
			Status:    http.StatusUnauthorized,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Error:   "Unauthorized",
		Message: "The request has not been completed because it lacks valid authentication credentials for the requested resource.",
	}

	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("WWW-Authenticate", "Custom realm=\"loginUserHandler via /api/login\"")
	w.WriteHeader(http.StatusUnauthorized)
	e := json.NewEncoder(w)
	if err := e.Encode(payload); err != nil {
		slog.Error(err.Error())
	}
}

func ok(o interface{}, w http.ResponseWriter, r *http.Request) {
	payload := obj.HTTPObject{
		HTTPCore: obj.HTTPCore{
			Status:    http.StatusOK,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Data: o,
	}
	w.Header().Add("Content-Type", "application/json")
	e := json.NewEncoder(w)
	if err := e.Encode(payload); err != nil {
		slog.Error(err.Error())
	}
}

func badRequest(message string, w http.ResponseWriter, r *http.Request) {
	payload := obj.HTTPError{
		HTTPCore: obj.HTTPCore{
			Status:    http.StatusBadRequest,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Error:   "Bad Request",
		Message: message,
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	e := json.NewEncoder(w)
	if err := e.Encode(payload); err != nil {
		slog.Error(err.Error())
	}
}
