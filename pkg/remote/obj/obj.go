package obj

import "time"

type (
	HTTPCore struct {
		Status    int       `json:"status"`
		Timestamp time.Time `json:"timestamp"`
		Path      string    `json:"path"`
	}

	HTTPError struct {
		HTTPCore
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	HTTPObject struct {
		HTTPCore
		Data any `json:"data"`
	}
)
