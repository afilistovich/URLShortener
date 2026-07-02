package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math/big"
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

var (
	urlStorage = map[string]string{}
	storeMu    sync.RWMutex
)

func isValidUrl(s string) bool {
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
			return "", fmt.Errorf("failed to generate random data")
		}
		b[i] = idAlphabet[idx.Int64()]
	}
	return string(b), nil
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "bad request", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	rawUrl := strings.TrimSpace(string(body))

	if rawUrl == "" || !isValidUrl(rawUrl) {
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
		storeMu.RLock()
		_, exist := urlStorage[candidate]
		storeMu.RUnlock()

		if !exist {
			id = candidate
			break
		}
	}

	storeMu.Lock()
	urlStorage[id] = rawUrl
	storeMu.Unlock()

	shortURL := "http://" + serverHost + "/" + id
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(shortURL))
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "bad request", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	storeMu.RLock()
	original, exist := urlStorage[id]
	storeMu.RUnlock()

	if !exist {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", original)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		shortenHandler(w, r)
		return
	}
	redirectHandler(w, r)
}

func main() {
	log.Printf("URL shortener is running on http://%s", serverHost)
	if err := http.ListenAndServe(serverHost, http.HandlerFunc(mainHandler)); err != nil {
		log.Fatal(err)
	}
}
