package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	val := "da9efaeb-4a01-49cd-9918-584cf48eec7f"

	cookieNames := []string{
		"session",
		"session_id",
		"sessionId",
		"farm_session",
		"farm-session",
		"farm_session_id",
		"farm-session-id",
		"farmricochet_session",
		"farm-ricochet-session",
		"farmricochet",
		"farm",
		"jwt",
		"access_token",
		"auth",
		"authorization",
		"id",
		"user_id",
		"userId",
		"account_id",
		"accountId",
		"__Secure-session",
		"__Host-session",
		"__Secure-farm-session",
		"app_session",
		"token",
	}

	client := &http.Client{}

	for _, name := range cookieNames {
		req, _ := http.NewRequest("GET", "https://api.farmricochet.com/api/v1/bootstrap", nil)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", name, val))
		req.Header.Set("Origin", "https://farmricochet.com")
		req.Header.Set("Referer", "https://farmricochet.com/play")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
		resp, err := client.Do(req)
		if err == nil {
			if resp.StatusCode != 401 {
				b, _ := io.ReadAll(resp.Body)
				fmt.Printf("HIT! %s: HTTP %d -> %s\n", name, resp.StatusCode, string(b))
				resp.Body.Close()
				return
			}
			resp.Body.Close()
		}
	}
	fmt.Println("None of the standard cookie names matched directly.")
}
