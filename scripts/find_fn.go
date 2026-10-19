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
	idx := strings.Index(s, "extensionSignature")
	if idx != -1 {
		start := idx - 200
		if start < 0 {
			start = 0
		}
		end := idx + 300
		if end > len(s) {
			end = len(s)
		}
		fmt.Printf("Around extensionSignature:\n%s\n", s[start:end])
	}

	// Also let's find Cn definition
	// e.g. function Cn or const Cn = or let Cn =
	for _, pat := range []string{"function Cn(", "const Cn=", "let Cn=", "var Cn=", "Cn=()=>", "Cn=function"} {
		idx = strings.Index(s, pat)
		if idx != -1 {
			start := idx - 100
			if start < 0 {
				start = 0
			}
			end := idx + 300
			if end > len(s) {
				end = len(s)
			}
			fmt.Printf("Found %s:\n%s\n", pat, s[start:end])
		}
	}
}
