package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	pat := "/api/v1/auth/google"
	idx := strings.Index(s, pat)
	if idx != -1 {
		start := idx - 200
		if start < 0 {
			start = 0
		}
		end := idx + 300
		if end > len(s) {
			end = len(s)
		}
		fmt.Printf("main.js: %s\n", s[start:end])
	}

	b2, _ := os.ReadFile("farm_src/auth-text-rPtqp-nt.js")
	s2 := string(b2)
	idx2 := strings.Index(s2, pat)
	if idx2 != -1 {
		fmt.Printf("auth-text.js: %s\n", s2[idx2-100:idx2+200])
	}
}
