package xmpp

import "testing"

func TestJID(t *testing.T) {
	jid, err := ParseJID("Alice@EXAMPLE.org/Phone")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := jid.String(), "Alice@example.org/Phone"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if jid.Bare() != "Alice@example.org" {
		t.Fatal(jid.Bare())
	}
	escaped := EscapeLocal("space cadet@example.org")
	if UnescapeLocal(escaped) != "space cadet@example.org" {
		t.Fatal(escaped)
	}
}
