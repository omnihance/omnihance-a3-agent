package server

import (
	"io"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(
		httplog.RequestLogger(httplog.NewLogger(
			"omnihance-a3-agent",
			httplog.Options{JSON: true, LogLevel: s.cfg.GetLogLevel().String()}),
		),
	)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://omnihance.com", "https://*.omnihance.com", "http://localhost:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(middleware.Heartbeat("/health"))

	s.InitializeDocsRoutes(r)
	s.InitializeStatusRoutes(r)
	s.InitializeAuthRoutes(r)
	s.InitializeFileSystemRoutes(r)
	s.InitializeMetricsRoutes(r)
	s.InitializeSessionRoutes(r)
	r.Handle("/*", s.FrontendHandler())

	return r
}

func (s *Server) FrontendHandler() http.Handler {
	distFs, err := fs.Sub(s.frontendFiles, "omnihance-a3-agent-ui/dist")
	if err != nil {
		return http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = path[1:]
		}

		file, err := distFs.Open(path)
		if err != nil {
			indexFile, err := distFs.Open("index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}

			defer func() {
				_ = indexFile.Close()
			}()
			stat, _ := indexFile.Stat()
			http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
			return
		}

		defer func() {
			_ = file.Close()
		}()
		stat, _ := file.Stat()
		if stat.IsDir() {
			http.NotFound(w, r)
			return
		}

		http.ServeContent(w, r, stat.Name(), stat.ModTime(), file.(io.ReadSeeker))
	})
}
