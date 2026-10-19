package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	files, _ := filepath.Glob("farm_src/*.js")
	re := regexp.MustCompile(`[a-zA-Z0-9_\-]+=[a-zA-Z0-9_\-]+`)
	cookieRe := regexp.MustCompile(`(?i)(cookie|session|token|auth|header)`)

	for _, f := range files {
		data, _ := os.ReadFile(f)
		s := string(data)
		matches := cookieRe.FindAllStringIndex(s, -1)
		for _, m := range matches {
			start := m[0] - 40
			if start < 0 {
				start = 0
			}
			end := m[1] + 60
			if end > len(s) {
				end = len(s)
			}
			snippet := s[start:end]
			if re.MatchString(snippet) && (filepath.Base(f) == "client-DeGCHmUx.js" || filepath.Base(f) == "main-B9niyWwq.js" || filepath.Base(f) == "game.js") {
				fmt.Printf("[%s] %s\n", filepath.Base(f), snippet)
			}
		}
	}
}
