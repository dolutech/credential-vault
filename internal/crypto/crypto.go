// Package crypto provides AES-256-GCM authenticated encryption with Argon2id
// key derivation for the credential vault.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// KDFParams holds the Argon2id parameters used to derive the encryption key.
type KDFParams struct {
	Algorithm  string `json:"algorithm"`
	Salt       string `json:"salt"`       // base64 encoded
	Memory     uint32 `json:"memory"`    // in KB (65536 = 64 MB)
	Iterations uint32 `json:"iterations"` // number of passes
	Parallelism uint8  `json:"parallelism"`
}

// DefaultKDFParams returns the recommended Argon2id parameters.
// memory=64MB, iterations=3, parallelism=2 — provides strong resistance
// against GPU/ASIC brute-force attacks.
func DefaultKDFParams() KDFParams {
	return KDFParams{
		Algorithm:  "argon2id",
		Memory:     64 * 1024,
		Iterations:  3,
		Parallelism: 2,
	}
}

// GenerateSalt generates a cryptographically random 16-byte salt.
func GenerateSalt() (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

// DeriveKey derives a 32-byte (AES-256) key from a master password and salt
// using Argon2id.
func DeriveKey(masterPassword string, params KDFParams) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(params.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	// argon2.IDKey(password, salt, time, memory, parallelism, keyLen)
	key := argon2.IDKey(
		[]byte(masterPassword),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		32, // AES-256 key length
	)

	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with the provided key.
// Returns base64-encoded ciphertext (nonce prepended to encrypted data).
func Encrypt(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends the authentication tag to the ciphertext.
	// We prepend the nonce so Decrypt can retrieve it.
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded AES-256-GCM ciphertext.
func Decrypt(key []byte, ciphertextB64 string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:gcm.NonceSize()]
	encrypted := ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt (wrong password or corrupted data): %w", err)
	}

	return plaintext, nil
}