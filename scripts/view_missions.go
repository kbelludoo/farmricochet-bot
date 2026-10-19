package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/WeeklyMissionsUi-MskdgUhX.js")
	s := string(b)
	pat := "/api/v1/missions"
	idx := 0
	for {
		pos := strings.Index(s[idx:], pat)
		if pos == -1 {
			break
		}
		realPos := idx + pos
		fmt.Printf("Missions API: %s\n", s[realPos:realPos+150])
		idx = realPos + len(pat)
	}
}
