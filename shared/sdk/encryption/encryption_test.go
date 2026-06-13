package encryption

import (
	"strings"
	"testing"
)

const validKey = "abcdefghijklmnop1234567890ABCDEF" // 32 bytes

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plaintext := "081234567890"
	ciphertext, err := Encrypt(plaintext, validKey)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if ciphertext == "" || ciphertext == plaintext {
		t.Fatal("ciphertext should be different from plaintext")
	}
	decrypted, err := Decrypt(ciphertext, validKey)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptDecrypt_LongInput(t *testing.T) {
	plaintext := strings.Repeat("HelloWorld", 100)
	ciphertext, err := Encrypt(plaintext, validKey)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	decrypted, err := Decrypt(ciphertext, validKey)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncrypt_InvalidKey(t *testing.T) {
	_, err := Encrypt("hello", "short")
	if err != ErrInvalidKey {
		t.Fatalf("expected ErrInvalidKey, got %v", err)
	}
}

func TestDecrypt_InvalidKey(t *testing.T) {
	ciphertext, _ := Encrypt("hello", validKey)
	_, err := Decrypt(ciphertext, "short")
	if err != ErrInvalidKey {
		t.Fatalf("expected ErrInvalidKey, got %v", err)
	}
}

func TestDecrypt_ShortCiphertext(t *testing.T) {
	_, err := Decrypt("too-short", validKey)
	if err == nil {
		t.Fatal("expected error for short ciphertext")
	}
}

func TestEncrypt_DeterministicOutputIsDifferent(t *testing.T) {
	// Each Encrypt call should produce different output due to random nonce
	c1, _ := Encrypt("hello", validKey)
	c2, _ := Encrypt("hello", validKey)
	if c1 == c2 {
		t.Fatal("two encryptions of same plaintext should produce different ciphertext (random nonce)")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	otherKey := "1234567890ABCDEFabcdefghijklmnop"
	ciphertext, _ := Encrypt("sensitive data", validKey)
	_, err := Decrypt(ciphertext, otherKey)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	ciphertext, err := Encrypt("", validKey)
	if err != nil {
		t.Fatalf("Encrypt empty string failed: %v", err)
	}
	decrypted, err := Decrypt(ciphertext, validKey)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != "" {
		t.Fatalf("expected empty string, got %q", decrypted)
	}
}

func TestEncrypt_KeyNot32Bytes(t *testing.T) {
	_, err := Encrypt("hello", "this-is-31-bytes-key!!!!!!!!!")
	if err != ErrInvalidKey {
		t.Fatalf("expected ErrInvalidKey, got %v", err)
	}
	_, err = Encrypt("hello", "this-is-33-bytes-key!!!!!!!!!!!")
	if err != ErrInvalidKey {
		t.Fatalf("expected ErrInvalidKey, got %v", err)
	}
}
