package humanauth

import "testing"

func TestEncryptDecryptRefreshToken_RoundTrips(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encrypted, err := encryptRefreshToken(key, "my-refresh-token")
	if err != nil {
		t.Fatalf("encryptRefreshToken() error = %v", err)
	}
	if encrypted == "my-refresh-token" {
		t.Fatal("encryptRefreshToken() returned the plaintext unchanged")
	}

	decrypted, err := decryptRefreshToken(key, encrypted)
	if err != nil {
		t.Fatalf("decryptRefreshToken() error = %v", err)
	}
	if decrypted != "my-refresh-token" {
		t.Errorf("decrypted = %q, want %q", decrypted, "my-refresh-token")
	}
}

func TestDecryptRefreshToken_FailsWithWrongKey(t *testing.T) {
	key := make([]byte, 32)
	wrongKey := make([]byte, 32)
	wrongKey[0] = 1

	encrypted, err := encryptRefreshToken(key, "my-refresh-token")
	if err != nil {
		t.Fatalf("encryptRefreshToken() error = %v", err)
	}

	if _, err := decryptRefreshToken(wrongKey, encrypted); err == nil {
		t.Fatal("decryptRefreshToken() error = nil, want an error when using the wrong key")
	}
}

func TestDecryptRefreshToken_FailsOnTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)

	encrypted, err := encryptRefreshToken(key, "my-refresh-token")
	if err != nil {
		t.Fatalf("encryptRefreshToken() error = %v", err)
	}

	tampered := []byte(encrypted)
	tampered[len(tampered)-1] ^= 0xFF
	if _, err := decryptRefreshToken(key, string(tampered)); err == nil {
		t.Fatal("decryptRefreshToken() error = nil, want an error for tampered ciphertext")
	}
}

func TestEncryptRefreshToken_RejectsWrongKeyLength(t *testing.T) {
	if _, err := encryptRefreshToken([]byte("too-short"), "x"); err == nil {
		t.Fatal("encryptRefreshToken() error = nil, want an error for a non-32-byte key")
	}
}
