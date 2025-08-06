package server

import (
	"fmt"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				internalServerError(w, r)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// BasicAuth implements a simple middleware handler for adding basic http auth to a route.
func BasicAuth(realm string, creds map[string]string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok {
				basicAuthFailed(w, r, realm)
				return
			}

			credPass := creds[user]
			if err := bcrypt.CompareHashAndPassword([]byte(credPass), []byte(pass)); err != nil {
				basicAuthFailed(w, r, realm)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func basicAuthFailed(w http.ResponseWriter, r *http.Request, realm string) {
	unauthorized(realm, w, r)
}

func unauthorized(realm string, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	w.WriteHeader(http.StatusUnauthorized)
	_, err := w.Write([]byte(UnauthorizedErrorHTMLPage))
	if err != nil {
		log.Println(err)
	}
}

func internalServerError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	_, err := w.Write([]byte(InternalServerErrorHTMLPage))
	if err != nil {
		log.Println(err)
	}
}
