package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	idx := strings.Index(s, "function Pn(")
	if idx == -1 {
		idx = strings.Index(s, "Pn=")
	}
	if idx != -1 {
		start := idx - 100
		if start < 0 {
			start = 0
		}
		end := idx + 500
		if end > len(s) {
			end = len(s)
		}
		fmt.Printf("%s\n", s[start:end])
	}
}
