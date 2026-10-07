package identity

import (
	"regexp"
	"testing"
)

func TestNewUUIDReturnsVersionFourIdentifier(t *testing.T) {
	id, err := NewUUID()
	if err != nil {
		t.Fatalf("NewUUID() error = %v", err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(id) {
		t.Fatalf("NewUUID() = %q, not an RFC 4122 version 4 UUID", id)
	}
}
