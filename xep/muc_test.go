package xep

import (
	"github.com/1kamma/slixmpp-go/xmpp"
	"testing"
)

func TestParseMUCStatus(t *testing.T) {
	x := xmpp.NewNode(MUCUserNS, "x")
	s := xmpp.NewNode(MUCUserNS, "status")
	s.SetAttr("code", "110")
	x.AddChild(s)
	item := xmpp.NewNode(MUCUserNS, "item")
	item.SetAttr("role", "participant")
	x.AddChild(item)
	got, err := ParseMUCStatus(x)
	if err != nil || !containsInt(got.Codes, 110) || got.Item.Role != "participant" {
		t.Fatal(got, err)
	}
}
