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

	for _, pat := range []string{"powerMultipliers", "skillUpgradeVersion", "calcTeamPower", "synergy", "harvestMultiplier"} {
		idx := strings.Index(s, pat)
		if idx != -1 {
			start := idx - 100
			if start < 0 {
				start = 0
			}
			end := idx + 300
			if end > len(s) {
				end = len(s)
			}
			fmt.Printf("--- Found %s ---\n%s\n", pat, s[start:end])
		}
	}
}
