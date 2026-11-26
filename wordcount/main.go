//go:build !solution

package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	count := make(map[string]int)

	for _, filename := range os.Args[1:] {
		file, err := os.Open(filename)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			count[line]++
		}

		if err := scanner.Err(); err != nil {
			panic(err)
		}
	}

	for line, c := range count {
		if c >= 2 {
			fmt.Printf("%d\t%s\n", c, line)
		}
	}

}
