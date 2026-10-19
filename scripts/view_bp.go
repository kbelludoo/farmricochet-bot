package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	b, _ := os.ReadFile("farm_src/BattlePassMenuUi-CXgsnHEZ.js")
	s := string(b)
	pat := "/api/v1/battle-pass"
	idx := 0
	for {
		pos := strings.Index(s[idx:], pat)
		if pos == -1 {
			break
		}
		realPos := idx + pos
		fmt.Printf("BP API: %s\n", s[realPos:realPos+180])
		idx = realPos + len(pat)
	}
}
