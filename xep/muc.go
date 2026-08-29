package xep

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/1kamma/slixmpp-go/xmpp"
)

const (
	MUCNS      = "http://jabber.org/protocol/muc"
	MUCUserNS  = "http://jabber.org/protocol/muc#user"
	MUCAdminNS = "http://jabber.org/protocol/muc#admin"
	MUCOwnerNS = "http://jabber.org/protocol/muc#owner"
)

type MUCHistory struct {
	MaxChars, MaxStanzas, Seconds *int
	Since                         *time.Time
}
type JoinOptions struct {
	Password string
	History  *MUCHistory
}
type MUCItem struct{ Affiliation, Role, JID, Nick, Reason, Actor string }
type MUCStatus struct {
	Codes                                 []int
	Item                                  *MUCItem
	Password                              string
	DestroyJID, DestroyReason             string
	InviteFrom, InviteTo, InviteReason    string
	DeclineFrom, DeclineTo, DeclineReason string
}
type Occupant struct {
	Room, Nick, FullJID                                  string
	Affiliation, Role, RealJID, Show, Status, OccupantID string
	Available                                            bool
}
type Room struct {
	JID, Nick string
	Joined    bool
	Subject   string
	Occupants map[string]Occupant
}
type joinResult struct {
	room Room
	err  error
}

type MUC struct {
	client   *xmpp.Client
	mu       sync.RWMutex
	rooms    map[string]*Room
	pending  map[string]chan joinResult
	handlers []*xmpp.Handler
}

