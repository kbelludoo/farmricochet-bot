package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	files, _ := filepath.Glob("farm_src/*.js")

	fmt.Println("=== SEARCHING COMPENDIUM & CLASS DATA ===")
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		s := string(data)

		bn := filepath.Base(f)
		if strings.Contains(bn, "compendium") || strings.Contains(bn, "class") || strings.Contains(bn, "character") {
			fmt.Printf("Analyzing %s (%d bytes)...\n", bn, len(s))
		}
	}

	compBytes, err := os.ReadFile("farm_src/compendium-CA1PmrPK.js")
	if err == nil {
		s := string(compBytes)
		fmt.Printf("\n--- Compendium snippet ---\n%s\n", s[:min(1200, len(s))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
