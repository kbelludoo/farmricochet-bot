package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	pat := "async function qr("
	idx := strings.Index(s, pat)
	if idx == -1 {
		pat = "function qr("
		idx = strings.Index(s, pat)
	}
	if idx == -1 {
		pat = "qr="
		idx = strings.Index(s, pat)
	}
	if idx != -1 {
		start := idx
		end := idx + 2000
		if end > len(s) {
			end = len(s)
		}
		fmt.Printf("qr definition:\n%s\n", s[start:end])
	}
}
