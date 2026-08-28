package xmpp

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
)

// StanzaHandler processes a matched stanza.
type StanzaHandler func(context.Context, *Client, Stanza) error

// HandlerOptions controls dispatch.
type HandlerOptions struct {
	Priority    int
	Once, Async bool
}

// Handler is a registered stanza callback.
type Handler struct {
	id          uint64
	Name        string
	Matcher     Matcher
	Callback    StanzaHandler
	Priority    int
	Once, Async bool
}
type handlerRegistry struct {
	mu       sync.RWMutex
	nextID   atomic.Uint64
	handlers []*Handler
}

func (r *handlerRegistry) add(name string, m Matcher, cb StanzaHandler, o HandlerOptions) *Handler {
	if m == nil {
		m = MatchAll
	}
	h := &Handler{id: r.nextID.Add(1), Name: name, Matcher: m, Callback: cb, Priority: o.Priority, Once: o.Once, Async: o.Async}
	r.mu.Lock()
	r.handlers = append(r.handlers, h)
	sort.SliceStable(r.handlers, func(i, j int) bool { return r.handlers[i].Priority < r.handlers[j].Priority })
	r.mu.Unlock()
	return h
}
func (r *handlerRegistry) remove(h *Handler) {
	if h == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, item := range r.handlers {
		if item.id == h.id {
			r.handlers = append(r.handlers[:i], r.handlers[i+1:]...)
			return
		}
	}
}
func (r *handlerRegistry) snapshot() []*Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]*Handler(nil), r.handlers...)
}
