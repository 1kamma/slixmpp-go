package xmpp

import (
	"context"
	"fmt"
	"sync"
)

// APICall is a Slixmpp-style internal API invocation.
type APICall struct {
	Category, Operation, JID, Node, From string
	Args                                 any
}

// APIHandler services an internal operation.
type APIHandler func(context.Context, APICall) (any, error)
type apiKey struct{ category, operation, jid, node string }

// APIRegistry resolves exact JID+node, JID, node, then global handlers.
type APIRegistry struct {
	mu       sync.RWMutex
	handlers map[apiKey]APIHandler
}

func NewAPIRegistry() *APIRegistry { return &APIRegistry{handlers: make(map[apiKey]APIHandler)} }
func (r *APIRegistry) Register(category, operation, jid, node string, h APIHandler) error {
	if category == "" || operation == "" || h == nil {
		return fmt.Errorf("xmpp: category, operation, and handler are required")
	}
	r.mu.Lock()
	r.handlers[apiKey{category, operation, jid, node}] = h
	r.mu.Unlock()
	return nil
}
func (r *APIRegistry) Unregister(category, operation, jid, node string) {
	r.mu.Lock()
	delete(r.handlers, apiKey{category, operation, jid, node})
	r.mu.Unlock()
}
func (r *APIRegistry) Purge(category string) {
	r.mu.Lock()
	for k := range r.handlers {
		if k.category == category {
			delete(r.handlers, k)
		}
	}
	r.mu.Unlock()
}
func (r *APIRegistry) Run(ctx context.Context, c APICall) (any, error) {
	keys := []apiKey{{c.Category, c.Operation, c.JID, c.Node}, {c.Category, c.Operation, c.JID, ""}, {c.Category, c.Operation, "", c.Node}, {c.Category, c.Operation, "", ""}}
	r.mu.RLock()
	var h APIHandler
	for _, k := range keys {
		if candidate, ok := r.handlers[k]; ok {
			h = candidate
			break
		}
	}
	r.mu.RUnlock()
	if h == nil {
		return nil, fmt.Errorf("xmpp: no API handler for %s.%s", c.Category, c.Operation)
	}
	return h(ctx, c)
}

// APIProxy binds calls to one category.
type APIProxy struct {
	registry *APIRegistry
	category string
}

func (r *APIRegistry) Proxy(category string) APIProxy { return APIProxy{r, category} }
func (p APIProxy) Register(operation, jid, node string, h APIHandler) error {
	return p.registry.Register(p.category, operation, jid, node, h)
}
func (p APIProxy) Run(ctx context.Context, operation, jid, node, from string, args any) (any, error) {
	return p.registry.Run(ctx, APICall{Category: p.category, Operation: operation, JID: jid, Node: node, From: from, Args: args})
}
