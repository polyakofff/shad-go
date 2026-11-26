//go:build !solution

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	args := os.Args[1:]
	for _, url := range args {
		resp, err := http.Get(url)
		if err != nil {
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			os.Exit(2)
		}
		fmt.Println(string(body))
	}
}
