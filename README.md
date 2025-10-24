# SecretAPI

SecretAPI is a lightweight, self-hostable API for securely sharing short-lived secrets such as passwords, tokens, or messages. Each secret is encrypted with a user-provided passphrase and stored temporarily in Redis with a chosen expiry time (1 hour, 6 hours, 1 day, or 3 days).

A secret can only be read once with the correct passphrase. After that, it is deleted automatically. If a wrong passphrase is used too many times, the secret is permanently removed.

## How it works

When a secret is created:

1. A plaintext message and passphrase are sent to the `/create` endpoint.
2. A unique salt (16 bytes) is generated, and a 256-bit encryption key is derived from the passphrase using the Argon2id key derivation function. Argon2id is designed to be resistant to brute-force and GPU attacks.
3. The message is encrypted using AES-256 in Galois/Counter Mode (GCM), which ensures both confidentiality and integrity.
4. The salt, nonce, and ciphertext are combined and Base64-encoded for safe storage as a single string.
5. The encoded blob is stored in Redis under a unique UUID key, with an expiry time set according to user choice.

When someone retrieves the secret through `/read/{id}`, the same passphrase must be supplied. The service:
- Fetches the encrypted blob.
- Extracts the salt and nonce.
- Recreates the encryption key using Argon2id.
- Decrypts the ciphertext using AES-GCM.

If the passphrase matches, the decrypted secret is returned and deleted immediately from Redis. If the passphrase is wrong three times, the secret is also deleted.

## Running SecretAPI

### Prerequisites
- Docker and Docker Compose  
- (Optional) Go 1.22+ if building manually

### Quick start with Docker Compose
Clone the repository and run:

    docker compose up --build

The service will be available at `http://localhost:8080`.

Check health:

    curl http://localhost:8080/health

### Manual build and run

    make build
    ./secretapi

Environment variables:

    PORT=8080
    REDIS_ADDR=localhost:6379
    REDIS_PASSWORD=
    REDIS_DB=0

## API Usage

### Create a secret

Endpoint:
    POST /create

Body:
    {
      "secret": "My login password is Hunter2!",
      "passphrase": "Secure123",
      "expiry": "1h"
    }

The passphrase must be at least 8 characters long and include at least one letter and one digit.

Response:
    {
      "id": "d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33",
      "expires_at": "2025-10-24T16:00:00Z"
    }

Example:
    curl -X POST http://localhost:8080/create \
      -H "Content-Type: application/json" \
      -d '{"secret":"This is top secret","passphrase":"Secret123","expiry":"1h"}'

### Read a secret

Endpoint:
    POST /read/{id}

Body:
    {
      "passphrase": "Secret123"
    }

Response:
    {
      "secret": "This is top secret"
    }

Example:
    curl -X POST http://localhost:8080/read/d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33 \
      -H "Content-Type: application/json" \
      -d '{"passphrase":"Secret123"}'

## Hosting SecretAPI

You can host SecretAPI on any server or container platform that supports Docker.  
For production deployments:
1. Use HTTPS through a reverse proxy like Nginx or Caddy.  
2. Protect access to Redis with a password or private network.  

## Security notes

- Encryption: AES-256-GCM.  
- Key derivation: [Argon2id](https://pkg.go.dev/golang.org/x/crypto/argon2#hdr-Argon2id).  
- Ephemerality: Secrets expire automatically and are deleted after reading.  
- Passphrase validation: At least 8 characters and 1 digit.  
- Stateless: The API stores no passphrases, only encrypted data in Redis.

SecretAPI is designed to minimize exposure — even the host server cannot decrypt stored secrets without the user's passphrase.

## License

SecretAPI is open source under the MIT License.
