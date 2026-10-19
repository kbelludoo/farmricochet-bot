package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/main-B9niyWwq.js")
	s := string(b)
	idx := strings.Index(s, "class Ao{")
	if idx != -1 {
		fmt.Printf("%s\n", s[idx:idx+1500])
	}
}
