package humanauth

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := "my-refresh-token"

	enc, err := EncryptRefreshTokenForTesting(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	dec, err := DecryptRefreshTokenForTesting(key, enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if dec != plaintext {
		t.Fatalf("got %q, want %q", dec, plaintext)
	}
}

func TestEncryptDifferentEachTime(t *testing.T) {
	key := make([]byte, 32)
	plaintext := "same-token"
	enc1, _ := EncryptRefreshTokenForTesting(key, plaintext)
	enc2, _ := EncryptRefreshTokenForTesting(key, plaintext)
	if enc1 == enc2 {
		t.Fatal("expected different ciphertexts due to random nonce")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 1

	enc, _ := EncryptRefreshTokenForTesting(key1, "token")
	_, err := DecryptRefreshTokenForTesting(key2, enc)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestEncryptBadKey(t *testing.T) {
	_, err := EncryptRefreshTokenForTesting([]byte("short"), "x")
	if err == nil {
		t.Fatal("expected error for short key")
	}
}
