package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

func main() {
	files, _ := filepath.Glob("farm_src/*.js")
	apiRegex := regexp.MustCompile(`/api/v1/[a-zA-Z0-9_\-\./\$\{\}\: ]+`)
	cmdRegex := regexp.MustCompile(`["'](?:run\.|campaign\.|hero\.|inventory\.|chest\.|market\.|pvp\.|farm\.|quest\.|mission\.|item\.|forge\.|legacy\.|ascend\.|divinity\.|battlepass\.|prestige\.|talent\.|gacha\.|chat\.|social\.|mail\.|claim\.|buy\.|sell\.)[a-zA-Z0-9_\.\-]+["']`)

	apiMap := make(map[string]bool)
	cmdMap := make(map[string]bool)

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		str := string(data)
		matches := apiRegex.FindAllString(str, -1)
		for _, m := range matches {
			apiMap[m] = true
		}
		cMatches := cmdRegex.FindAllString(str, -1)
		for _, cm := range cMatches {
			cmdMap[cm] = true
		}
	}

	fmt.Println("=== DETECTED API ENDPOINTS ===")
	apis := make([]string, 0, len(apiMap))
	for k := range apiMap {
		apis = append(apis, k)
	}
	sort.Strings(apis)
	for _, a := range apis {
		fmt.Println(a)
	}

	fmt.Println("\n=== DETECTED COMMANDS ===")
	cmds := make([]string, 0, len(cmdMap))
	for k := range cmdMap {
		cmds = append(cmds, k)
	}
	sort.Strings(cmds)
	for _, c := range cmds {
		fmt.Println(c)
	}
}
