package xep

import (
	"github.com/saret/slixmpp-go/xmpp"
	"testing"
)

func TestBoBAndForwarded(t *testing.T) {
	bob := NewBoB("text/plain", []byte("hello"))
	parsed, err := ParseBoB(bob.ToNode())
	if err != nil || string(parsed.Data) != "hello" {
		t.Fatal(parsed, err)
	}
	node, err := (Forwarded{Stanza: xmpp.Message{From: "a@example", Body: "x"}}).ToNode()
	if err != nil {
		t.Fatal(err)
	}
	forward, err := ParseForwarded(node)
	if err != nil {
		t.Fatal(err)
	}
	if forward.Stanza.(xmpp.Message).Body != "x" {
		t.Fatal(forward)
	}
}
func TestHash(t *testing.T) {
	h, err := ComputeHash("sha-256", []byte("x"))
	if err != nil || !h.Verify([]byte("x")) || h.Verify([]byte("y")) {
		t.Fatal(err)
	}
}
