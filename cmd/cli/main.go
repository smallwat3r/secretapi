package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultBaseURL = "https://secret.smallwat3r.com"

type CreateReq struct {
	Secret string `json:"secret"`
	Expiry string `json:"expiry,omitempty"`
}

type CreateRes struct {
	ID        string    `json:"id"`
	Passcode  string    `json:"passcode"`
	ExpiresAt time.Time `json:"expires_at"`
	ReadURL   string    `json:"read_url"`
}

type ReadRes struct {
	Secret string `json:"secret"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	baseURL := os.Getenv("SECRET_API_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	switch os.Args[1] {
	case "create":
		if len(os.Args) != 3 && len(os.Args) != 4 {
			fmt.Fprintf(os.Stderr, "Usage: %s create <secret> [expiry]\n", os.Args[0])
			os.Exit(1)
		}
		secret := os.Args[2]
		expiry := ""
		if len(os.Args) == 4 {
			expiry = os.Args[3]
		}
		createSecret(baseURL, secret, expiry)
	case "read":
		if len(os.Args) != 4 {
			fmt.Fprintf(os.Stderr, "Usage: %s read <url> <passcode>\n", os.Args[0])
			os.Exit(1)
		}
		urlArg := os.Args[2]
		passcode := os.Args[3]
		readSecret(urlArg, passcode)
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("Usage: %s <command> [arguments]\n", os.Args[0])
	fmt.Println("A simple CLI to create and read secrets.")
	fmt.Println("\nCommands:")
	fmt.Println("  create <secret> [expiry] Create a new secret (expiry: 1h, 6h, 1d, 3d)")
	fmt.Println("  read <url> <passcode>    Read a secret")
	fmt.Println("  help                     Show this help message")
	fmt.Println("\nEnvironment variables:")
	fmt.Println("  SECRET_API_URL           Set the base URL for the secret API (default: https://secret.smallwat3r.com)")
}

func createSecret(baseURL, secret, expiry string) {
	reqBody, err := json.Marshal(CreateReq{Secret: secret, Expiry: expiry})
	if err != nil {
		log.Fatalf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(baseURL+"/create", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("failed to create secret: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("failed to create secret: status %d, body: %s", resp.StatusCode, body)
	}

	var createRes CreateRes
	if err := json.NewDecoder(resp.Body).Decode(&createRes); err != nil {
		log.Fatalf("failed to decode response: %v", err)
	}

	fmt.Println("Your secret is ready to share:")
	fmt.Printf("URL: %s\n", createRes.ReadURL)
	fmt.Printf("Passcode: %s\n", createRes.Passcode)
	fmt.Printf("Expires: %s\n", createRes.ExpiresAt.Format(time.RFC1123))
}

func readSecret(rawURL, passcode string) {
	rawURL = strings.TrimRight(rawURL, "/")

	// Force https for the production domain to avoid redirects
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Fatalf("failed to parse URL: %v", err)
	}
	if parsedURL.Scheme == "http" && strings.Contains(parsedURL.Host, "smallwat3r.com") {
		parsedURL.Scheme = "https"
		rawURL = parsedURL.String()
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", rawURL, nil)
	if err != nil {
		log.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-Passcode", passcode)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to read secret: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("failed to read secret: status %d, body: %s", resp.StatusCode, body)
	}

	var readRes ReadRes
	if err := json.NewDecoder(resp.Body).Decode(&readRes); err != nil {
		log.Fatalf("failed to decode response: %v", err)
	}

	fmt.Println(readRes.Secret)
}
