package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type (
	httpCore struct {
		Status    int       `json:"status"`
		Timestamp time.Time `json:"timestamp"`
		Path      string    `json:"path"`
	}

	httpError struct {
		httpCore
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	httpObject struct {
		httpCore
		Data any `json:"data"`
	}
)

func internalServerError(w http.ResponseWriter, r *http.Request) {
	e := httpError{
		httpCore: httpCore{
			Status:    http.StatusInternalServerError,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Error:   "Internal Server Error",
		Message: "The server encountered an unexpected condition that prevented it from fulfilling the request.",
	}

	payload, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, err = w.Write(payload)
	if err != nil {
		log.Println(err)
	}
}

func notFound(message string, w http.ResponseWriter, r *http.Request) {
	e := httpError{
		httpCore: httpCore{
			Status:    http.StatusNotFound,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Error:   "Not Found",
		Message: message,
	}

	payload, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_, err = w.Write(payload)
	if err != nil {
		log.Println(err)
	}
}

func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	e := httpError{
		httpCore: httpCore{
			Status:    http.StatusMethodNotAllowed,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Error:   "Method Not Allowed",
		Message: "The server knows the request method, but the target resource doesn't support this method",
	}

	payload, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	_, err = w.Write(payload)
	if err != nil {
		log.Println(err)
	}
}

func unauthorized(w http.ResponseWriter, r *http.Request) {
	e := httpError{
		httpCore: httpCore{
			Status:    http.StatusUnauthorized,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Error:   "Unauthorized",
		Message: "The request has not been completed because it lacks valid authentication credentials for the requested resource.",
	}

	payload, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("WWW-Authenticate", "Custom realm=\"loginUserHandler via /api/login\"")
	w.WriteHeader(http.StatusUnauthorized)
	_, err = w.Write(payload)
	if err != nil {
		log.Println(err)
	}
}

func forbidden(w http.ResponseWriter, r *http.Request) {
	e := httpError{
		httpCore: httpCore{
			Status:    http.StatusForbidden,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Error:   "Forbidden",
		Message: "The access is permanently forbidden and tied to the application logic, such as insufficient rights to a resource.",
	}

	payload, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, err = w.Write(payload)
	if err != nil {
		log.Println(err)
	}
}

func ok(obj interface{}, w http.ResponseWriter, r *http.Request) {
	e := httpObject{
		httpCore: httpCore{
			Status:    http.StatusOK,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Data: obj,
	}

	payload, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
	}
	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write(payload)
	if err != nil {
		log.Println(err)
	}
}

func badRequest(message string, w http.ResponseWriter, r *http.Request) {
	e := httpError{
		httpCore: httpCore{
			Status:    http.StatusBadRequest,
			Path:      r.RequestURI,
			Timestamp: time.Now(),
		},
		Error:   "Bad Request",
		Message: message,
	}

	payload, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, err = w.Write(payload)
	if err != nil {
		log.Println(err)
	}
}
