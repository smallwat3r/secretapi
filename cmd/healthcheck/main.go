package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func check(port string) error {
	url := fmt.Sprintf("http://localhost:%s/health", port)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"unexpected status code: %d", resp.StatusCode,
		)
	}
	return nil
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := check(port); err != nil {
		os.Exit(1)
	}
}
