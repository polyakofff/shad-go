//go:build !solution

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	args := os.Args[1:]

	start := time.Now()
	ch := make(chan string)

	for _, url := range args {
		go func(url string) {
			resp, err := http.Get(url)
			if err != nil {
				ch <- fmt.Sprintf("GET %s: %v", url, err)
				return
			}

			defer resp.Body.Close()

			_, err = io.ReadAll(resp.Body)
			if err != nil {
				ch <- fmt.Sprintf("read all %s: %v", url, err)
				return
			}

			ch <- fmt.Sprintf("%dms %s", time.Since(start).Milliseconds(), url)

		}(url)
	}

	for range args {
		fmt.Println(<-ch)
	}

	fmt.Printf("%dms\n", time.Since(start).Milliseconds())
}
