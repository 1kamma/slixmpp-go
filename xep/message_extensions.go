package xep

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/saret/slixmpp-go/xmpp"
)

const (
	ChatStatesNS      = "http://jabber.org/protocol/chatstates"
	ReceiptsNS        = "urn:xmpp:receipts"
	CorrectionNS      = "urn:xmpp:message-correct:0"
	MarkersNS         = "urn:xmpp:chat-markers:0"
	HintsNS           = "urn:xmpp:hints"
	JSONNS            = "urn:xmpp:json:0"
	StanzaIDNS        = "urn:xmpp:sid:0"
	ReferenceNS       = "urn:xmpp:reference:0"
	EncryptionNS      = "urn:xmpp:eme:0"
	SpoilerNS         = "urn:xmpp:spoiler:0"
	OccupantIDNS      = "urn:xmpp:occupant-id:0"
	RetractionNS      = "urn:xmpp:message-retract:1"
	FallbackNS        = "urn:xmpp:fallback:0"
	ReactionsNS       = "urn:xmpp:reactions:0"
	ReplyNS           = "urn:xmpp:reply:0"
	MentionNS         = "urn:xmpp:mention:0"
	AttentionNS       = "urn:xmpp:attention:0"
	DirectMUCInviteNS = "jabber:x:conference"
	MessageMarkupNS   = "urn:xmpp:markup:0"
	DisplayedSyncNS   = "urn:xmpp:mds:displayed:0"
	CallInvitesNS     = "urn:xmpp:call-invites:0"
)

type ChatState string

const (
	ChatActive    ChatState = "active"
	ChatComposing ChatState = "composing"
	ChatPaused    ChatState = "paused"
	ChatInactive  ChatState = "inactive"
	ChatGone      ChatState = "gone"
)

func SetChatState(message *xmpp.Message, state ChatState) {
	removeExtension(message, ChatStatesNS, "active", "composing", "paused", "inactive", "gone")
	if state != "" {
		message.Extensions = append(message.Extensions, xmpp.NewNode(ChatStatesNS, string(state)))
	}
}
func GetChatState(message xmpp.Message) (ChatState, bool) {
	for _, state := range []ChatState{ChatActive, ChatComposing, ChatPaused, ChatInactive, ChatGone} {
		if message.Extension(ChatStatesNS, string(state)) != nil {
			return state, true
		}
	}
	return "", false
}

// Receipt describes a delivery-receipt extension.
type Receipt struct {
	ID      string
	Request bool
}

func RequestReceipt(message *xmpp.Message) {
	if message.Extension(ReceiptsNS, "request") == nil {
		message.Extensions = append(message.Extensions, xmpp.NewNode(ReceiptsNS, "request"))
	}
}
func ReceiptID(message xmpp.Message) (string, bool) {
	n := message.Extension(ReceiptsNS, "received")
	if n == nil {
		return "", false
	}
	id, _ := n.AttrValue("id")
	return id, id != ""
}
func ReceiptMessage(to, id string) xmpp.Message {
	n := xmpp.NewNode(ReceiptsNS, "received")
	n.SetAttr("id", id)
	return xmpp.Message{To: to, Type: xmpp.MessageChat, Extensions: []xmpp.Node{n}}
}

type Receipts struct {
	client  *xmpp.Client
	AutoAck bool
	handler *xmpp.Handler
}

