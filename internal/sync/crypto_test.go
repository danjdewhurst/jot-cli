package sync

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	passphrase := "test-passphrase-1234"
	plaintext := []byte("hello, this is a secret note")

	ciphertext, err := Encrypt(passphrase, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Ciphertext should differ from plaintext.
	if string(ciphertext) == string(plaintext) {
		t.Error("ciphertext should differ from plaintext")
	}

	got, err := Decrypt(passphrase, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(got) != string(plaintext) {
		t.Errorf("Decrypt = %q, want %q", got, plaintext)
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	ciphertext, err := Encrypt("correct-passphrase", []byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = Decrypt("wrong-passphrase", ciphertext)
	if err == nil {
		t.Error("Decrypt with wrong passphrase should return an error")
	}
}

func TestEncryptDecryptEmptyInput(t *testing.T) {
	passphrase := "test-passphrase"

	ciphertext, err := Encrypt(passphrase, []byte{})
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}

	got, err := Decrypt(passphrase, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("Decrypt = %q, want empty", got)
	}
}

func TestEncryptDecryptLargeInput(t *testing.T) {
	passphrase := "test-passphrase"
	plaintext := make([]byte, 256*1024) // 256KB
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	ciphertext, err := Encrypt(passphrase, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := Decrypt(passphrase, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if len(got) != len(plaintext) {
		t.Fatalf("Decrypt length = %d, want %d", len(got), len(plaintext))
	}
	for i := range got {
		if got[i] != plaintext[i] {
			t.Errorf("byte %d: got %d, want %d", i, got[i], plaintext[i])
			break
		}
	}
}
