package xmpp

import "testing"

func TestNodeRoundTrip(t *testing.T) {
	n := NewNode("urn:test", "root")
	n.SetAttr("id", "1")
	n.AddText("a")
	n.AddChild(NewTextNode("urn:test", "child", "b"))
	n.AddText("c")
	raw, err := n.XML()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseNode([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Name.Space != "urn:test" || parsed.Text() != "ac" || parsed.ChildText("urn:test", "child") != "b" {
		t.Fatalf("%#v %s", parsed, raw)
	}
}
