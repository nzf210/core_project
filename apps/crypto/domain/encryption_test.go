package domain

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := "12345678901234567890123456789012" // 32 bytes
	plaintext := "my-super-secret-api-key"

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if ciphertext == "" {
		t.Fatal("Expected ciphertext to not be empty")
	}

	if ciphertext == plaintext {
		t.Fatal("Expected ciphertext to be different from plaintext")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Expected decrypted text to be %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptInvalidKeyLength(t *testing.T) {
	key := "short-key"
	plaintext := "test"

	_, err := Encrypt(plaintext, key)
	if err != ErrInvalidEncryptionKey {
		t.Errorf("Expected ErrInvalidEncryptionKey, got %v", err)
	}

	_, err = Decrypt("some-ciphertext", key)
	if err != ErrInvalidEncryptionKey {
		t.Errorf("Expected ErrInvalidEncryptionKey, got %v", err)
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	key := "12345678901234567890123456789012" // 32 bytes
	
	// Base64 invalid
	_, err := Decrypt("not-base64-!", key)
	if err == nil {
		t.Error("Expected error for invalid base64")
	}

	// Too short
	_, err = Decrypt("YmFzZTY0c2hvcnQ=", key)
	if err != ErrCiphertextTooShort && err == nil {
		t.Errorf("Expected error for short ciphertext, got %v", err)
	}
}
