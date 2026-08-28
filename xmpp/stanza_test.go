package xmpp

import "testing"

func TestMessageRoundTrip(t *testing.T) {
	m := Message{From: "a@example/a", To: "b@example/b", ID: "m1", Type: MessageChat, Body: "hello", Extensions: []Node{NewNode("urn:test", "x")}}
	raw, err := StanzaXML(m)
	if err != nil {
		t.Fatal(err)
	}
	stanza, err := ParseStanza(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := stanza.(Message)
	if got.Body != "hello" || got.Extension("urn:test", "x") == nil {
		t.Fatalf("%s %#v", raw, got)
	}
}
func TestIQRequiresID(t *testing.T) {
	if _, err := StanzaXML(IQ{Type: IQGet}); err == nil {
		t.Fatal("expected error")
	}
}