func NewMUC() *MUC {
	return &MUC{rooms: make(map[string]*Room), pending: make(map[string]chan joinResult)}
}
func (m *MUC) Name() string           { return "xep_0045" }
func (m *MUC) Description() string    { return "XEP-0045 Multi-User Chat" }
func (m *MUC) Dependencies() []string { return []string{"xep_0030", "xep_0004"} }
func (m *MUC) Features() []string     { return []string{MUCNS, MUCUserNS, MUCAdminNS, MUCOwnerNS} }
func (m *MUC) Init(c *xmpp.Client) error {
	m.client = c
	if m.rooms == nil {
		m.rooms = make(map[string]*Room)
	}
	if m.pending == nil {
		m.pending = make(map[string]chan joinResult)
	}
	m.handlers = append(m.handlers, c.AddHandler("muc-presence", xmpp.MatchAnd(xmpp.MatchKind("presence"), xmpp.MatchPayload(MUCUserNS, "x")), m.handlePresence), c.AddHandler("muc-message", xmpp.MatchKind("message"), m.handleMessage))
	return nil
}
func (m *MUC) Shutdown(context.Context) error {
	if m.client != nil {
		for _, h := range m.handlers {
			m.client.RemoveHandler(h)
		}
	}
	return nil
}
func (m *MUC) Rooms() []Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Room, 0, len(m.rooms))
	for _, r := range m.rooms {
		out = append(out, cloneRoom(r))
	}
	return out
}
func (m *MUC) Room(jid string) (Room, bool) {
	m.mu.RLock()
	r, ok := m.rooms[xmpp.BareJIDString(jid)]
	m.mu.RUnlock()
	if !ok {
		return Room{}, false
	}
	v := cloneRoom(r)
	return v, true
}
func (m *MUC) Join(ctx context.Context, room, nick string, options JoinOptions) (Room, error) {
	room = xmpp.BareJIDString(room)
	if room == "" || nick == "" {
		return Room{}, fmt.Errorf("xep: room and nickname are required")
	}
	full := room + "/" + nick
	x := xmpp.NewNode(MUCNS, "x")
	if options.Password != "" {
		x.AddChild(xmpp.NewTextNode(MUCNS, "password", options.Password))
	}
	if options.History != nil {
		h := xmpp.NewNode(MUCNS, "history")
		if options.History.MaxChars != nil {
			h.SetAttr("maxchars", strconv.Itoa(*options.History.MaxChars))
		}
		if options.History.MaxStanzas != nil {
			h.SetAttr("maxstanzas", strconv.Itoa(*options.History.MaxStanzas))
		}
		if options.History.Seconds != nil {
			h.SetAttr("seconds", strconv.Itoa(*options.History.Seconds))
		}
		if options.History.Since != nil {
			h.SetAttr("since", FormatDateTime(*options.History.Since))
		}
		x.AddChild(h)
	}
	ch := make(chan joinResult, 1)
	m.mu.Lock()
	if _, exists := m.pending[full]; exists {
		m.mu.Unlock()
		return Room{}, fmt.Errorf("xep: join already pending for %s", full)
	}
	m.pending[full] = ch
	m.rooms[room] = &Room{JID: room, Nick: nick, Occupants: map[string]Occupant{}}
	m.mu.Unlock()
	defer func() { m.mu.Lock(); delete(m.pending, full); m.mu.Unlock() }()
	if err := m.client.Send(xmpp.Presence{To: full, ID: m.client.NextID(), Extensions: []xmpp.Node{x}}); err != nil {
		return Room{}, err
	}
	select {
	case result := <-ch:
		return result.room, result.err
	case <-ctx.Done():
		return Room{}, ctx.Err()
	case <-m.client.Done():
		return Room{}, xmpp.ErrClosed
	}
}
func (m *MUC) Leave(room, status string) error {
	room = xmpp.BareJIDString(room)
	m.mu.RLock()
	r := m.rooms[room]
	m.mu.RUnlock()
	if r == nil {
		return fmt.Errorf("xep: not joined to %s", room)
	}
	return m.client.Send(xmpp.Presence{To: room + "/" + r.Nick, Type: xmpp.PresenceUnavailable, Status: status, ID: m.client.NextID()})
}
func (m *MUC) SendMessage(room, body string) error {
	return m.client.Send(xmpp.Message{To: xmpp.BareJIDString(room), ID: m.client.NextID(), Type: xmpp.MessageGroupChat, Body: body})
}
func (m *MUC) SetSubject(room, subject string) error {
	return m.client.Send(xmpp.Message{To: xmpp.BareJIDString(room), ID: m.client.NextID(), Type: xmpp.MessageGroupChat, Subject: subject})
}
func (m *MUC) Invite(room, jid, reason string) error {
	x := xmpp.NewNode(MUCUserNS, "x")
	invite := xmpp.NewNode(MUCUserNS, "invite")
	invite.SetAttr("to", jid)
	if reason != "" {
		invite.AddChild(xmpp.NewTextNode(MUCUserNS, "reason", reason))
	}
	x.AddChild(invite)
	return m.client.Send(xmpp.Message{To: xmpp.BareJIDString(room), ID: m.client.NextID(), Extensions: []xmpp.Node{x}})
}
func (m *MUC) Decline(room, to, reason string) error {
	x := xmpp.NewNode(MUCUserNS, "x")
	decline := xmpp.NewNode(MUCUserNS, "decline")
	decline.SetAttr("to", to)
	if reason != "" {
		decline.AddChild(xmpp.NewTextNode(MUCUserNS, "reason", reason))
	}
	x.AddChild(decline)
	return m.client.Send(xmpp.Message{To: xmpp.BareJIDString(room), ID: m.client.NextID(), Extensions: []xmpp.Node{x}})
}
func (m *MUC) GetConfig(ctx context.Context, room string) (Form, error) {
	q := xmpp.NewNode(MUCOwnerNS, "query")
	response, err := m.client.RequestIQ(ctx, xmpp.IQ{To: xmpp.BareJIDString(room), Type: xmpp.IQGet, Payloads: []xmpp.Node{q}})
	if err != nil {
		return Form{}, err
	}
	p := response.Payload()
	if p == nil {
		return Form{}, fmt.Errorf("xep: missing MUC owner response")
	}
	formNode := p.Child(DataFormsNS, "x")
	if formNode == nil {
		return Form{}, fmt.Errorf("xep: missing MUC configuration form")
	}
	return ParseForm(*formNode)
}
func (m *MUC) SetConfig(ctx context.Context, room string, form Form) error {
	form.Type = FormTypeSubmit
	q := xmpp.NewNode(MUCOwnerNS, "query")
	q.AddChild(form.ToNode())
	_, err := m.client.RequestIQ(ctx, xmpp.IQ{To: xmpp.BareJIDString(room), Type: xmpp.IQSet, Payloads: []xmpp.Node{q}})
	return err
}
func (m *MUC) Destroy(ctx context.Context, room, alternate, reason string) error {
	q := xmpp.NewNode(MUCOwnerNS, "query")
	d := xmpp.NewNode(MUCOwnerNS, "destroy")
	if alternate != "" {
		d.SetAttr("jid", alternate)
	}
	if reason != "" {
		d.AddChild(xmpp.NewTextNode(MUCOwnerNS, "reason", reason))
	}
	q.AddChild(d)
	_, err := m.client.RequestIQ(ctx, xmpp.IQ{To: xmpp.BareJIDString(room), Type: xmpp.IQSet, Payloads: []xmpp.Node{q}})
	return err
}
func (m *MUC) SetRole(ctx context.Context, room, nick, role, reason string) error {
	return m.setAdmin(ctx, room, MUCItem{Nick: nick, Role: role, Reason: reason})
}
func (m *MUC) SetAffiliation(ctx context.Context, room, jid, affiliation, reason string) error {
	return m.setAdmin(ctx, room, MUCItem{JID: jid, Affiliation: affiliation, Reason: reason})
}
func (m *MUC) setAdmin(ctx context.Context, room string, item MUCItem) error {
	q := xmpp.NewNode(MUCAdminNS, "query")
	q.AddChild(mucItemNode(MUCAdminNS, item))
	_, err := m.client.RequestIQ(ctx, xmpp.IQ{To: xmpp.BareJIDString(room), Type: xmpp.IQSet, Payloads: []xmpp.Node{q}})
	return err
}
func (m *MUC) GetAffiliations(ctx context.Context, room, affiliation string) ([]MUCItem, error) {
	q := xmpp.NewNode(MUCAdminNS, "query")
	item := xmpp.NewNode(MUCAdminNS, "item")
	item.SetAttr("affiliation", affiliation)
	q.AddChild(item)
	response, err := m.client.RequestIQ(ctx, xmpp.IQ{To: xmpp.BareJIDString(room), Type: xmpp.IQGet, Payloads: []xmpp.Node{q}})
	if err != nil {
		return nil, err
	}
	p := response.Payload()
	if p == nil {
		return nil, fmt.Errorf("xep: missing MUC admin response")
	}
	var out []MUCItem
	for _, n := range p.Children() {
		if n.Name.Local == "item" {
			out = append(out, parseMUCItem(n))
		}
	}
	return out, nil
}
func (m *MUC) handlePresence(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
	p := asPresence(s)
	room, nick := splitOccupant(p.From)
	status, _ := ParseMUCStatus(*p.Extension(MUCUserNS, "x"))
	occupant := Occupant{Room: room, Nick: nick, FullJID: p.From, Show: p.Show, Status: p.Status, Available: p.Type != xmpp.PresenceUnavailable}
	if status.Item != nil {
		occupant.Affiliation = status.Item.Affiliation
		occupant.Role = status.Item.Role
		occupant.RealJID = status.Item.JID
		if status.Item.Nick != "" && p.Type == xmpp.PresenceUnavailable {
			occupant.Nick = status.Item.Nick
		}
	}
	if id, ok := OccupantID(xmpp.Message{Extensions: p.Extensions}); ok {
		occupant.OccupantID = id
	}
	m.mu.Lock()
	r := m.rooms[room]
	if r == nil {
		r = &Room{JID: room, Occupants: map[string]Occupant{}}
		m.rooms[room] = r
	}
	if p.Type == xmpp.PresenceUnavailable {
		delete(r.Occupants, nick)
	} else {
		r.Occupants[nick] = occupant
	}
	self := containsInt(status.Codes, 110)
	if self && p.Type != xmpp.PresenceUnavailable {
		r.Joined = true
		if r.Nick == "" {
			r.Nick = nick
		}
	}
	if self && p.Type == xmpp.PresenceUnavailable {
		r.Joined = false
	}
	snapshot := cloneRoom(r)
	pending := m.pending[room+"/"+r.Nick]
	m.mu.Unlock()
	if p.Type == xmpp.PresenceError {
		err := p.Error
		if pending != nil {
			pending <- joinResult{err: err}
		}
		_ = c.Events.Emit(ctx, "muc_join_failed", p)
		return nil
	}
	if self && pending != nil && p.Type != xmpp.PresenceUnavailable {
		pending <- joinResult{room: snapshot}
	}
	if self && p.Type == xmpp.PresenceUnavailable {
		_ = c.Events.Emit(ctx, "muc_left", snapshot)
	}
	event := "muc_occupant_online"
	if p.Type == xmpp.PresenceUnavailable {
		event = "muc_occupant_offline"
	}
	_ = c.Events.Emit(ctx, event, occupant)
	return c.Events.Emit(ctx, "muc_presence", struct {
		Presence xmpp.Presence
		Status   MUCStatus
	}{p, status})
}
func (m *MUC) handleMessage(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
	message := asMessage(s)
	room := xmpp.BareJIDString(message.From)
	if message.Type == xmpp.MessageGroupChat && message.Subject != "" {
		m.mu.Lock()
		if r := m.rooms[room]; r != nil {
			r.Subject = message.Subject
		}
		m.mu.Unlock()
		_ = c.Events.Emit(ctx, "muc_subject", message)
	}
	if x := message.Extension(MUCUserNS, "x"); x != nil {
		status, _ := ParseMUCStatus(*x)
		if status.InviteFrom != "" || status.DeclineFrom != "" {
			_ = c.Events.Emit(ctx, "muc_invitation", struct {
				Message xmpp.Message
				Status  MUCStatus
			}{message, status})
		}
	}
	return nil
}
func ParseMUCStatus(x xmpp.Node) (MUCStatus, error) {
	if x.Name.Space != MUCUserNS || x.Name.Local != "x" {
		return MUCStatus{}, fmt.Errorf("xep: invalid MUC user payload")
	}
	var s MUCStatus
	for _, n := range x.Children() {
		switch n.Name.Local {
		case "status":
			v, _ := n.AttrValue("code")
			if i, e := strconv.Atoi(v); e == nil {
				s.Codes = append(s.Codes, i)
			}
		case "item":
			item := parseMUCItem(n)
			s.Item = &item
		case "password":
			s.Password = n.Text()
		case "destroy":
			s.DestroyJID, _ = n.AttrValue("jid")
			s.DestroyReason = n.ChildText(MUCUserNS, "reason")
		case "invite":
			s.InviteFrom, _ = n.AttrValue("from")
			s.InviteTo, _ = n.AttrValue("to")
			s.InviteReason = n.ChildText(MUCUserNS, "reason")
		case "decline":
			s.DeclineFrom, _ = n.AttrValue("from")
			s.DeclineTo, _ = n.AttrValue("to")
			s.DeclineReason = n.ChildText(MUCUserNS, "reason")
		}
	}
	return s, nil
}
func parseMUCItem(n xmpp.Node) MUCItem {
	i := MUCItem{}
	i.Affiliation, _ = n.AttrValue("affiliation")
	i.Role, _ = n.AttrValue("role")
	i.JID, _ = n.AttrValue("jid")
	i.Nick, _ = n.AttrValue("nick")
	i.Reason = n.ChildText(n.Name.Space, "reason")
	if a := n.Child(n.Name.Space, "actor"); a != nil {
		i.Actor, _ = a.AttrValue("jid")
	}
	return i
}
func mucItemNode(ns string, item MUCItem) xmpp.Node {
	n := xmpp.NewNode(ns, "item")
	if item.Affiliation != "" {
		n.SetAttr("affiliation", item.Affiliation)
	}
	if item.Role != "" {
		n.SetAttr("role", item.Role)
	}
	if item.JID != "" {
		n.SetAttr("jid", item.JID)
	}
	if item.Nick != "" {
		n.SetAttr("nick", item.Nick)
	}
	if item.Reason != "" {
		n.AddChild(xmpp.NewTextNode(ns, "reason", item.Reason))
	}
	return n
}
func splitOccupant(jid string) (string, string) {
	slash := strings.IndexByte(jid, '/')
	if slash < 0 {
		return jid, ""
	}
	return jid[:slash], jid[slash+1:]
}
func asPresence(s xmpp.Stanza) xmpp.Presence {
	switch v := s.(type) {
	case xmpp.Presence:
		return v
	case *xmpp.Presence:
		return *v
	}
	panic("xep: stanza is not presence")
}
func cloneRoom(r *Room) Room {
	if r == nil {
		return Room{}
	}
	out := *r
	out.Occupants = make(map[string]Occupant, len(r.Occupants))
	for k, v := range r.Occupants {
		out.Occupants[k] = v
	}
	return out
}
func containsInt(values []int, want int) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
func init() { registerSpecialized(45, func() xmpp.Plugin { return NewMUC() }) }
