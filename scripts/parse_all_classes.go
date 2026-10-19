package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	b, err := os.ReadFile("farm_src/compendium-CA1PmrPK.js")
	if err != nil {
		panic(err)
	}
	s := string(b)

	// Find the classes array: [{name:"Ladino",role:...
	idx := strings.Index(s, `[{name:"`)
	if idx == -1 {
		idx = strings.Index(s, `[{name:`)
	}
	if idx != -1 {
		endIdx := strings.Index(s[idx:], "}]}],")
		if endIdx != -1 {
			classesChunk := s[idx : idx+endIdx+4]
			// Clean up e("pt-BR", ...) helper
			rePt := regexp.MustCompile(`e\("([^"]+)",[^\)]+\)`)
			cleaned := rePt.ReplaceAllString(classesChunk, `"$1"`)
			fmt.Println("=== ALL CLASSES & SKILLS ===")
			fmt.Println(cleaned)
		}
	}
}
