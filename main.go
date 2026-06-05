package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
)

//go:embed frontend/dist/*
var frontendFS embed.FS

var veilProxy *httputil.ReverseProxy

func init() {
	remote, _ := url.Parse("https://veil.ortlinde.com")
	veilProxy = httputil.NewSingleHostReverseProxy(remote)
	director := veilProxy.Director
	veilProxy.Director = func(req *http.Request) {
		director(req)
		req.Host = remote.Host
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		req.Header.Set("Referer", "https://veil.ortlinde.com/")
		req.Header.Del("X-Forwarded-For")
		req.Header.Del("X-Real-IP")
	}
}

func main() {
	mux := http.NewServeMux()

	// API proxy
	mux.HandleFunc("/api/proxy/", veilProxyHandler)
	mux.HandleFunc("/api/proxy", veilProxyHandler)

	// Static files with SPA fallback
	staticFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		fpath := path.Clean(r.URL.Path)
		if fpath == "/" {
			fpath = "/index.html"
		}
		_, err := fs.Stat(staticFS, fpath[1:])
		if err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8808"
	}
	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func veilProxyHandler(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api/proxy")
	veilProxy.ServeHTTP(w, r)
}
