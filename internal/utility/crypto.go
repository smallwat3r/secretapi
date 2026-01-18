package utility

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

const (
	saltLen  = 16
	nonceLen = 12 // GCM standard
	keyLen   = 32 // AES-256
)

// CryptoConfig holds configuration parameters for cryptographic operations.
type CryptoConfig struct {
	ArgonTime    uint32
	ArgonMemory  uint32
	ArgonThreads uint8
}

// DefaultCryptoConfig returns the default production configuration.
func DefaultCryptoConfig() CryptoConfig {
	return CryptoConfig{
		ArgonTime:    1,
		ArgonMemory:  64 * 1024, // 64 MB
		ArgonThreads: 4,
	}
}

// TestCryptoConfig returns a faster configuration suitable for testing.
func TestCryptoConfig() CryptoConfig {
	return CryptoConfig{
		ArgonTime:    1,
		ArgonMemory:  1024, // 1 MB - faster for tests
		ArgonThreads: 4,
	}
}

// cryptoConfig is the current active configuration.
// It defaults to production settings and can be modified for tests.
// Access is protected by cryptoConfigMu for thread safety.
var (
	cryptoConfig   = DefaultCryptoConfig()
	cryptoConfigMu sync.RWMutex
)

// getCryptoConfig returns a copy of the current crypto configuration.
func getCryptoConfig() CryptoConfig {
	cryptoConfigMu.RLock()
	defer cryptoConfigMu.RUnlock()
	return cryptoConfig
}

// setCryptoConfig sets the crypto configuration. This should only be used in tests.
func setCryptoConfig(cfg CryptoConfig) {
	cryptoConfigMu.Lock()
	defer cryptoConfigMu.Unlock()
	cryptoConfig = cfg
}

func GeneratePasscode() (string, error) {
	var words []string
	for i := 0; i < 3; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(Wordlist))))
		if err != nil {
			return "", err
		}
		words = append(words, Wordlist[n.Int64()])
	}
	return strings.Join(words, "-"), nil
}

func deriveKey(passcode string, salt []byte) []byte {
	cfg := getCryptoConfig()
	return argon2.IDKey(
		[]byte(passcode),
		salt,
		cfg.ArgonTime,
		cfg.ArgonMemory,
		cfg.ArgonThreads,
		keyLen,
	)
}

func Encrypt(plaintext []byte, passcode string) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("salt: %w", err)
	}
	key := deriveKey(passcode, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}

	ct := gcm.Seal(nil, nonce, plaintext, nil)

	// store as base64(salt|nonce|ciphertext) with a "v1:" prefix.
	raw := make([]byte, 0, len(salt)+len(nonce)+len(ct))
	raw = append(raw, salt...)
	raw = append(raw, nonce...)
	raw = append(raw, ct...)

	out := "v1:" + base64.StdEncoding.EncodeToString(raw)
	return []byte(out), nil
}

func Decrypt(blob []byte, passcode string) ([]byte, error) {
	s := string(blob)
	if !strings.HasPrefix(s, "v1:") {
		return nil, errors.New("unsupported format")
	}
	b64 := strings.TrimPrefix(s, "v1:")
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("b64: %w", err)
	}
	if len(raw) < saltLen+nonceLen+1 {
		return nil, errors.New("blob too short")
	}

	salt := raw[:saltLen]
	nonce := raw[saltLen : saltLen+nonceLen]
	ct := raw[saltLen+nonceLen:]

	key := deriveKey(passcode, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, errors.New("auth failed")
	}
	return pt, nil
}
