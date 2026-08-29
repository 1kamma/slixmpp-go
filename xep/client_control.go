package xep

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/1kamma/slixmpp-go/xmpp"
)

const (
	BlockingNS         = "urn:xmpp:blocking"
	CarbonsNS          = "urn:xmpp:carbons:2"
	CSINS              = "urn:xmpp:csi:0"
	StreamManagementNS = "urn:xmpp:sm:3"
)

// Blocking implements XEP-0191 block-list operations and pushes.
type Blocking struct {
	client   *xmpp.Client
	mu       sync.RWMutex
	blocked  map[string]bool
	handlers []*xmpp.Handler
}

func NewBlocking() *Blocking               { return &Blocking{blocked: make(map[string]bool)} }
func (b *Blocking) Name() string           { return "xep_0191" }
func (b *Blocking) Description() string    { return "XEP-0191 Blocking Command" }
func (b *Blocking) Dependencies() []string { return nil }
func (b *Blocking) Features() []string     { return []string{BlockingNS} }
func (b *Blocking) Init(c *xmpp.Client) error {
	b.client = c
	if b.blocked == nil {
		b.blocked = make(map[string]bool)
	}
	b.handlers = append(b.handlers, c.AddHandler("block-push", xmpp.MatchAnd(xmpp.MatchIQType(xmpp.IQSet), xmpp.MatchPayload(BlockingNS, "block")), b.handlePush), c.AddHandler("unblock-push", xmpp.MatchAnd(xmpp.MatchIQType(xmpp.IQSet), xmpp.MatchPayload(BlockingNS, "unblock")), b.handlePush))
	return nil
}
func (b *Blocking) Shutdown(context.Context) error {
	if b.client != nil {
		for _, h := range b.handlers {
			b.client.RemoveHandler(h)
		}
	}
	return nil
}
func (b *Blocking) List(ctx context.Context) ([]string, error) {
	blocklist := xmpp.NewNode(BlockingNS, "blocklist")
	response, err := b.client.RequestIQ(ctx, xmpp.IQ{Type: xmpp.IQGet, Payloads: []xmpp.Node{blocklist}})
	if err != nil {
		return nil, err
	}
	p := response.Payload()
	if p == nil {
		return nil, fmt.Errorf("xep: missing blocklist")
	}
	items := parseBlockItems(*p)
	b.mu.Lock()
	b.blocked = make(map[string]bool, len(items))
	for _, jid := range items {
		b.blocked[jid] = true
	}
	b.mu.Unlock()
	return items, nil
}
func (b *Blocking) Block(ctx context.Context, jids ...string) error {
	return b.change(ctx, "block", jids)
}
func (b *Blocking) Unblock(ctx context.Context, jids ...string) error {
	return b.change(ctx, "unblock", jids)
}
func (b *Blocking) UnblockAll(ctx context.Context) error { return b.change(ctx, "unblock", nil) }
func (b *Blocking) IsBlocked(jid string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.blocked[xmpp.BareJIDString(jid)]
}
func (b *Blocking) change(ctx context.Context, action string, jids []string) error {
	n := xmpp.NewNode(BlockingNS, action)
	for _, jid := range jids {
		item := xmpp.NewNode(BlockingNS, "item")
		item.SetAttr("jid", jid)
		n.AddChild(item)
	}
	_, err := b.client.RequestIQ(ctx, xmpp.IQ{Type: xmpp.IQSet, Payloads: []xmpp.Node{n}})
	return err
}
func (b *Blocking) handlePush(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
	iq := asIQ(s)
	if iq.From != "" && xmpp.BareJIDString(iq.From) != c.JID().Bare() && iq.From != c.JID().Domain {
		return c.Send(iq.ErrorResult(&xmpp.StanzaError{Type: "cancel", Condition: "not-allowed"}))
	}
	p := iq.Payload()
	items := parseBlockItems(*p)
	b.mu.Lock()
	if p.Name.Local == "block" {
		for _, jid := range items {
			b.blocked[jid] = true
		}
	} else if len(items) == 0 {
		b.blocked = make(map[string]bool)
	} else {
		for _, jid := range items {
			delete(b.blocked, jid)
		}
	}
	b.mu.Unlock()
	if err := c.ReplyIQ(iq); err != nil {
		return err
	}
	return c.Events.Emit(ctx, "blocking_"+p.Name.Local, items)
}
func parseBlockItems(n xmpp.Node) []string {
	var out []string
	for _, item := range n.Children() {
		if item.Name.Local == "item" {
			if jid, ok := item.AttrValue("jid"); ok {
				out = append(out, jid)
			}
		}
	}
	return out
}

// Carbon is a parsed XEP-0280 sent or received copy.
type Carbon struct {
	Direction string
	Forwarded Forwarded
	Envelope  xmpp.Message
}
type Carbons struct {
	client  *xmpp.Client
	handler *xmpp.Handler
	mu      sync.RWMutex
	enabled bool
}

