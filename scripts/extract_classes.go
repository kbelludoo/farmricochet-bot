package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, err := os.ReadFile("farm_src/compendium-CA1PmrPK.js")
	if err != nil {
		panic(err)
	}
	s := string(b)

	// Search for class definitions
	// Each class has: farmer, knight, runner, priest, wizard, swordsman, lancer, woodcutter, templar
	classes := []string{"farmer", "knight", "runner", "priest", "wizard", "swordsman", "lancer", "woodcutter", "templar"}

	for _, c := range classes {
		idx := strings.Index(s, `code:"`+c+`"`)
		if idx == -1 {
			idx = strings.Index(s, `code:'`+c+`'`)
		}
		if idx == -1 {
			idx = strings.Index(s, `"`+c+`"`)
		}
		fmt.Printf("\n==================== CLASS: %s ====================\n", strings.ToUpper(c))
		if idx != -1 {
			start := idx - 50
			if start < 0 {
				start = 0
			}
			end := idx + 2500
			if end > len(s) {
				end = len(s)
			}
			fmt.Println(s[start:end])
		} else {
			fmt.Println("Not found directly by code")
		}
	}
}
