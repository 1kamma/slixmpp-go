package xmpp

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Event is a named notification emitted by the client or a plugin.
type Event struct {
	Name string
	Data any
	At   time.Time
}

// EventHandler receives an event.
type EventHandler func(context.Context, Event) error

// EventOptions controls event dispatch.
type EventOptions struct {
	Priority    int
	Once, Async bool
}

// Subscription identifies a registered event handler.
type Subscription struct {
	id          uint64
	name        string
	priority    int
	once, async bool
	handler     EventHandler
}

// EventBus is a concurrency-safe named event dispatcher.
type EventBus struct {
	mu       sync.RWMutex
	nextID   atomic.Uint64
	handlers map[string][]*Subscription
}

// NewEventBus creates an empty event bus.
func NewEventBus() *EventBus { return &EventBus{handlers: make(map[string][]*Subscription)} }

// On registers a handler for name. The special name * receives every event.
func (b *EventBus) On(name string, handler EventHandler, options ...EventOptions) *Subscription {
	if handler == nil {
		panic("xmpp: nil event handler")
	}
	var o EventOptions
	if len(options) > 0 {
		o = options[0]
	}
	s := &Subscription{id: b.nextID.Add(1), name: name, priority: o.Priority, once: o.Once, async: o.Async, handler: handler}
	b.mu.Lock()
	b.handlers[name] = append(b.handlers[name], s)
	sort.SliceStable(b.handlers[name], func(i, j int) bool { return b.handlers[name][i].priority < b.handlers[name][j].priority })
	b.mu.Unlock()
	return s
}

// Once registers a one-shot handler.
func (b *EventBus) Once(name string, handler EventHandler) *Subscription {
	return b.On(name, handler, EventOptions{Once: true})
}

// Off removes a subscription.
func (b *EventBus) Off(s *Subscription) {
	if s == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	items := b.handlers[s.name]
	for i, item := range items {
		if item.id == s.id {
			items = append(items[:i], items[i+1:]...)
			break
		}
	}
	if len(items) == 0 {
		delete(b.handlers, s.name)
	} else {
		b.handlers[s.name] = items
	}
}

// Emit invokes synchronous handlers in priority order and joins their errors.
func (b *EventBus) Emit(ctx context.Context, name string, data any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	b.mu.RLock()
	items := append([]*Subscription(nil), b.handlers[name]...)
	items = append(items, b.handlers["*"]...)
	b.mu.RUnlock()
	sort.SliceStable(items, func(i, j int) bool { return items[i].priority < items[j].priority })
	event := Event{Name: name, Data: data, At: time.Now()}
	var errs []error
	for _, item := range items {
		if item.once {
			b.Off(item)
		}
		if item.async {
			go func(s *Subscription) { _ = s.handler(ctx, event) }(item)
			continue
		}
		if err := item.handler(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf("event %q: %w", name, err))
		}
	}
	return joinErrors(errs)
}
func joinErrors(errs []error) error {
	filtered := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			filtered = append(filtered, err)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return multiError(filtered)
}

type multiError []error

func (m multiError) Error() string   { return fmt.Sprintf("%d errors; first: %v", len(m), m[0]) }
func (m multiError) Unwrap() []error { return []error(m) }
