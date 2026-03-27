package security

import "testing"

func TestEncryptProducesCiphertext(t *testing.T) {
	crypto, err := NewCipher("credit-flow-notification-test-key")
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}

	encrypted, err := crypto.Encrypt("maria@example.com")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if len(encrypted) == 0 {
		t.Fatal("expected encrypted payload")
	}
	if string(encrypted) == "maria@example.com" {
		t.Fatal("expected ciphertext to differ from plaintext")
	}
}

func TestMaskRecipientForEmail(t *testing.T) {
	masked := MaskRecipient("email", "maria@example.com")
	if masked != "m****@example.com" {
		t.Fatalf("unexpected masked value %s", masked)
	}
}
