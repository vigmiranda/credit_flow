package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"strings"
)

type Cipher struct {
	gcm cipher.AEAD
}

func NewCipher(secret string) (*Cipher, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("encryption key is required")
	}

	sum := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Cipher{gcm: gcm}, nil
}

func (c *Cipher) Encrypt(value string) ([]byte, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := c.gcm.Seal(nil, nonce, []byte(value), nil)
	return append(nonce, ciphertext...), nil
}

func MaskRecipient(channel, recipient string) string {
	switch strings.TrimSpace(strings.ToLower(channel)) {
	case "email":
		return maskEmail(recipient)
	default:
		return maskLastFour(recipient)
	}
}

func maskEmail(value string) string {
	trimmed := strings.TrimSpace(value)
	parts := strings.Split(trimmed, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return maskLastFour(trimmed)
	}

	local := parts[0]
	if len(local) == 1 {
		return "*" + "@" + parts[1]
	}

	return local[:1] + strings.Repeat("*", len(local)-1) + "@" + parts[1]
}

func maskLastFour(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 4 {
		return trimmed
	}

	return strings.Repeat("*", len(trimmed)-4) + trimmed[len(trimmed)-4:]
}
