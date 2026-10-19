package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	pat := "const n=i.phase?.name||"
	idx := strings.Index(s, pat)
	if idx != -1 {
		start := idx
		end := idx + 2500
		if end > len(s) {
			end = len(s)
		}
		fmt.Printf("Rest of qr:\n%s\n", s[start:end])
	}
}
