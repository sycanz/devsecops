package crypto

import "testing"

func TestRoundTrip(t *testing.T) {
	c := New("test-key")
	original := 85000
	encrypted, err := c.Encrypt(original)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := c.Decrypt(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != original {
		t.Fatalf("expected %d, got %d", original, decrypted)
	}
}
