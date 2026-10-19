package main

import (
	"fmt"
	"os"
)

func main() {
	b, _ := os.ReadFile("farm_src/campaign-D5M1rayi.js")
	s := string(b)
	fmt.Printf("File content of campaign-D5M1rayi.js:\n%s\n", s)
}
