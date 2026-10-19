package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	pat := "./campaign-D5M1rayi.js"
	idx := strings.Index(s, pat)
	if idx != -1 {
		start := idx - 200
		if start < 0 {
			start = 0
		}
		end := idx + 50
		fmt.Printf("import: %s\n", s[start:end])
	}
}
