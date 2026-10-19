package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, err := os.ReadFile("farm_src/main-B9niyWwq.js")
	if err != nil {
		panic(err)
	}
	s := string(b)
	pat := "combatIntegrity"
	idx := 0
	for {
		pos := strings.Index(s[idx:], pat)
		if pos == -1 {
			break
		}
		realPos := idx + pos
		start := realPos - 100
		if start < 0 {
			start = 0
		}
		end := realPos + 200
		if end > len(s) {
			end = len(s)
		}
		fmt.Printf("--- Found combatIntegrity at %d ---\n%s\n", realPos, s[start:end])
		idx = realPos + len(pat)
	}
}
