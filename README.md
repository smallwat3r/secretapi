# SecretAPI

SecretAPI is a lightweight, self-hostable API for securely sharing short-lived secrets such as passwords, tokens, or messages. Each secret is encrypted with a server-generated passcode and stored temporarily in Redis with a chosen expiry time (1 hour, 6 hours, 1 day, or 3 days).

A secret can only be read once with the correct passcode. After that, it is deleted automatically. If a wrong passcode is used too many times, the secret is permanently removed.

## How it works

When a secret is created:

1. A plaintext message is sent to the `/create` endpoint.
2. The server generates a random, high-entropy passcode (32 bytes, base64 encoded).
3. A unique salt (16 bytes) is generated, and a 256-bit encryption key is derived from the passcode using the Argon2id key derivation function.
4. The message is encrypted using AES-256 in Galois/Counter Mode (GCM).
5. The salt, nonce, and ciphertext are combined and Base64-encoded for safe storage as a single string.
6. The encoded blob is stored in Redis under a unique UUID key, with an expiry time set according to user choice.
7. The secret's ID and the generated passcode are returned to the user.

When someone retrieves the secret through `/read/{id}/{passcode}/`, the service:
- Fetches the encrypted blob.
- Extracts the salt and nonce.
- Recreates the encryption key using Argon2id.
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

Check health:

    curl http://localhost:8080/health/

### Manual build and run

    make build
    ./secretapi

Environment variables:

    PORT=8080
    REDIS_URL=redis://localhost:6379/0

## API Usage

### Create a secret

Endpoint:

    POST /create/

Body:

    {
        "secret": "My login password is Hunter2!",
        "expiry": "1h"
    }

Response:

    {
        "id": "d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33",
        "passcode": "q5m6rX-WhoO9muvwCwGXxc3vpL_K4lGo_8RKzNlX4CQ",
        "expires_at": "2025-10-24T16:00:00Z",
        "read_url": "http://localhost:8080/read/d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33/q5m6rX-WhoO9muvwCwGXxc3vpL_K4lGo_8RKzNlX4CQ/?format=plain"
    }

Example:

    curl -X POST http://localhost:8080/create/ \
      -H "Content-Type: application/json" \
      -d '{"secret":"This is top secret","expiry":"1h"}'

### Read a secret

Endpoint:

    GET /read/{id}/{passcode}/

Query parameters:

- `format`: `json` (default) or `plain`.

If `format` is `plain`, the API returns only the secret as a plaintext string.

Response (`json`):

    {
        "secret": "This is top secret"
    }

Response (`plain`):

    This is top secret

Example:

    curl http://localhost:8080/read/d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33/q5m6rX-WhoO9muvwCwGXxc3vpL_K4lGo_8RKzNlX4CQ/?format=plain


## Hosting SecretAPI

You can host SecretAPI on any server or container platform that supports Docker.  
For production deployments:
1. Use HTTPS through a reverse proxy like Nginx or Caddy.  
2. Protect access to Redis with a password or private network.  

## Security notes

- Encryption: AES-256-GCM.  
- Key derivation: [Argon2id](https://pkg.go.dev/golang.org/x/crypto/argon2#hdr-Argon2id).  
- Ephemerality: Secrets expire automatically and are deleted after reading.  
- Passcode: A random, high-entropy secret (32 bytes, base64 encoded) is generated on the server for each secret.
- The passcode is never stored: The server only retains the encrypted secret and never stores the passcode.
- Stateless: The API stores no passcodes, only encrypted data in Redis.

SecretAPI is designed to minimize exposure, even the host server cannot decrypt stored secrets without the user's passcode.

## License

SecretAPI is open source under the MIT License.
