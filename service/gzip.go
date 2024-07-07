package service

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// gzipHandler
func gzipHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzWriter := gzipResponseWriter{ResponseWriter: w, Writer: gz}
		h.ServeHTTP(gzWriter, r)
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	*gzip.Writer
}

func (w gzipResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w gzipResponseWriter) Flush() {
	w.Writer.Flush()
}

func (w gzipResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}