func NewReceipts() *Receipts               { return &Receipts{AutoAck: true} }
func (r *Receipts) Name() string           { return "xep_0184" }
func (r *Receipts) Description() string    { return "XEP-0184 Message Delivery Receipts" }
func (r *Receipts) Dependencies() []string { return nil }
func (r *Receipts) Features() []string     { return []string{ReceiptsNS} }
func (r *Receipts) Init(c *xmpp.Client) error {
	r.client = c
	r.handler = c.AddHandler("receipts", xmpp.MatchKind("message"), r.handle)
	return nil
}
func (r *Receipts) Shutdown(context.Context) error {
	if r.client != nil {
		r.client.RemoveHandler(r.handler)
	}
	return nil
}
func (r *Receipts) handle(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
	m := asMessage(s)
	if id, ok := ReceiptID(m); ok {
		_ = c.Events.Emit(ctx, "receipt_received", Receipt{ID: id})
		return nil
	}
	if !r.AutoAck || m.ID == "" || m.Extension(ReceiptsNS, "request") == nil || m.Type == xmpp.MessageGroupChat || m.Type == xmpp.MessageError {
		return nil
	}
	ack := ReceiptMessage(m.From, m.ID)
	ack.ID = c.NextID()
	return c.Send(ack)
}
func (r *Receipts) Send(ctx context.Context, message xmpp.Message) (string, error) {
	if message.ID == "" {
		message.ID = r.client.NextID()
	}
	RequestReceipt(&message)
	return message.ID, r.client.Send(message)
}

func SetCorrection(message *xmpp.Message, replacesID string) {
	removeExtension(message, CorrectionNS, "replace")
	if replacesID != "" {
		n := xmpp.NewNode(CorrectionNS, "replace")
		n.SetAttr("id", replacesID)
		message.Extensions = append(message.Extensions, n)
	}
}
func CorrectionID(message xmpp.Message) (string, bool) {
	n := message.Extension(CorrectionNS, "replace")
	if n == nil {
		return "", false
	}
	id, _ := n.AttrValue("id")
	return id, id != ""
}

type MarkerType string

const (
	MarkerMarkable     MarkerType = "markable"
	MarkerReceived     MarkerType = "received"
	MarkerDisplayed    MarkerType = "displayed"
	MarkerAcknowledged MarkerType = "acknowledged"
)

type Marker struct {
	Type MarkerType
	ID   string
}

func SetMarker(message *xmpp.Message, marker Marker) {
	removeExtension(message, MarkersNS, "markable", "received", "displayed", "acknowledged")
	n := xmpp.NewNode(MarkersNS, string(marker.Type))
	if marker.ID != "" {
		n.SetAttr("id", marker.ID)
	}
	message.Extensions = append(message.Extensions, n)
}
func GetMarker(message xmpp.Message) (Marker, bool) {
	for _, kind := range []MarkerType{MarkerMarkable, MarkerReceived, MarkerDisplayed, MarkerAcknowledged} {
		if n := message.Extension(MarkersNS, string(kind)); n != nil {
			id, _ := n.AttrValue("id")
			return Marker{Type: kind, ID: id}, true
		}
	}
	return Marker{}, false
}

type Hint string

const (
	HintNoPermanentStore Hint = "no-permanent-store"
	HintNoStore          Hint = "no-store"
	HintNoCopy           Hint = "no-copy"
	HintStore            Hint = "store"
)

func AddHint(message *xmpp.Message, hint Hint) {
	if message.Extension(HintsNS, string(hint)) == nil {
		message.Extensions = append(message.Extensions, xmpp.NewNode(HintsNS, string(hint)))
	}
}
func HasHint(message xmpp.Message, hint Hint) bool {
	return message.Extension(HintsNS, string(hint)) != nil
}

func SetJSON(message *xmpp.Message, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var check any
	if err := json.Unmarshal(data, &check); err != nil {
		return err
	}
	removeExtension(message, JSONNS, "json")
	message.Extensions = append(message.Extensions, xmpp.NewTextNode(JSONNS, "json", string(data)))
	return nil
}
func GetJSON(message xmpp.Message, target any) error {
	n := message.Extension(JSONNS, "json")
	if n == nil {
		return fmt.Errorf("xep: message has no JSON container")
	}
	return json.Unmarshal([]byte(n.Text()), target)
}

type OriginID struct{ ID string }
type StanzaID struct{ ID, By string }

