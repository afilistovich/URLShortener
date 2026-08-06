package main

import (
	"URLShortener/internal/config"
	"URLShortener/internal/http-server/handlers/redirect"
	"URLShortener/internal/http-server/handlers/remove"
	"URLShortener/internal/http-server/handlers/save"
	"URLShortener/internal/http-server/middleware/mwLogger"
	"URLShortener/internal/lib/logger/sl"
	"URLShortener/internal/storage/sqlite"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lmittmann/tint"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()
	fmt.Println(cfg)

	log := setupLogger(cfg.Env)

	log.Info("starting url-shortener", slog.String("envi ronment", cfg.Env))
	log.Debug("debug logging enabled")

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/url", save.New(log, storage))
	router.Get("/{alias}", redirect.New(log, storage))
	router.Delete("/url/{alias}", remove.New(log, storage))

	log.Info("starting server", slog.String("address", cfg.Address))

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	if err = srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}

}

func setupLogger(env string) *slog.Logger {
	var logger *slog.Logger

	switch env {
	case envLocal:
		logger = setupPrettySlog()
	case envDev:
		logger = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		logger = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return logger
}

func setupPrettySlog() *slog.Logger {
	handler := tint.NewTextHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "15:05:05:000",
	})
	return slog.New(handler)
}

/*
import (
	"crypto/rand"
	"io"
	"log"
	"math/big"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	serverHost = "localhost:8080"
	idLength   = 8
	idAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type URLShortener struct {
	urlStorage map[string]string
	storeMu    sync.RWMutex
}

func NewURLShortener() *URLShortener {
	return &URLShortener{urlStorage: make(map[string]string)}
}

func isValidURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}

	if u.Host == "" {
		return false
	}

	return u.Scheme == "http" || u.Scheme == "https"
}

func generateID() (string, error) {
	b := make([]byte, idLength)
	n := big.NewInt(int64(len(idAlphabet)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", err
		}
		b[i] = idAlphabet[idx.Int64()]
	}
	return string(b), nil
}

func (s *URLShortener) ShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "bad request", http.StatusMethodNotAllowed)
		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "text/plain" {
		http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	rawURL := strings.TrimSpace(string(body))

	if rawURL == "" || !isValidURL(rawURL) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var id string

	for {
		candidate, err := generateID()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		s.storeMu.Lock()
		if _, exists := s.urlStorage[candidate]; !exists {
			id = candidate
			s.urlStorage[id] = rawURL
			s.storeMu.Unlock()
			break
		}
		s.storeMu.Unlock()
	}

	shortURL := "http://" + serverHost + "/" + id
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(shortURL))
}

func (s *URLShortener) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "bad request", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	s.storeMu.RLock()
	original, exist := s.urlStorage[id]
	s.storeMu.RUnlock()

	if !exist {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", original)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {
	shortener := NewURLShortener()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", shortener.ShortenHandler)
	mux.HandleFunc("GET /{id}", shortener.RedirectHandler)

	log.Printf("URL shortener is running on http://%s", serverHost)
	if err := http.ListenAndServe(serverHost, mux); err != nil {
		log.Fatal(err)
	}
}
*/