func NewCarbons() *Carbons                 { return &Carbons{} }
func (ca *Carbons) Name() string           { return "xep_0280" }
func (ca *Carbons) Description() string    { return "XEP-0280 Message Carbons" }
func (ca *Carbons) Dependencies() []string { return []string{"xep_0297"} }
func (ca *Carbons) Features() []string     { return []string{CarbonsNS} }
func (ca *Carbons) Init(c *xmpp.Client) error {
	ca.client = c
	ca.handler = c.AddHandler("carbons", xmpp.MatchKind("message"), ca.handle)
	return nil
}
func (ca *Carbons) Shutdown(context.Context) error {
	if ca.client != nil {
		ca.client.RemoveHandler(ca.handler)
	}
	return nil
}
func (ca *Carbons) Enable(ctx context.Context) error  { return ca.set(ctx, true) }
func (ca *Carbons) Disable(ctx context.Context) error { return ca.set(ctx, false) }
func (ca *Carbons) Enabled() bool                     { ca.mu.RLock(); defer ca.mu.RUnlock(); return ca.enabled }
func (ca *Carbons) set(ctx context.Context, enabled bool) error {
	local := "disable"
	if enabled {
		local = "enable"
	}
	_, err := ca.client.RequestIQ(ctx, xmpp.IQ{Type: xmpp.IQSet, Payloads: []xmpp.Node{xmpp.NewNode(CarbonsNS, local)}})
	if err == nil {
		ca.mu.Lock()
		ca.enabled = enabled
		ca.mu.Unlock()
	}
	return err
}
func (ca *Carbons) handle(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
	m := asMessage(s)
	for _, direction := range []string{"sent", "received"} {
		node := m.Extension(CarbonsNS, direction)
		if node == nil {
			continue
		}
		forward := node.Child(ForwardNS, "forwarded")
		if forward == nil {
			return fmt.Errorf("xep: carbon has no forwarded stanza")
		}
		parsed, err := ParseForwarded(*forward)
		if err != nil {
			return err
		}
		carbon := Carbon{Direction: direction, Forwarded: parsed, Envelope: m}
		_ = c.Events.Emit(ctx, "carbon_"+direction, carbon)
		return c.Events.Emit(ctx, "carbon", carbon)
	}
	return nil
}

// CSI implements XEP-0352 client-state indication.
type CSI struct {
	client *xmpp.Client
	mu     sync.RWMutex
	active bool
}

func NewCSI() *CSI                            { return &CSI{active: true} }
func (c *CSI) Name() string                   { return "xep_0352" }
func (c *CSI) Description() string            { return "XEP-0352 Client State Indication" }
func (c *CSI) Dependencies() []string         { return nil }
func (c *CSI) Features() []string             { return []string{CSINS} }
func (c *CSI) Init(client *xmpp.Client) error { c.client = client; return nil }
func (c *CSI) Shutdown(context.Context) error { return nil }
func (c *CSI) Active() error {
	if err := c.client.SendNode(xmpp.NewNode(CSINS, "active")); err != nil {
		return err
	}
	c.mu.Lock()
	c.active = true
	c.mu.Unlock()
	return nil
}
func (c *CSI) Inactive() error {
	if err := c.client.SendNode(xmpp.NewNode(CSINS, "inactive")); err != nil {
		return err
	}
	c.mu.Lock()
	c.active = false
	c.mu.Unlock()
	return nil
}
func (c *CSI) IsActive() bool { c.mu.RLock(); defer c.mu.RUnlock(); return c.active }

// SMState is a snapshot of XEP-0198 counters and resume information.
type SMState struct {
	Enabled, Resume          bool
	ID, Location             string
	Max                      int
	Inbound, Outbound, Acked uint32
	Unacked                  []xmpp.Stanza
}
type StreamManagement struct {
	client        *xmpp.Client
	mu            sync.RWMutex
	state         SMState
	unacked       []xmpp.Stanza
	subscriptions []*xmpp.Subscription
}

