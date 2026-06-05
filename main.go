package main

import (
	"bytes"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

//go:embed frontend/dist/*
var frontendFS embed.FS

var veilProxy *httputil.ReverseProxy
var proxyCache *Cache

func init() {
	remote, _ := url.Parse("https://veil.ortlinde.com")
	veilProxy = httputil.NewSingleHostReverseProxy(remote)
	veilProxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
		Proxy:                 http.ProxyFromEnvironment,
	}
	director := veilProxy.Director
	veilProxy.Director = func(req *http.Request) {
		director(req)
		req.Host = remote.Host
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		req.Header.Set("Referer", "https://veil.ortlinde.com/")
		req.Header.Del("X-Forwarded-For")
		req.Header.Del("X-Real-IP")
	}

	proxyCache = NewCache()
	proxyCache.SetTTL("/v1/tags", 120*time.Second)
	proxyCache.SetTTL("/v1/featured-tags", 120*time.Second)
	proxyCache.SetTTL("/v1/categories", 120*time.Second)
	proxyCache.SetTTL("/v1/site-config", 300*time.Second)
}

func main() {
	password := os.Getenv("ACCESS_PASSWORD")
	if password == "" {
		log.Println("ACCESS_PASSWORD not set — no auth required")
	}

	mux := http.NewServeMux()

	// Auth
	if password != "" {
		mux.HandleFunc("/api/auth", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				http.Error(w, "method not allowed", 405)
				return
			}
			var body struct {
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "bad request", 400)
				return
			}
			if subtle.ConstantTimeCompare([]byte(body.Password), []byte(password)) == 1 {
				w.WriteHeader(200)
				json.NewEncoder(w).Encode(map[string]bool{"ok": true})
				return
			}
			http.Error(w, "forbidden", 403)
		})
	}

	mux.HandleFunc("/api/proxy/", proxyHandler)
	mux.HandleFunc("/api/proxy", proxyHandler)

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

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	targetPath := strings.TrimPrefix(r.URL.Path, "/api/proxy")
	start := time.Now()

	// Skip cache for image/random streams
	if shouldCache(targetPath) {
		if entry, ok := proxyCache.Get(targetPath, r.URL.RawQuery); ok {
			w.Header().Set("Content-Type", entry.ContentType)
			if entry.ContentEncoding != "" {
				w.Header().Set("Content-Encoding", entry.ContentEncoding)
			}
			w.Header().Set("X-Cache", "HIT")
			w.Write(entry.Data)
			log.Printf("[PROXY] %s %s -> 200 (cache) %s", r.Method, r.URL.Path, time.Since(start))
			return
		}
	}

	r.URL.Path = targetPath
	rec := &responseRecorder{ResponseWriter: w, code: 200}
	veilProxy.ServeHTTP(rec, r)

	elapsed := time.Since(start)
	log.Printf("[PROXY] %s %s -> %d (%dms)", r.Method, r.URL.Path, rec.code, elapsed.Milliseconds())

	if shouldCache(targetPath) && rec.code == 200 && len(rec.body.Bytes()) > 0 {
		proxyCache.Set(targetPath, r.URL.RawQuery, rec.body.Bytes(), rec.Header().Get("Content-Type"), rec.Header().Get("Content-Encoding"))
	}
}

func shouldCache(path string) bool {
	if strings.HasPrefix(path, "/v1/tags") ||
		strings.HasPrefix(path, "/v1/featured-tags") ||
		strings.HasPrefix(path, "/v1/categories") ||
		(strings.HasPrefix(path, "/v1/image/") && strings.HasSuffix(path, "/meta")) {
		return true
	}
	return false
}

type responseRecorder struct {
	http.ResponseWriter
	code int
	body bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

type cacheEntry struct {
	Data            []byte
	ContentType     string
	ContentEncoding string
	Expires         time.Time
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]*cacheEntry
	ttl   time.Duration
	ttlKV map[string]time.Duration
}

func NewCache() *Cache {
	return &Cache{
		items: make(map[string]*cacheEntry),
		ttl:   60 * time.Second,
		ttlKV: make(map[string]time.Duration),
	}
}

func (c *Cache) SetTTL(path string, ttl time.Duration) {
	c.ttlKV[path] = ttl
}

func (c *Cache) key(path, query string) string {
	if query != "" {
		return path + "?" + query
	}
	return path
}

func (c *Cache) ttlFor(path string) time.Duration {
	if t, ok := c.ttlKV[path]; ok {
		return t
	}
	return c.ttl
}

func (c *Cache) Get(path, query string) (*cacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[c.key(path, query)]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.Expires) {
		return nil, false
	}
	return entry, true
}

func (c *Cache) Set(path, query string, data []byte, contentType, contentEncoding string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[c.key(path, query)] = &cacheEntry{
		Data:            data,
		ContentType:     contentType,
		ContentEncoding: contentEncoding,
		Expires:         time.Now().Add(c.ttlFor(path)),
	}
}
