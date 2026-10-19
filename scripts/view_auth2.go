package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	pat := "async function un("
	idx := strings.Index(s, pat)
	if idx != -1 {
		fmt.Printf("%s\n", s[idx:idx+1200])
	}
}
