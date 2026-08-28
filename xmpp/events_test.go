package xmpp

import (
	"context"
	"testing"
)

func TestEventOrderAndOnce(t *testing.T) {
	bus := NewEventBus()
	var order []int
	bus.On("x", func(context.Context, Event) error { order = append(order, 2); return nil }, EventOptions{Priority: 2})
	bus.On("x", func(context.Context, Event) error { order = append(order, 1); return nil }, EventOptions{Priority: 1, Once: true})
	if err := bus.Emit(context.Background(), "x", nil); err != nil {
		t.Fatal(err)
	}
	_ = bus.Emit(context.Background(), "x", nil)
	want := []int{1, 2, 2}
	if len(order) != len(want) {
		t.Fatal(order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatal(order)
		}
	}
}
