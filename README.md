# SecretAPI

SecretAPI is a lightweight (image <4MB on [Docker Hub](https://hub.docker.com/repository/docker/smallwat3r/secretapi/general)), self-hostable API for securely sharing short-lived secrets such as passwords, tokens, or messages. Each secret is encrypted with a server-generated passcode and stored temporarily in Redis with a chosen expiry time (1 hour, 6 hours, 1 day, or 3 days).

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

When someone retrieves the secret through `POST /read/{id}`, the service:
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
- (Optional) Go 1.24 and Node if building manually

### Quick start with Docker Compose
Clone the repository and run:

    docker compose up --build

The service will be available at `http://localhost:8080`.

### Manual build and run

    make build
    ./secretapi

Environment variables:

    PORT=8080
    REDIS_URL=redis://localhost:6379/0

## Usage

You can interact with SecretAPI through the web interface, a command-line client, or the REST API.

### Web UI

- **Create a secret**: Navigate to the root URL to write a secret and generate a shareable link with an auto-generated passcode.
- **Read a secret**: Browse to the generated URL. You will be prompted to enter the passcode to view the message.

### CLI Usage

A command-line client (`secret-cli`) is provided for easy terminal-based interaction.

#### Installation

**1. Using Go**

This command will build and install the binary to your Go bin path (`$GOPATH/bin` or `$HOME/go/bin`).

```bash
go install github.com/smallwat3r/secretapi/cmd/cli@latest
# the binary is named 'cli', so rename it for consistency:
mv "$(go env GOPATH)/bin/cli" "$(go env GOPATH)/bin/secret-cli"
```

**2. Build from Source**

If you have cloned the repository, you can build and install it using the `Makefile`:

```bash
make build-cli
# the binary is named 'secretcli', so move and rename it:
sudo mv secretcli /usr/local/bin/secret-cli
```

#### Configuration

By default, the CLI uses `https://secret.smallwat3r.com` as the API server. You can specify a different server by setting the `SECRET_API_URL` environment variable.

Example:
```bash
export SECRET_API_URL="http://<my_domain>"
secret-cli create "a secret value"
```

#### Create a secret

    secret-cli create "<your-secret>" [expiry]

Example:
```bash
$ secret-cli create "This is top secret" 1h
Your secret is ready to share:
URL: http://localhost:8080/read/d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33
Passcode: lemon-nemesis-onshore
Expires: Fri, 24 Oct 2025 16:00:00 UTC
```

**Security Warning:** Your secret may be stored in your shell's history file. To prevent this, you can either:

1.  **Use your shell's history ignore feature.** If your shell is configured with `HISTCONTROL=ignorespace` (Bash) or `setopt HIST_IGNORE_SPACE` (Zsh), you can prefix the command with a space to prevent it from being saved.
    ```bash
     secret-cli create "This is top secret"
    ```

2.  **Read the secret from a file.** You can then securely delete the file. The `shred` command overwrites the file to hide its contents, and the `-u` option deletes it afterward.
    ```bash
    secret-cli create "$(<my_secret.txt)"
    shred -u my_secret.txt
    ```

#### Read a secret

    secret-cli read <url> <passcode>

Example:
```bash
$ secret-cli read http://localhost:8080/read/d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33 lemon-nemesis-onshore
This is top secret
```

### API Usage

#### Create a secret

- **Endpoint**: `POST /create`
- **Body**: `{"secret": "...", "expiry": "1h"}`

Example response:
```json
{
    "id": "d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33",
    "passcode": "lemon-nemesis-onshore",
    "expires_at": "2025-10-24T16:00:00Z",
    "read_url": "http://localhost:8080/read/d47ef7c1-4a3b-412f-b6ab-5c25b2b68d33"
}
```

#### Read a secret

- **Endpoint**: `POST /read/{id}`
- **Header**: `X-Passcode: <passcode>`

Example response:
```json
{"secret": "This is top secret"}
```


## Hosting SecretAPI

You can host SecretAPI on any server or container platform that supports Docker.  
For production deployments:
1. Use HTTPS through a reverse proxy like Nginx or Caddy.  
2. Protect access to Redis with a password or private network.  

## Security notes

- Encryption: AES-256-GCM.  
- Key derivation: [Argon2id](https://pkg.go.dev/golang.org/x/crypto/argon2#hdr-Argon2id).  
- Ephemerality: Secrets expire automatically and are deleted after reading or too many read attempts.  
- Passcode: A memorable passcode is generated on the server for each secret by combining three random words (e.g., `word1-word2-word3`). With a word list of 7,775 words, this results in over 470 billion possible passcodes (7,775³), making it computationally infeasible to guess, also the secret gets deleted after 3 wrongs read attempts.
- Stateless: The API stores no passcodes, only encrypted data in Redis.

SecretAPI is designed to minimize exposure, even the host server cannot decrypt stored secrets without the user's passcode.

## License

SecretAPI is open source under the MIT License.
