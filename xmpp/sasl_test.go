package xmpp

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"
)

func TestPBKDF2Vector(t *testing.T) {
	got := pbkdf2(sha1.New, []byte("password"), []byte("salt"), 1, 20)
	if hex.EncodeToString(got) != "0c60c80f961f0e71f3a9b524af6012062fe037a6" {
		t.Fatal(hex.EncodeToString(got))
	}
}
