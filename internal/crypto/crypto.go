package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// Constants defining encryption format and defaults.
const (
	FormatVersion       byte   = 1
	KDFArgon2id         byte   = 1
	DefaultArgonTime    uint32 = 3
	DefaultArgonMemory  uint32 = 65536 // 64 MiB in KiB
	DefaultArgonThreads uint8  = 4
	SaltSize                   = 16
	NonceSize                  = 24 // XChaCha20-Poly1305
	KeySize                    = 32
)

// MagicBytes identifies the file as a Kosha vault.
var MagicBytes = []byte("KOSHAVLT")

// ArgonParams stores the parameters for Argon2id key derivation.
type ArgonParams struct {
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"`
	Threads uint8  `json:"threads"`
}

// DefaultParams returns the default parameters for Argon2id.
func DefaultParams() ArgonParams {
	return ArgonParams{
		Time:    DefaultArgonTime,
		Memory:  DefaultArgonMemory,
		Threads: DefaultArgonThreads,
	}
}

// DeriveMasterKey derives a 32-byte master key from a passphrase and salt using Argon2id.
func DeriveMasterKey(passphrase []byte, salt []byte, params ArgonParams) []byte {
	return argon2.IDKey(passphrase, salt, params.Time, params.Memory, params.Threads, KeySize)
}

// DeriveSubKey derives a 32-byte subkey from a master key using HKDF-SHA256.
func DeriveSubKey(masterKey []byte, fileSalt []byte, info string) ([]byte, error) {
	hkdfReader := hkdf.New(sha256.New, masterKey, fileSalt, []byte(info))
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// GenerateSalt generates 16 random bytes.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// Seal encrypts the plaintext using the masterKey.
// It generates a random salt and nonce, derives a subkey, and uses XChaCha20-Poly1305 AEAD.
func Seal(plaintext []byte, masterKey []byte) ([]byte, error) {
	fileSalt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	subKey, err := DeriveSubKey(masterKey, fileSalt, "kosha-file-encryption")
	if err != nil {
		return nil, err
	}
	defer ZeroBytes(subKey)

	aead, err := chacha20poly1305.NewX(subKey)
	if err != nil {
		return nil, err
	}

	// Build header: 59 bytes total
	// magic(8) + version(1) + kdfID(1) + time(4) + memory(4) + threads(1) + salt(16) + nonce(24)
	header := make([]byte, 59)
	copy(header[0:8], MagicBytes)
	header[8] = FormatVersion
	header[9] = KDFArgon2id
	binary.LittleEndian.PutUint32(header[10:14], DefaultArgonTime)
	binary.LittleEndian.PutUint32(header[14:18], DefaultArgonMemory)
	header[18] = DefaultArgonThreads
	copy(header[19:35], fileSalt)
	copy(header[35:59], nonce)

	ciphertext := aead.Seal(nil, nonce, plaintext, header)

	result := make([]byte, len(header)+len(ciphertext))
	copy(result[:len(header)], header)
	copy(result[len(header):], ciphertext)

	return result, nil
}

// Open decrypts the data using the masterKey, after validating the header.
func Open(data []byte, masterKey []byte) ([]byte, error) {
	if len(data) < 59+16 { // 59 byte header + 16 byte Poly1305 tag minimum
		return nil, errors.New("data too short to be a valid encrypted payload")
	}

	header := data[:59]
	magic := header[0:8]
	if !bytes.Equal(magic, MagicBytes) {
		return nil, errors.New("invalid magic bytes, not a kosha vault")
	}

	version := header[8]
	if version != FormatVersion {
		return nil, errors.New("unsupported format version")
	}

	kdfID := header[9]
	if kdfID != KDFArgon2id {
		return nil, errors.New("unsupported KDF ID")
	}

	// Extract salt and nonce
	fileSalt := header[19:35]
	nonce := header[35:59]

	subKey, err := DeriveSubKey(masterKey, fileSalt, "kosha-file-encryption")
	if err != nil {
		return nil, err
	}
	defer ZeroBytes(subKey)

	aead, err := chacha20poly1305.NewX(subKey)
	if err != nil {
		return nil, err
	}

	ciphertext := data[59:]
	plaintext, err := aead.Open(nil, nonce, ciphertext, header)
	if err != nil {
		return nil, errors.New("authentication failed or incorrect key: " + err.Error())
	}

	return plaintext, nil
}

// ZeroBytes overwrites the byte slice with zeros for memory safety.
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

const keyCheckString = "KOSHA_KEY_CHECK_OK"

// CreateKeyCheck creates an encrypted blob verifying the master key.
func CreateKeyCheck(masterKey []byte) ([]byte, error) {
	return Seal([]byte(keyCheckString), masterKey)
}

// ValidateKeyCheck validates whether the master key can decrypt the key check blob successfully.
func ValidateKeyCheck(blob []byte, masterKey []byte) bool {
	plaintext, err := Open(blob, masterKey)
	if err != nil {
		return false
	}
	return string(plaintext) == keyCheckString
}
