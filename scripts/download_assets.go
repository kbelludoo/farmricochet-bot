package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var files = []string{
	"main-B9niyWwq.js",
	"client-DeGCHmUx.js",
	"GameSessionLease-hDJLfaFU.js",
	"GameEvents-BQQbvFH2.js",
	"identity-DkTwuxH8.js",
	"html-Ds7N7FgU.js",
	"hero-portraits-DHdUCmRS.js",
	"reward-badge-CVZXnZZy.js",
	"class-presentation-DT0MRBXk.js",
	"icons-DKqKqHsB.js",
	"vip-Swj5hWc5.js",
	"item-i18n-BSURDFjV.js",
	"features-knM0tJNj.js",
	"progress-D7vilDYf.js",
	"campaign-D5M1rayi.js",
	"characters-okRVcJ51.js",
	"hero-xp-HTzhmPhB.js",
	"infinite-DGANHuqr.js",
	"run-loot-D9EEUOkK.js",
	"onboarding-i18n-BYJ7gFrK.js",
	"InventoryCapacityUi-CTHiREQM.js",
	"index-tLx1ZRcz.js",
	"auth-text-rPtqp-nt.js",
}

func main() {
	outDir := "farm_src"
	_ = os.MkdirAll(outDir, 0755)

	client := &http.Client{}
	for _, f := range files {
		url := "https://farmricochet.com/assets/" + f
		dest := filepath.Join(outDir, f)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("err req %s: %v\n", f, err)
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("err get %s: %v\n", f, err)
			continue
		}
		if resp.StatusCode != 200 {
			fmt.Printf("status %d for %s\n", resp.StatusCode, f)
			resp.Body.Close()
			continue
		}
		out, err := os.Create(dest)
		if err != nil {
			resp.Body.Close()
			continue
		}
		n, _ := io.Copy(out, resp.Body)
		out.Close()
		resp.Body.Close()
		fmt.Printf("Downloaded %s (%d bytes)\n", f, n)
	}

	// Also let's scan main-B9niyWwq.js for other imported .js files
	mainBytes, err := os.ReadFile(filepath.Join(outDir, "main-B9niyWwq.js"))
	if err == nil {
		content := string(mainBytes)
		for _, part := range strings.Split(content, "\"") {
			if strings.HasPrefix(part, "./") && strings.HasSuffix(part, ".js") {
				fn := strings.TrimPrefix(part, "./")
				dest := filepath.Join(outDir, fn)
				if _, err := os.Stat(dest); os.IsNotExist(err) {
					url := "https://farmricochet.com/assets/" + fn
					resp, err := client.Get(url)
					if err == nil && resp.StatusCode == 200 {
						out, _ := os.Create(dest)
						n, _ := io.Copy(out, resp.Body)
						out.Close()
						resp.Body.Close()
						fmt.Printf("Downloaded extra %s (%d bytes)\n", fn, n)
					}
				}
			}
		}
	}
}
