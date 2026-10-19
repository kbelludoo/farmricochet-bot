package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	idx := strings.Index(s, "recordStructureEvent(e,t,a){")
	if idx != -1 {
		start := idx
		end := idx + 1000
		if end > len(s) {
			end = len(s)
		}
		fmt.Println(s[start:end])
	}
}
