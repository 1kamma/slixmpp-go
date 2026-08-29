package xep

import (
	"github.com/1kamma/slixmpp-go/xmpp"
	"testing"
)

func TestReplyFallbackAndReaction(t *testing.T) {
	m := xmpp.Message{Body: "answer"}
	SetReply(&m, Reply{To: "a@example", ID: "m1"}, "> quoted\n")
	stripped, err := StripFallback(m, ReplyNS)
	if err != nil {
		t.Fatal(err)
	}
	if stripped.Body != "answer" {
		t.Fatalf("%q", stripped.Body)
	}
	SetReactions(&m, "m1", "👍", "👍", "🎉")
	id, reactions, ok := GetReactions(m)
	if !ok || id != "m1" || len(reactions) != 2 {
		t.Fatal(id, reactions)
	}
}
func TestJSON(t *testing.T) {
	m := xmpp.Message{}
	if err := SetJSON(&m, map[string]int{"x": 1}); err != nil {
		t.Fatal(err)
	}
	var v map[string]int
	if err := GetJSON(m, &v); err != nil || v["x"] != 1 {
		t.Fatal(v, err)
	}
}
