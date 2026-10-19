package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	idx := strings.Index(s, "recordStructureEvent(")
	if idx != -1 {
		start := idx - 50
		end := idx + 700
		if end > len(s) {
			end = len(s)
		}
		fmt.Println(s[start:end])
	}
}
