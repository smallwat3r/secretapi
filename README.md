# SecretAPI

SecretAPI is a lightweight, self-hostable API for securely sharing short-lived secrets such as passwords, tokens, or messages. Each secret is encrypted with a server-generated passcode and stored temporarily in Redis with a chosen expiry time (1 hour, 6 hours, 1 day, or 3 days).

A secret can only be read once with the correct passcode. After that, it is deleted automatically. If a wrong passcode is used too many times, the secret is permanently removed.

## How it works

When a secret is created:

1. A plaintext message is sent to the `/create` endpoint.
2. The server generates a passcode by combining three random words from a word list (e.g., `shore-outdoors-letter`).
3. A unique salt (16 bytes) is generated, and a 256-bit encryption key is derived from the passcode using the Argon2id key derivation function.
4. The message is encrypted using AES-256 in Galois/Counter Mode (GCM).
5. The salt, nonce, and ciphertext are combined and Base64-encoded for safe storage as a single string.
6. The encoded blob is stored in Redis under a unique UUID key, with an expiry time set according to user choice.
7. The secret's ID and the generated passcode are returned to the user.

When someone retrieves the secret through `POST /read/{id}/`, the service:
- Fetches the encrypted blob.
- Extracts the salt and nonce.
- Recreates the encryption key using Argon2id from the passcode in the `X-Passcode` header.
- Decrypts the ciphertext using AES-GCM.

If the passcode matches, the decrypted secret is returned and deleted immediately from Redis. If the passcode is wrong three times, the secret is also deleted.

## Security Warning: Use HTTPS in Production

When a secret is created, the server returns the generated passcode in the response. To protect this passcode from being intercepted, it is **critical** that you use this API over an **HTTPS** connection in any production or real-world environment. Without HTTPS, both the initial message when creating the secret and the passcode required to read it can be intercepted.

Without HTTPS, the response containing the passcode will be sent in plaintext, allowing anyone on the network to see it and decrypt your secret.

Always terminate TLS/SSL and serve the API over HTTPS, for example, by using a reverse proxy like Nginx or Caddy.

## Running SecretAPI

### Prerequisites
- Docker and Docker Compose  
- (Optional) Go 1.24 if building manually

### Quick start with Docker Compose
Clone the repository and run:

    docker compose up --build

The service will be available at `http://localhost:8080/`.

### Manual build and run

    make build
    ./secretapi

Environment variables:

    PORT=8080
    REDIS_URL=redis://localhost:6379/0

## Usage

You can interact with SecretAPI through the web interface or the REST API.

### Web UI

- **Create a secret**: Navigate to the root URL to write a secret, and generate a shareable link with a auto-generated passcode.
- **Read a secret**: Browse to the generated URL after creating a secret. You will be prompted to enter the passcode to view the message.

### API Usage

#### Create a secret

Endpoint:

    POST /create/

Example:

    curl -X POST http://localhost:8080/create/ \
      -H "Content-Type: application/json" \
      -d '{"secret":"This is top secret","expiry":"1h"}'

Response:

    {
        "id": "d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33",
        "passcode": "lemon-nemesis-onshore",
        "expires_at": "2025-10-24T16:00:00Z",
        "read_url": "http://localhost:8080/read/d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33/"
    }

#### Read a secret

Endpoint:

    POST /read/{id}/

Headers:

- `X-Passcode`: The passcode for the secret.

Example:

    curl -X POST http://localhost:8080/read/d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33/ \
      -H "X-Passcode: lemon-nemesis-onshore"

Response:

    {
        "secret": "This is top secret"
    }


## Hosting SecretAPI

You can host SecretAPI on any server or container platform that supports Docker.  
For production deployments:
1. Use HTTPS through a reverse proxy like Nginx or Caddy.  
2. Protect access to Redis with a password or private network.  

## Security notes

- Encryption: AES-256-GCM.  
- Key derivation: [Argon2id](https://pkg.go.dev/golang.org/x/crypto/argon2#hdr-Argon2id).  
- Ephemerality: Secrets expire automatically and are deleted after reading.  
- Passcode: A memorable passcode is generated on the server for each secret by combining three random words (e.g., `word1-word2-word3`).
- The passcode is never stored: The server only retains the encrypted secret and never stores the passcode.
- Stateless: The API stores no passcodes, only encrypted data in Redis.

SecretAPI is designed to minimize exposure, even the host server cannot decrypt stored secrets without the user's passcode.

## License

SecretAPI is open source under the MIT License.
