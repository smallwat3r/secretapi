package utility

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	LowerCryptoParamsForTest(t)

	passcode := "abacus-abdomen-abdominal"
	plaintext := []byte("this is a very secret message")

	encrypted, err := Encrypt(plaintext, passcode)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if len(encrypted) == 0 {
		t.Fatal("Encrypt() returned empty byte slice")
	}

	decrypted, err := Decrypt(encrypted, passcode)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypt() got = %s, want %s", string(decrypted), string(plaintext))
	}
}

func TestDecrypt_WrongPasscode(t *testing.T) {
	LowerCryptoParamsForTest(t)

	passcode := "abacus-abdomen-abdominal"
	wrongPasscode := "abide-abiding-ability"
	plaintext := []byte("this is a very secret message")

	encrypted, err := Encrypt(plaintext, passcode)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	_, err = Decrypt(encrypted, wrongPasscode)
	if err == nil {
		t.Error("Decrypt() with wrong passcode should return an error")
	}
}

func TestDecrypt_InvalidBlob(t *testing.T) {
	passcode := "abacus-abdomen-abdominal"
	tests := []struct {
		name string
		blob []byte
	}{
		{"no prefix", []byte("invalidblob")},
		{"short blob", []byte("v1:short")},
		{"bad base64", []byte("v1:!@#$%^")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Decrypt(tt.blob, passcode); err == nil {
				t.Errorf("Decrypt() with blob '%s' should fail", tt.blob)
			}
		})
	}
}

func TestGeneratePasscode(t *testing.T) {
	t.Run("generates valid passcode format", func(t *testing.T) {
		passcode, err := GeneratePasscode()
		if err != nil {
			t.Fatalf("GeneratePasscode() error = %v", err)
		}

		parts := strings.Split(passcode, "-")
		if len(parts) != 3 {
			t.Errorf("expected 3 dash-separated words, got %d parts: %s",
				len(parts), passcode)
		}

		for i, part := range parts {
			if len(part) == 0 {
				t.Errorf("word %d is empty in passcode: %s", i+1, passcode)
			}
		}
	})

	t.Run("generates unique passcodes", func(t *testing.T) {
		passcodes := make(map[string]bool)
		for i := 0; i < 100; i++ {
			passcode, err := GeneratePasscode()
			if err != nil {
				t.Fatalf("GeneratePasscode() error = %v", err)
			}
			if passcodes[passcode] {
				t.Errorf("duplicate passcode generated: %s", passcode)
			}
			passcodes[passcode] = true
		}
	})

	t.Run("uses words from wordlist", func(t *testing.T) {
		passcode, err := GeneratePasscode()
		if err != nil {
			t.Fatalf("GeneratePasscode() error = %v", err)
		}

		parts := strings.Split(passcode, "-")
		wordSet := make(map[string]bool)
		for _, w := range Wordlist {
			wordSet[w] = true
		}

		for _, part := range parts {
			if !wordSet[part] {
				t.Errorf("word %q not found in wordlist", part)
			}
		}
	})
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	LowerCryptoParamsForTest(t)

	passcode := "abacus-abdomen-abdominal"
	plaintext := []byte("")

	encrypted, err := Encrypt(plaintext, passcode)
	if err != nil {
		t.Fatalf("Encrypt() with empty plaintext error = %v", err)
	}

	decrypted, err := Decrypt(encrypted, passcode)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypt() got = %v, want %v", decrypted, plaintext)
	}
}

func TestEncrypt_LargePlaintext(t *testing.T) {
	LowerCryptoParamsForTest(t)

	passcode := "abacus-abdomen-abdominal"
	// 64KB of data
	plaintext := bytes.Repeat([]byte("a"), 64*1024)

	encrypted, err := Encrypt(plaintext, passcode)
	if err != nil {
		t.Fatalf("Encrypt() with large plaintext error = %v", err)
	}

	decrypted, err := Decrypt(encrypted, passcode)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypt() returned wrong data for large plaintext")
	}
}

func TestEncrypt_BinaryData(t *testing.T) {
	LowerCryptoParamsForTest(t)

	passcode := "abacus-abdomen-abdominal"
	// Binary data with null bytes and special characters
	plaintext := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0x00, 0x7F}

	encrypted, err := Encrypt(plaintext, passcode)
	if err != nil {
		t.Fatalf("Encrypt() with binary data error = %v", err)
	}

	decrypted, err := Decrypt(encrypted, passcode)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypt() got = %v, want %v", decrypted, plaintext)
	}
}

func TestEncrypt_SameInputDifferentOutput(t *testing.T) {
	LowerCryptoParamsForTest(t)

	passcode := "abacus-abdomen-abdominal"
	plaintext := []byte("test message")

	encrypted1, err := Encrypt(plaintext, passcode)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	encrypted2, err := Encrypt(plaintext, passcode)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Due to random salt and nonce, same plaintext should produce different ciphertext
	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("Encrypt() same ciphertext for same plaintext - not random")
	}
}

func TestEncrypt_OutputFormat(t *testing.T) {
	LowerCryptoParamsForTest(t)

	passcode := "abacus-abdomen-abdominal"
	plaintext := []byte("test")

	encrypted, err := Encrypt(plaintext, passcode)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Check output starts with version prefix
	if !strings.HasPrefix(string(encrypted), "v1:") {
		t.Errorf("encrypted should start with 'v1:', got: %s",
			string(encrypted[:10]))
	}
}