func SetOriginID(message *xmpp.Message, id string) {
	removeExtension(message, StanzaIDNS, "origin-id")
	n := xmpp.NewNode(StanzaIDNS, "origin-id")
	n.SetAttr("id", id)
	message.Extensions = append(message.Extensions, n)
}
func GetOriginID(message xmpp.Message) (string, bool) {
	n := message.Extension(StanzaIDNS, "origin-id")
	if n == nil {
		return "", false
	}
	id, _ := n.AttrValue("id")
	return id, id != ""
}
func AddStanzaID(message *xmpp.Message, id, by string) {
	n := xmpp.NewNode(StanzaIDNS, "stanza-id")
	n.SetAttr("id", id)
	n.SetAttr("by", by)
	message.Extensions = append(message.Extensions, n)
}
func StanzaIDs(message xmpp.Message) []StanzaID {
	var out []StanzaID
	for _, n := range message.Extensions {
		if n.Name.Space == StanzaIDNS && n.Name.Local == "stanza-id" {
			id, _ := n.AttrValue("id")
			by, _ := n.AttrValue("by")
			out = append(out, StanzaID{ID: id, By: by})
		}
	}
	return out
}

type Reference struct {
	Type, URI, Anchor string
	Begin, End        *int
}

func (r Reference) ToNode() xmpp.Node {
	n := xmpp.NewNode(ReferenceNS, "reference")
	if r.Type != "" {
		n.SetAttr("type", r.Type)
	}
	if r.URI != "" {
		n.SetAttr("uri", r.URI)
	}
	if r.Anchor != "" {
		n.SetAttr("anchor", r.Anchor)
	}
	if r.Begin != nil {
		n.SetAttr("begin", strconv.Itoa(*r.Begin))
	}
	if r.End != nil {
		n.SetAttr("end", strconv.Itoa(*r.End))
	}
	return n
}
func ParseReference(n xmpp.Node) (Reference, error) {
	if n.Name.Space != ReferenceNS || n.Name.Local != "reference" {
		return Reference{}, fmt.Errorf("xep: invalid reference")
	}
	r := Reference{}
	r.Type, _ = n.AttrValue("type")
	r.URI, _ = n.AttrValue("uri")
	r.Anchor, _ = n.AttrValue("anchor")
	for name, target := range map[string]**int{"begin": &r.Begin, "end": &r.End} {
		if v, ok := n.AttrValue(name); ok {
			i, err := strconv.Atoi(v)
			if err != nil {
				return r, err
			}
			*target = &i
		}
	}
	return r, nil
}
func AddReference(message *xmpp.Message, reference Reference) {
	message.Extensions = append(message.Extensions, reference.ToNode())
}
func References(message xmpp.Message) ([]Reference, error) {
	var out []Reference
	for _, n := range message.Extensions {
		if n.Name.Space == ReferenceNS && n.Name.Local == "reference" {
			v, err := ParseReference(n)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
	}
	return out, nil
}

type EncryptionInfo struct{ Namespace, Name string }

func SetEncryptionInfo(message *xmpp.Message, info EncryptionInfo) {
	removeExtension(message, EncryptionNS, "encryption")
	n := xmpp.NewNode(EncryptionNS, "encryption")
	n.SetAttr("namespace", info.Namespace)
	if info.Name != "" {
		n.SetAttr("name", info.Name)
	}
	message.Extensions = append(message.Extensions, n)
}
func GetEncryptionInfo(message xmpp.Message) (EncryptionInfo, bool) {
	n := message.Extension(EncryptionNS, "encryption")
	if n == nil {
		return EncryptionInfo{}, false
	}
	v := EncryptionInfo{}
	v.Namespace, _ = n.AttrValue("namespace")
	v.Name, _ = n.AttrValue("name")
	return v, true
}
func SetSpoiler(message *xmpp.Message, hint string) {
	removeExtension(message, SpoilerNS, "spoiler")
	message.Extensions = append(message.Extensions, xmpp.NewTextNode(SpoilerNS, "spoiler", hint))
}
func Spoiler(message xmpp.Message) (string, bool) {
	n := message.Extension(SpoilerNS, "spoiler")
	if n == nil {
		return "", false
	}
	return n.Text(), true
}
func SetOccupantID(message *xmpp.Message, id string) {
	removeExtension(message, OccupantIDNS, "occupant-id")
	n := xmpp.NewNode(OccupantIDNS, "occupant-id")
	n.SetAttr("id", id)
	message.Extensions = append(message.Extensions, n)
}
func OccupantID(message xmpp.Message) (string, bool) {
	n := message.Extension(OccupantIDNS, "occupant-id")
	if n == nil {
		return "", false
	}
	id, _ := n.AttrValue("id")
	return id, id != ""
}

type FallbackBody struct{ Start, End *int }
type Fallback struct {
	For    string
	Bodies []FallbackBody
}

func (f Fallback) ToNode() xmpp.Node {
	n := xmpp.NewNode(FallbackNS, "fallback")
	n.SetAttr("for", f.For)
	for _, b := range f.Bodies {
		x := xmpp.NewNode(FallbackNS, "body")
		if b.Start != nil {
			x.SetAttr("start", strconv.Itoa(*b.Start))
		}
		if b.End != nil {
			x.SetAttr("end", strconv.Itoa(*b.End))
		}
		n.AddChild(x)
	}
	return n
}
func AddFallback(message *xmpp.Message, f Fallback) {
	message.Extensions = append(message.Extensions, f.ToNode())
}
func Fallbacks(message xmpp.Message) ([]Fallback, error) {
	var out []Fallback
	for _, n := range message.Extensions {
		if n.Name.Space != FallbackNS || n.Name.Local != "fallback" {
			continue
		}
		f := Fallback{}
		f.For, _ = n.AttrValue("for")
		for _, b := range n.Children() {
			if b.Name.Local != "body" {
				continue
			}
			v := FallbackBody{}
			if x, ok := b.AttrValue("start"); ok {
				i, e := strconv.Atoi(x)
				if e != nil {
					return nil, e
				}
				v.Start = &i
			}
			if x, ok := b.AttrValue("end"); ok {
				i, e := strconv.Atoi(x)
				if e != nil {
					return nil, e
				}
				v.End = &i
			}
			f.Bodies = append(f.Bodies, v)
		}
		out = append(out, f)
	}
	return out, nil
}

// StripFallback removes the first fallback body range for namespace from Body.
func StripFallback(message xmpp.Message, namespace string) (xmpp.Message, error) {
	fallbacks, err := Fallbacks(message)
	if err != nil {
		return message, err
	}
	runes := []rune(message.Body)
	for _, f := range fallbacks {
		if f.For != namespace {
			continue
		}
		for _, b := range f.Bodies {
			start, end := 0, len(runes)
			if b.Start != nil {
				start = *b.Start
			}
			if b.End != nil {
				end = *b.End
			}
			if start < 0 || end < start || end > len(runes) {
				return message, fmt.Errorf("xep: fallback range %d..%d outside body", start, end)
			}
			message.Body = string(append(runes[:start], runes[end:]...))
			return message, nil
		}
	}
	return message, nil
}

type Reply struct{ To, ID string }

func SetReply(message *xmpp.Message, reply Reply, fallbackPrefix string) {
	removeExtension(message, ReplyNS, "reply")
	n := xmpp.NewNode(ReplyNS, "reply")
	n.SetAttr("to", reply.To)
	n.SetAttr("id", reply.ID)
	message.Extensions = append(message.Extensions, n)
	if fallbackPrefix != "" {
		end := len([]rune(fallbackPrefix))
		message.Body = fallbackPrefix + message.Body
		AddFallback(message, Fallback{For: ReplyNS, Bodies: []FallbackBody{{Start: Int(0), End: &end}}})
	}
}
func GetReply(message xmpp.Message) (Reply, bool) {
	n := message.Extension(ReplyNS, "reply")
	if n == nil {
		return Reply{}, false
	}
	r := Reply{}
	r.To, _ = n.AttrValue("to")
	r.ID, _ = n.AttrValue("id")
	return r, r.ID != ""
}
func SetRetraction(message *xmpp.Message, id, fallback string) {
	removeExtension(message, RetractionNS, "retract")
	n := xmpp.NewNode(RetractionNS, "retract")
	n.SetAttr("id", id)
	message.Extensions = append(message.Extensions, n)
	if fallback != "" {
		message.Body = fallback
		end := len([]rune(fallback))
		AddFallback(message, Fallback{For: RetractionNS, Bodies: []FallbackBody{{Start: Int(0), End: &end}}})
	}
}
func RetractionID(message xmpp.Message) (string, bool) {
	n := message.Extension(RetractionNS, "retract")
	if n == nil {
		return "", false
	}
	id, _ := n.AttrValue("id")
	return id, id != ""
}
func SetReactions(message *xmpp.Message, id string, reactions ...string) {
	removeExtension(message, ReactionsNS, "reactions")
	n := xmpp.NewNode(ReactionsNS, "reactions")
	n.SetAttr("id", id)
	seen := map[string]bool{}
	for _, reaction := range reactions {
		if reaction != "" && !seen[reaction] {
			seen[reaction] = true
			n.AddChild(xmpp.NewTextNode(ReactionsNS, "reaction", reaction))
		}
	}
	message.Extensions = append(message.Extensions, n)
}
func GetReactions(message xmpp.Message) (id string, reactions []string, ok bool) {
	n := message.Extension(ReactionsNS, "reactions")
	if n == nil {
		return "", nil, false
	}
	id, _ = n.AttrValue("id")
	for _, c := range n.Children() {
		if c.Name.Space == ReactionsNS && c.Name.Local == "reaction" {
			reactions = append(reactions, c.Text())
		}
	}
	return id, reactions, true
}

type Mention struct {
	URI        string
	Start, End *int
}

func (m Mention) ToNode() xmpp.Node {
	n := xmpp.NewNode(MentionNS, "mention")
	n.SetAttr("uri", m.URI)
	if m.Start != nil {
		n.SetAttr("start", strconv.Itoa(*m.Start))
	}
	if m.End != nil {
		n.SetAttr("end", strconv.Itoa(*m.End))
	}
	return n
}
func AddMention(message *xmpp.Message, mention Mention) {
	message.Extensions = append(message.Extensions, mention.ToNode())
}
func Mentions(message xmpp.Message) ([]Mention, error) {
	var out []Mention
	for _, n := range message.Extensions {
		if n.Name.Space != MentionNS || n.Name.Local != "mention" {
			continue
		}
		m := Mention{}
		m.URI, _ = n.AttrValue("uri")
		for name, target := range map[string]**int{"start": &m.Start, "end": &m.End} {
			if x, ok := n.AttrValue(name); ok {
				i, e := strconv.Atoi(x)
				if e != nil {
					return nil, e
				}
				*target = &i
			}
		}
		out = append(out, m)
	}
	return out, nil
}
func AddAttention(message *xmpp.Message) {
	if message.Extension(AttentionNS, "attention") == nil {
		message.Extensions = append(message.Extensions, xmpp.NewNode(AttentionNS, "attention"))
	}
}
func SetDirectMUCInvite(message *xmpp.Message, room, password, reason, thread string) {
	removeExtension(message, DirectMUCInviteNS, "x")
	n := xmpp.NewNode(DirectMUCInviteNS, "x")
	n.SetAttr("jid", room)
	if password != "" {
		n.SetAttr("password", password)
	}
	if reason != "" {
		n.SetAttr("reason", reason)
	}
	if thread != "" {
		n.SetAttr("thread", thread)
	}
	message.Extensions = append(message.Extensions, n)
}

type MarkupSpan struct {
	Start, End int
	Styles     []string
}
type Markup struct {
	Spans       []MarkupSpan
	BlockQuotes [][2]int
	Codes       [][2]int
}

func (m Markup) ToNode() xmpp.Node {
	n := xmpp.NewNode(MessageMarkupNS, "markup")
	for _, span := range m.Spans {
		s := xmpp.NewNode(MessageMarkupNS, "span")
		s.SetAttr("start", strconv.Itoa(span.Start))
		s.SetAttr("end", strconv.Itoa(span.End))
		for _, style := range span.Styles {
			s.AddChild(xmpp.NewNode(MessageMarkupNS, style))
		}
		n.AddChild(s)
	}
	for _, q := range m.BlockQuotes {
		x := xmpp.NewNode(MessageMarkupNS, "bquote")
		x.SetAttr("start", strconv.Itoa(q[0]))
		x.SetAttr("end", strconv.Itoa(q[1]))
		n.AddChild(x)
	}
	for _, q := range m.Codes {
		x := xmpp.NewNode(MessageMarkupNS, "code")
		x.SetAttr("start", strconv.Itoa(q[0]))
		x.SetAttr("end", strconv.Itoa(q[1]))
		n.AddChild(x)
	}
	return n
}

type CallInviteAction string

const (
	CallInvitePropose CallInviteAction = "propose"
	CallInviteAccept  CallInviteAction = "accept"
	CallInviteReject  CallInviteAction = "reject"
	CallInviteRetract CallInviteAction = "retract"
	CallInviteLeft    CallInviteAction = "left"
)

type CallInvite struct {
	Action       CallInviteAction
	ID, JID      string
	Video, Audio bool
}

func (c CallInvite) ToNode() xmpp.Node {
	n := xmpp.NewNode(CallInvitesNS, string(c.Action))
	n.SetAttr("id", c.ID)
	if c.JID != "" {
		n.SetAttr("jid", c.JID)
	}
	if c.Audio {
		n.AddChild(xmpp.NewNode(CallInvitesNS, "audio"))
	}
	if c.Video {
		n.AddChild(xmpp.NewNode(CallInvitesNS, "video"))
	}
	return n
}

func removeExtension(message *xmpp.Message, namespace string, locals ...string) {
	set := map[string]bool{}
	for _, v := range locals {
		set[v] = true
	}
	out := message.Extensions[:0]
	for _, n := range message.Extensions {
		if n.Name.Space == namespace && set[n.Name.Local] {
			continue
		}
		out = append(out, n)
	}
	message.Extensions = out
}
func asMessage(s xmpp.Stanza) xmpp.Message {
	switch v := s.(type) {
	case xmpp.Message:
		return v
	case *xmpp.Message:
		return *v
	}
	panic("xep: stanza is not a message")
}

type featurePlugin struct{ xmpp.BasicPlugin }

func staticPlugin(number int, features ...string) xmpp.PluginFactory {
	d, _ := Lookup(number)
	features = append([]string(nil), features...)
	sort.Strings(features)
	return func() xmpp.Plugin {
		return &featurePlugin{BasicPlugin: xmpp.BasicPlugin{PluginName: d.Name(), PluginDescription: d.XEP() + ": " + d.Title, PluginFeatures: features}}
	}
}
func init() {
	registerSpecialized(85, staticPlugin(85, ChatStatesNS))
	registerSpecialized(184, func() xmpp.Plugin { return NewReceipts() })
	registerSpecialized(224, staticPlugin(224, AttentionNS))
	registerSpecialized(249, staticPlugin(249, DirectMUCInviteNS))
	registerSpecialized(308, staticPlugin(308, CorrectionNS))
	registerSpecialized(333, staticPlugin(333, MarkersNS))
	registerSpecialized(334, staticPlugin(334, HintsNS))
	registerSpecialized(335, staticPlugin(335, JSONNS))
	registerSpecialized(359, staticPlugin(359, StanzaIDNS))
	registerSpecialized(372, staticPlugin(372, ReferenceNS))
	registerSpecialized(380, staticPlugin(380, EncryptionNS))
	registerSpecialized(382, staticPlugin(382, SpoilerNS))
	registerSpecialized(394, staticPlugin(394, MessageMarkupNS))
	registerSpecialized(421, staticPlugin(421, OccupantIDNS))
	registerSpecialized(424, staticPlugin(424, RetractionNS, FallbackNS))
	registerSpecialized(428, staticPlugin(428, FallbackNS))
	registerSpecialized(444, staticPlugin(444, ReactionsNS))
	registerSpecialized(461, staticPlugin(461, ReplyNS, FallbackNS))
	registerSpecialized(482, staticPlugin(482, CallInvitesNS))
	registerSpecialized(490, staticPlugin(490, DisplayedSyncNS))
	registerSpecialized(513, staticPlugin(513, MentionNS))
}

var _ = strings.Builder{}
