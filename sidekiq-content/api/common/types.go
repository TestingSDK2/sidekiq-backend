package common

import (
	"net/http"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app"
)

// HandlerFuncWithCTX - type is an adapter to use handlerfunc with ctx
type HandlerFuncWithCTX func(*app.Context, http.ResponseWriter, *http.Request) error

type StatusCodeRecorder struct {
	http.ResponseWriter
	http.Hijacker
	StatusCode int
}

func (r *StatusCodeRecorder) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
