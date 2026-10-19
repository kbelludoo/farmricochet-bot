package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	pat := "campaign-D5M1rayi.js"
	idx := 0
	for {
		pos := strings.Index(s[idx:], pat)
		if pos == -1 {
			break
		}
		realPos := idx + pos
		start := realPos - 150
		if start < 0 {
			start = 0
		}
		end := realPos + 100
		if end > len(s) {
			end = len(s)
		}
		fmt.Printf("found at %d: %s\n", realPos, s[start:end])
		idx = realPos + len(pat)
	}
}
