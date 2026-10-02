package crypto

import (
	"bytes"
	"testing"
)

func TestSealOpen(t *testing.T) {
	masterKey := make([]byte, KeySize)
	for i := range masterKey {
		masterKey[i] = byte(i) // Dummy key
	}

	plaintext := []byte("hello world, this is a secret")

	ciphertext, err := Seal(plaintext, masterKey)
	if err != nil {
		t.Fatalf("Seal failed: %v", err)
	}

	decrypted, err := Open(ciphertext, masterKey)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestTamperDetection(t *testing.T) {
	masterKey := make([]byte, KeySize)
	plaintext := []byte("tamper detection test")

	ciphertext, err := Seal(plaintext, masterKey)
	if err != nil {
		t.Fatalf("Seal failed: %v", err)
	}

	// Tamper with the ciphertext (payload)
	tamperedCiphertext := make([]byte, len(ciphertext))
	copy(tamperedCiphertext, ciphertext)
	tamperedCiphertext[len(tamperedCiphertext)-1] ^= 1

	if _, err := Open(tamperedCiphertext, masterKey); err == nil {
		t.Error("expected error on tampered ciphertext, got nil")
	}

	// Tamper with the header
	tamperedHeader := make([]byte, len(ciphertext))
	copy(tamperedHeader, ciphertext)
	tamperedHeader[1] ^= 1 // flip a byte in magic string

	if _, err := Open(tamperedHeader, masterKey); err == nil {
		t.Error("expected error on tampered header, got nil")
	}
}

func TestWrongKey(t *testing.T) {
	masterKey := make([]byte, KeySize)
	wrongKey := make([]byte, KeySize)
	wrongKey[0] = 1

	plaintext := []byte("wrong key test")
	ciphertext, err := Seal(plaintext, masterKey)
	if err != nil {
		t.Fatalf("Seal failed: %v", err)
	}

	if _, err := Open(ciphertext, wrongKey); err == nil {
		t.Error("expected error with wrong key, got nil")
	}
}

func TestKeyCheck(t *testing.T) {
	masterKey := make([]byte, KeySize)

	blob, err := CreateKeyCheck(masterKey)
	if err != nil {
		t.Fatalf("CreateKeyCheck failed: %v", err)
	}

	if !ValidateKeyCheck(blob, masterKey) {
		t.Error("ValidateKeyCheck failed with correct key")
	}

	wrongKey := make([]byte, KeySize)
	wrongKey[0] = 1
	if ValidateKeyCheck(blob, wrongKey) {
		t.Error("ValidateKeyCheck succeeded with wrong key")
	}
}

func TestZeroBytes(t *testing.T) {
	b := []byte{1, 2, 3, 4, 5}
	ZeroBytes(b)
	for i, v := range b {
		if v != 0 {
			t.Errorf("byte at index %d is not 0: %d", i, v)
		}
	}
}

func TestDeriveSubKey(t *testing.T) {
	masterKey := make([]byte, KeySize)
	salt1 := []byte("1234567890123456")
	salt2 := []byte("6543210987654321")

	key1, err := DeriveSubKey(masterKey, salt1, "info")
	if err != nil {
		t.Fatalf("DeriveSubKey failed for salt1: %v", err)
	}

	key2, err := DeriveSubKey(masterKey, salt2, "info")
	if err != nil {
		t.Fatalf("DeriveSubKey failed for salt2: %v", err)
	}

	if bytes.Equal(key1, key2) {
		t.Error("expected different subkeys for different salts")
	}
}

func TestSealRandomness(t *testing.T) {
	masterKey := make([]byte, KeySize)
	plaintext := []byte("hello")

	c1, err := Seal(plaintext, masterKey)
	if err != nil {
		t.Fatalf("Seal 1 failed: %v", err)
	}
	c2, err := Seal(plaintext, masterKey)
	if err != nil {
		t.Fatalf("Seal 2 failed: %v", err)
	}

	if bytes.Equal(c1, c2) {
		t.Error("Seal produced identical ciphertexts for the same input; expected randomness")
	}
}
