// пакет криптографии
package db_package

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

// Шифруем байты в байты.
func EncryptB2B(bytesToEncrypt, key []byte) (encrypted []byte, err error) {
	// если нет ключа шифрования - что пришло то и ушло
	if len(key) == 0 {
		return bytesToEncrypt, nil
	}
	// MD5 checksum of key, 16 bytes
	keyB16 := md5.Sum(key)
	keyB := keyB16[:]
	//keyB = []byte("0123456789123456")
	// Базовый интерфейс симметричного шифрования — cipher.Block из пакета  https://pkg.go.dev/crypto/cipher
	// Зашифруем помощью алгоритма AES (Advanced Encryption Standard). Это блочный алгоритм, размер блока — 16 байт.
	// NewCipher создает и возвращает новый cipher.Block.
	block, err := aes.NewCipher(keyB)
	if err != nil {
		return nil, err
	}
	// NewGCM returns the given 128-bit, block cipher wrapped in Galois Counter Mode with the standard nonce length.
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// создаём вектор инициализации
	nonce, _ := RandBytes(aesGCM.NonceSize())
	// зашифровываем
	ciphertext := aesGCM.Seal(nonce, nonce, bytesToEncrypt, nil)
	return ciphertext, nil
}

// Расшифровываем байты в байты.
func DecryptB2B(encrypted, key []byte) (decrypted []byte, err error) {
	// если нет ключа шифрования - что пришло то и ушло
	if len(key) == 0 {
		return encrypted, nil
	}
	// MD5 checksum of key, 16 bytes
	keyB16 := md5.Sum(key)
	keyB := keyB16[:]
	//keyB = []byte("0123456789123456")
	block, err := aes.NewCipher(keyB)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aesGCM.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
	// расшифровываем
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func RandBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// подписываем алгоритмом HMAC, используя SHA-256
func MakeHash(prior, data, keyB []byte) []byte {
	h := hmac.New(sha256.New, keyB) // New returns a new HMAC hash using the given hash.Hash type and key.
	h.Write(data)                   // func (hash.Hash) Sum(b []byte) []byte
	dst := h.Sum(prior)             //Sum appends the current hash to b and returns the resulting slice. It does not change the underlying hash state.
	return dst

}
