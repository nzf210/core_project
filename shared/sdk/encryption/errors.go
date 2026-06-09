package encryption

import "errors"

var (
	ErrInvalidKey       = errors.New("encryption key must be exactly 32 bytes")
	ErrCiphertextShort  = errors.New("ciphertext too short")
)