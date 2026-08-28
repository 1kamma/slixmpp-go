package xmpp

import (
	"context"
	"testing"
)

func TestAPIResolution(t *testing.T) {
	r := NewAPIRegistry()
	_ = r.Register("x", "get", "", "", func(context.Context, APICall) (any, error) { return "global", nil })
	_ = r.Register("x", "get", "a@example", "", func(context.Context, APICall) (any, error) { return "jid", nil })
	v, err := r.Run(context.Background(), APICall{Category: "x", Operation: "get", JID: "a@example"})
	if err != nil || v != "jid" {
		t.Fatal(v, err)
	}
}
