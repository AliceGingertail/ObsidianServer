package crypto

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not equal plain password")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Test correct password
	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for correct password")
	}

	// Test incorrect password
	if CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword should return false for incorrect password")
	}
}

func TestHashToken(t *testing.T) {
	token := "test-refresh-token-123"

	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 == "" {
		t.Error("Hash should not be empty")
	}

	if hash1 != hash2 {
		t.Error("Same token should produce same hash")
	}

	differentToken := "different-token"
	differentHash := HashToken(differentToken)

	if hash1 == differentHash {
		t.Error("Different tokens should produce different hashes")
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := "benchmarkpassword"
	
	for i := 0; i < b.N; i++ {
		HashPassword(password)
	}
}

func BenchmarkCheckPassword(b *testing.B) {
	password := "benchmarkpassword"
	hash, _ := HashPassword(password)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckPassword(password, hash)
	}
}
