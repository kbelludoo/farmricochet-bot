package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	pat := "async startRun("
	idx := strings.Index(s, pat)
	if idx == -1 {
		pat = "startRun() {"
		idx = strings.Index(s, pat)
	}
	if idx == -1 {
		pat = "startRun(){"
		idx = strings.Index(s, pat)
	}
	if idx != -1 {
		start := idx
		end := idx + 2000
		if end > len(s) {
			end = len(s)
		}
		fmt.Printf("startRun definition:\n%s\n", s[start:end])
	}
}
