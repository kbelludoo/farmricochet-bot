package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	token := "da9efaeb-4a01-49cd-9918-584cf48eec7f"

	cookieNames := []string{
		"session",
		"farm-ricochet-session",
		"farm-session",
		"connect.sid",
		"token",
		"auth_token",
		"farm-ricochet-api-token",
		"session_token",
		"sid",
		"farm",
	}

	client := &http.Client{}

	// Test 1: Bearer token
	req, _ := http.NewRequest("GET", "https://api.farmricochet.com/api/v1/account", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Origin", "https://farmricochet.com")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err == nil {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("Authorization Bearer: HTTP %d -> %s\n", resp.StatusCode, string(b))
	}

	// Test 2: Cookies
	for _, name := range cookieNames {
		req, _ := http.NewRequest("GET", "https://api.farmricochet.com/api/v1/account", nil)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", name, token))
		req.Header.Set("Origin", "https://farmricochet.com")
		req.Header.Set("User-Agent", "Mozilla/5.0")
		resp, err := client.Do(req)
		if err == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != 401 {
				fmt.Printf("FOUND VALID COOKIE! %s=%s -> HTTP %d: %s\n", name, token, resp.StatusCode, string(b))
			} else {
				fmt.Printf("%s: HTTP %d\n", name, resp.StatusCode)
			}
		}
	}
}
