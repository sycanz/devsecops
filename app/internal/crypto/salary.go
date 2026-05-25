package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"

	"crypto/sha256"

	"golang.org/x/crypto/hkdf"
)

type Crypter interface {
	Encrypt(plaintext int) (string, error)
	Decrypt(ciphertext string) (int, error)
}

type crypter struct {
	key []byte
}

func New(keyMaterial string) Crypter {
	// Derive a 32-byte key using HKDF-SHA256
	salt := []byte("employeesalt")
	hkdf := hkdf.New(sha256.New, []byte(keyMaterial), salt, nil)
	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdf, key); err != nil {
		panic(fmt.Sprintf("key derivation failed: %v", err))
	}
	return &crypter{key: key}
}

func (c *crypter) Encrypt(plaintext int) (string, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aead.Seal(nil, nonce, []byte(strconv.Itoa(plaintext)), nil)
	return base64.RawStdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func (c *crypter) Decrypt(encoded string) (int, error) {
	data, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return 0, err
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return 0, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return 0, err
	}
	nonceSize := aead.NonceSize()
	if len(data) < nonceSize {
		return 0, errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return 0, err
	}
	var result int
	n, _ := fmt.Sscanf(string(plaintext), "%d", &result)
	if n != 1 {
		return 0, errors.New("invalid salary format")
	}
	return result, nil
}
