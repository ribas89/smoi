package main

import (
	"io/fs"
	"net/http"
	"strings"

	"smoi/static"
)

func registerFrontendRoutes(router *http.ServeMux) error {
	frontend, err := fs.Sub(static.Files, "dist")
	if err != nil {
		return err
	}

	index, err := fs.ReadFile(frontend, "index.html")
	if err != nil {
		return err
	}

	serveApp := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	}

	router.HandleFunc("GET /{$}", serveApp)
	router.HandleFunc("GET /c/{id}", serveApp)
	router.HandleFunc("GET /admin", serveApp)
	router.Handle("GET /assets/", staticAssetHandler(frontend))
	return nil
}

func staticAssetHandler(frontend fs.FS) http.Handler {
	files := http.FileServer(http.FS(frontend))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assetPath := strings.TrimPrefix(r.URL.Path, "/")
		info, err := fs.Stat(frontend, assetPath)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=3600")
		files.ServeHTTP(w, r)
	})
}
