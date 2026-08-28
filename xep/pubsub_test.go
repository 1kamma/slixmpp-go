package xep

import (
	"github.com/saret/slixmpp-go/xmpp"
	"testing"
)

func TestParsePubSubEvent(t *testing.T) {
	event := xmpp.NewNode(PubSubEventNS, "event")
	items := xmpp.NewNode(PubSubEventNS, "items")
	items.SetAttr("node", "n")
	item := xmpp.NewNode(PubSubEventNS, "item")
	item.SetAttr("id", "1")
	item.AddChild(xmpp.NewTextNode("urn:test", "v", "x"))
	items.AddChild(item)
	event.AddChild(items)
	got, err := parsePubSubEvent(event)
	if err != nil || got.Node != "n" || len(got.Items) != 1 {
		t.Fatal(got, err)
	}
}
