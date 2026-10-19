package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	pat := "startRun("
	idx := strings.Index(s, pat)
	if idx != -1 {
		start := idx - 50
		end := idx + 1500
		if end > len(s) {
			end = len(s)
		}
		fmt.Printf("startRun:\n%s\n", s[start:end])
	}

	pat2 := "startHeartbeat("
	idx2 := strings.Index(s, pat2)
	if idx2 != -1 {
		fmt.Printf("startHeartbeat:\n%s\n", s[idx2:idx2+500])
	}
}
