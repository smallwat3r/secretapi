package utility

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	lowerCryptoParamsForTest(t)

	passphrase := "strongpassword123"
	plaintext := []byte("this is a very secret message")

	encrypted, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if len(encrypted) == 0 {
		t.Fatal("Encrypt() returned empty byte slice")
	}

	decrypted, err := Decrypt(encrypted, passphrase)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypt() got = %s, want %s", string(decrypted), string(plaintext))
	}
}

func TestDecrypt_WrongPassphrase(t *testing.T) {
	lowerCryptoParamsForTest(t)

	passphrase := "strongpassword123"
	wrongPassphrase := "wrongpassword456"
	plaintext := []byte("this is a very secret message")

	encrypted, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	_, err = Decrypt(encrypted, wrongPassphrase)
	if err == nil {
		t.Error("Decrypt() with wrong passphrase should have returned an error, but it didn't")
	}
}

func TestDecrypt_InvalidBlob(t *testing.T) {
	passphrase := "strongpassword123"
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
			if _, err := Decrypt(tt.blob, passphrase); err == nil {
				t.Errorf("Decrypt() with blob '%s' should have failed, but it didn't", tt.blob)
			}
		})
	}
}
