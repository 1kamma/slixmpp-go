package xep

import (
	"github.com/1kamma/slixmpp-go/xmpp"
	"testing"
)

func TestMAMResult(t *testing.T) {
	forward, _ := (Forwarded{Stanza: xmpp.Message{Body: "archived"}}).ToNode()
	n := xmpp.NewNode(MAMNS, "result")
	n.SetAttr("queryid", "q")
	n.SetAttr("id", "r")
	n.AddChild(forward)
	got, err := parseMAMResult(n)
	if err != nil || got.ID != "r" || got.Forwarded.Stanza.(xmpp.Message).Body != "archived" {
		t.Fatal(got, err)
	}
}
