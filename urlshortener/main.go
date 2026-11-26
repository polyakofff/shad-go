//go:build !solution

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type ShortenRequest struct {
	Url string `json:"url"`
}

type ShortenResponse struct {
	Url string `json:"url"`
	Key string `json:"key"`
}

type URLShortener struct {
	mu      sync.RWMutex
	url2key map[string]string
	key2url map[string]string
}

func generateKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := 10
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func (sh *URLShortener) handleShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	sh.mu.Lock()
	key, ok := sh.url2key[req.Url]
	if !ok {
		key = generateKey()
		sh.url2key[req.Url] = key
		sh.key2url[key] = req.Url
	}
	sh.mu.Unlock()

	resp := ShortenResponse{
		Url: req.Url,
		Key: key,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(resp); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (sh *URLShortener) handleGo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.NotFound(w, r)
		return
	}

	key := parts[2]

	sh.mu.RLock()
	url, ok := sh.key2url[key]
	sh.mu.RUnlock()

	if !ok {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}

func runServer(port int) error {
	shortener := &URLShortener{
		url2key: make(map[string]string),
		key2url: make(map[string]string),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/shorten", shortener.handleShorten)
	mux.HandleFunc("/go/", shortener.handleGo)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverError := make(chan error, 1)
	go func() {
		serverError <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Println("notification received, shutting down server...")
	case err := <-serverError:

		return err
	}

	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		server.Close()
		return err
	}

	err := <-serverError
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func main() {
	port := flag.Int("port", 8080, "port to listen")
	flag.Parse()

	if err := runServer(*port); err != nil {
		log.Fatal(err)
	}
}