func NewStreamManagement() *StreamManagement        { return &StreamManagement{} }
func (sm *StreamManagement) Name() string           { return "xep_0198" }
func (sm *StreamManagement) Description() string    { return "XEP-0198 Stream Management" }
func (sm *StreamManagement) Dependencies() []string { return nil }
func (sm *StreamManagement) Features() []string     { return []string{StreamManagementNS} }
func (sm *StreamManagement) Init(c *xmpp.Client) error {
	sm.client = c
	sm.subscriptions = append(sm.subscriptions, c.Events.On("sent_stanza", sm.onSent), c.Events.On("received_stanza", sm.onReceived), c.Events.On("stream_element", sm.onElement))
	return nil
}
func (sm *StreamManagement) Shutdown(context.Context) error {
	if sm.client != nil {
		for _, s := range sm.subscriptions {
			sm.client.Events.Off(s)
		}
	}
	return nil
}
func (sm *StreamManagement) Enable(ctx context.Context, resume bool) (SMState, error) {
	ch := make(chan xmpp.Node, 1)
	sub := sm.client.Events.On("stream_element", func(ctx context.Context, event xmpp.Event) error {
		n, ok := event.Data.(xmpp.Node)
		if ok && n.Name.Space == StreamManagementNS && (n.Name.Local == "enabled" || n.Name.Local == "failed") {
			select {
			case ch <- n:
			default:
				{
				}
			}
		}
		return nil
	})
	defer sm.client.Events.Off(sub)
	n := xmpp.NewNode(StreamManagementNS, "enable")
	if resume {
		n.SetAttr("resume", "true")
	}
	if err := sm.client.SendNode(n); err != nil {
		return SMState{}, err
	}
	select {
	case result := <-ch:
		if result.Name.Local == "failed" {
			return SMState{}, fmt.Errorf("xep: stream management enable failed: %s", result.MustXML())
		}
		state := parseEnabled(result)
		sm.mu.Lock()
		sm.state = state
		sm.unacked = nil
		sm.mu.Unlock()
		return sm.State(), nil
	case <-ctx.Done():
		return SMState{}, ctx.Err()
	case <-sm.client.Done():
		return SMState{}, xmpp.ErrClosed
	}
}
func (sm *StreamManagement) RequestAck() error {
	return sm.client.SendNode(xmpp.NewNode(StreamManagementNS, "r"))
}
func (sm *StreamManagement) State() SMState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s := sm.state
	s.Unacked = append([]xmpp.Stanza(nil), sm.unacked...)
	return s
}
func (sm *StreamManagement) ResumeNode(previousID string, handled uint32) xmpp.Node {
	n := xmpp.NewNode(StreamManagementNS, "resume")
	n.SetAttr("previd", previousID)
	n.SetAttr("h", strconv.FormatUint(uint64(handled), 10))
	return n
}
func (sm *StreamManagement) onSent(ctx context.Context, event xmpp.Event) error {
	stanza, ok := event.Data.(xmpp.Stanza)
	if !ok {
		return nil
	}
	sm.mu.Lock()
	if sm.state.Enabled {
		sm.state.Outbound++
		sm.unacked = append(sm.unacked, stanza)
	}
	sm.mu.Unlock()
	return nil
}
func (sm *StreamManagement) onReceived(ctx context.Context, event xmpp.Event) error {
	sm.mu.Lock()
	if sm.state.Enabled {
		sm.state.Inbound++
	}
	sm.mu.Unlock()
	return nil
}
func (sm *StreamManagement) onElement(ctx context.Context, event xmpp.Event) error {
	n, ok := event.Data.(xmpp.Node)
	if !ok || n.Name.Space != StreamManagementNS {
		return nil
	}
	switch n.Name.Local {
	case "r":
		sm.mu.RLock()
		handled := sm.state.Inbound
		enabled := sm.state.Enabled
		sm.mu.RUnlock()
		if enabled {
			a := xmpp.NewNode(StreamManagementNS, "a")
			a.SetAttr("h", strconv.FormatUint(uint64(handled), 10))
			return sm.client.SendNode(a)
		}
	case "a":
		value, _ := n.AttrValue("h")
		h, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("xep: invalid SM acknowledgement %q", value)
		}
		sm.applyAck(uint32(h))
		_ = sm.client.Events.Emit(ctx, "sm_acknowledged", sm.State())
	case "resumed":
		value, _ := n.AttrValue("h")
		h, _ := strconv.ParseUint(value, 10, 32)
		sm.applyAck(uint32(h))
		sm.mu.Lock()
		sm.state.Enabled = true
		sm.state.ID, _ = n.AttrValue("previd")
		sm.mu.Unlock()
		_ = sm.client.Events.Emit(ctx, "sm_resumed", sm.State())
	case "failed":
		_ = sm.client.Events.Emit(ctx, "sm_failed", n)
	}
	return nil
}
func (sm *StreamManagement) applyAck(h uint32) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delta := h - sm.state.Acked
	if uint64(delta) > uint64(len(sm.unacked)) {
		delta = uint32(len(sm.unacked))
	}
	if delta > 0 {
		sm.unacked = append([]xmpp.Stanza(nil), sm.unacked[delta:]...)
	}
	sm.state.Acked = h
}
func parseEnabled(n xmpp.Node) SMState {
	s := SMState{Enabled: true}
	s.ID, _ = n.AttrValue("id")
	s.Location, _ = n.AttrValue("location")
	if v, ok := n.AttrValue("resume"); ok {
		s.Resume = v == "true" || v == "1"
	}
	if v, ok := n.AttrValue("max"); ok {
		s.Max, _ = strconv.Atoi(v)
	}
	return s
}

func init() {
	registerSpecialized(191, func() xmpp.Plugin { return NewBlocking() })
	registerSpecialized(198, func() xmpp.Plugin { return NewStreamManagement() })
	registerSpecialized(280, func() xmpp.Plugin { return NewCarbons() })
	registerSpecialized(352, func() xmpp.Plugin { return NewCSI() })
}
