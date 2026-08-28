package xmpp

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

const rosterNS = "jabber:iq:roster"

// RosterItem is an RFC 6121 roster entry.
type RosterItem struct {
	JID, Name, Subscription, Ask string
	Approved                     bool
	Groups                       []string
}

// RosterManager caches and modifies the account roster.
type RosterManager struct {
	client  *Client
	mu      sync.RWMutex
	items   map[string]RosterItem
	version string
	handler *Handler
}

// NewRosterManager attaches roster-push handling to client.
func NewRosterManager(client *Client) *RosterManager {
	r := &RosterManager{client: client, items: make(map[string]RosterItem)}
	if client != nil {
		r.handler = client.AddHandler("roster-push", MatchAnd(MatchIQType(IQSet), MatchPayload(rosterNS, "query")), r.handlePush)
	}
	return r
}

// Version returns the cached roster version.
func (r *RosterManager) Version() string { r.mu.RLock(); defer r.mu.RUnlock(); return r.version }

// Items returns a sorted copy of the cache.
func (r *RosterManager) Items() []RosterItem {
	r.mu.RLock()
	out := make([]RosterItem, 0, len(r.items))
	for _, item := range r.items {
		copy := item
		copy.Groups = append([]string(nil), item.Groups...)
		out = append(out, copy)
	}
	r.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].JID < out[j].JID })
	return out
}

// Get fetches the roster, using versioning when version is non-empty.
func (r *RosterManager) Get(ctx context.Context, version string) ([]RosterItem, error) {
	q := NewNode(rosterNS, "query")
	if version != "" {
		q.SetAttr("ver", version)
	}
	response, err := r.client.RequestIQ(ctx, IQ{Type: IQGet, Payloads: []Node{q}})
	if err != nil {
		return nil, err
	}
	payload := response.Payload()
	if payload == nil {
		return nil, fmt.Errorf("xmpp: missing roster query")
	}
	items, ver, err := parseRosterQuery(*payload)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	if len(items) > 0 || version == "" {
		for _, item := range items {
			if item.Subscription == "remove" {
				delete(r.items, BareJIDString(item.JID))
			} else {
				r.items[BareJIDString(item.JID)] = item
			}
		}
	}
	if ver != "" {
		r.version = ver
	}
	r.mu.Unlock()
	_ = r.client.Events.Emit(ctx, "roster_update", r.Items())
	return r.Items(), nil
}

// Set creates or updates a roster entry.
func (r *RosterManager) Set(ctx context.Context, item RosterItem) error {
	if item.JID == "" {
		return fmt.Errorf("xmpp: roster JID is required")
	}
	q := NewNode(rosterNS, "query")
	q.AddChild(rosterItemNode(item))
	_, err := r.client.RequestIQ(ctx, IQ{Type: IQSet, Payloads: []Node{q}})
	return err
}

// Remove deletes a roster entry.
func (r *RosterManager) Remove(ctx context.Context, jid string) error {
	return r.Set(ctx, RosterItem{JID: BareJIDString(jid), Subscription: "remove"})
}

// Subscribe sends a presence subscription request.
func (r *RosterManager) Subscribe(jid, status string) error {
	return r.client.Send(Presence{To: BareJIDString(jid), Type: PresenceSubscribe, Status: status})
}
func (r *RosterManager) Approve(jid string) error {
	return r.client.Send(Presence{To: BareJIDString(jid), Type: PresenceSubscribed})
}
func (r *RosterManager) Unsubscribe(jid string) error {
	return r.client.Send(Presence{To: BareJIDString(jid), Type: PresenceUnsubscribe})
}
func (r *RosterManager) Revoke(jid string) error {
	return r.client.Send(Presence{To: BareJIDString(jid), Type: PresenceUnsubscribed})
}

func (r *RosterManager) handlePush(ctx context.Context, c *Client, s Stanza) error {
	iq := asRosterIQ(s)
	payload := iq.Payload()
	items, ver, err := parseRosterQuery(*payload)
	if err != nil {
		return c.Send(iq.ErrorResult(&StanzaError{Type: "modify", Condition: "bad-request", Text: err.Error()}))
	}
	// RFC 6121 permits pushes only from the account bare JID or server.
	if iq.From != "" && BareJIDString(iq.From) != c.JID().Bare() && iq.From != c.JID().Domain {
		return c.Send(iq.ErrorResult(&StanzaError{Type: "cancel", Condition: "service-unavailable"}))
	}
	r.mu.Lock()
	for _, item := range items {
		key := BareJIDString(item.JID)
		if item.Subscription == "remove" {
			delete(r.items, key)
		} else {
			r.items[key] = item
		}
	}
	if ver != "" {
		r.version = ver
	}
	r.mu.Unlock()
	if err := c.ReplyIQ(iq); err != nil {
		return err
	}
	for _, item := range items {
		_ = c.Events.Emit(ctx, "roster_item", item)
	}
	return c.Events.Emit(ctx, "roster_update", r.Items())
}

func parseRosterQuery(q Node) ([]RosterItem, string, error) {
	if q.Name.Space != rosterNS || q.Name.Local != "query" {
		return nil, "", fmt.Errorf("xmpp: invalid roster query")
	}
	ver, _ := q.AttrValue("ver")
	var items []RosterItem
	for _, n := range q.Children() {
		if n.Name.Local != "item" {
			continue
		}
		item := RosterItem{}
		item.JID, _ = n.AttrValue("jid")
		item.Name, _ = n.AttrValue("name")
		item.Subscription, _ = n.AttrValue("subscription")
		item.Ask, _ = n.AttrValue("ask")
		if v, ok := n.AttrValue("approved"); ok {
			item.Approved = v == "true" || v == "1"
		}
		if item.JID == "" {
			return nil, ver, fmt.Errorf("xmpp: roster item missing jid")
		}
		for _, g := range n.Children() {
			if g.Name.Local == "group" {
				item.Groups = append(item.Groups, g.Text())
			}
		}
		items = append(items, item)
	}
	return items, ver, nil
}
func rosterItemNode(item RosterItem) Node {
	n := NewNode(rosterNS, "item")
	n.SetAttr("jid", BareJIDString(item.JID))
	if item.Name != "" {
		n.SetAttr("name", item.Name)
	}
	if item.Subscription != "" {
		n.SetAttr("subscription", item.Subscription)
	}
	for _, g := range item.Groups {
		n.AddChild(NewTextNode(rosterNS, "group", g))
	}
	return n
}
func asRosterIQ(s Stanza) IQ {
	switch v := s.(type) {
	case IQ:
		return v
	case *IQ:
		return *v
	}
	panic("xmpp: stanza is not IQ")
}
