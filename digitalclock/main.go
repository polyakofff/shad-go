//go:build !solution

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func getSymbol(c rune) string {
	switch c {
	case '0':
		return Zero
	case '1':
		return One
	case '2':
		return Two
	case '3':
		return Three
	case '4':
		return Four
	case '5':
		return Five
	case '6':
		return Six
	case '7':
		return Seven
	case '8':
		return Eight
	case '9':
		return Nine
	case ':':
		return Colon
	default:
		panic("invalid digit")
	}
}

func createImage(t string, k int) *image.RGBA {
	height := len(strings.Split(Zero, "\n"))
	width := len(strings.Split(Zero, "\n")[0])
	colonWidth := len(strings.Split(Colon, "\n")[0])
	img := image.NewRGBA(image.Rect(0, 0, (width*6+colonWidth*2)*k, height*k))

	draw.Draw(
		img,
		img.Bounds(),
		image.NewUniform(color.White),
		image.Point{},
		draw.Src,
	)

	shift := 0
	for _, c := range t {
		symbol := getSymbol(c)
		for y, line := range strings.Split(symbol, "\n") {
			for x, z := range line {
				if z == '1' {
					for imgX := 0; imgX < k; imgX++ {
						for imgY := 0; imgY < k; imgY++ {
							img.Set(shift+x*k+imgX, y*k+imgY, Cyan)
						}
					}
				}
			}
		}
		shift += len(strings.Split(symbol, "\n")[0]) * k
	}

	return img
}

func handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	const layout = "15:04:05"

	t := r.URL.Query().Get("time")
	if t == "" {
		t = time.Now().Format(layout)
	}
	parsed, err := time.Parse(layout, t)
	if err != nil || parsed.Format(layout) != t {
		http.Error(w, "Invalid time format", http.StatusBadRequest)
		return
	}

	k := r.URL.Query().Get("k")
	if k == "" {
		k = "1"
	}
	kInt, err := strconv.Atoi(k)
	if err != nil || (kInt < 1 || kInt > 30) {
		http.Error(w, "Invalid k value", http.StatusBadRequest)
		return
	}

	log.Printf("Received request for time: %s, k: %d", t, kInt)

	img := createImage(t, kInt)

	w.Header().Set("Content-Type", "image/png")
	err = png.Encode(w, img)

	if err != nil {
		http.Error(w, "Failed to encode image", http.StatusInternalServerError)
		return
	}
}

func runServer(port int) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(handle),
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

	port := flag.Int("port", 8080, "port to listen on")
	flag.Parse()

	if err := runServer(*port); err != nil {
		log.Fatal(err)
	}
}
