package xmpp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

const stanzaErrorNS = "urn:ietf:params:xml:ns:xmpp-stanzas"

// Stanza is implemented by Message, Presence, and IQ.
type Stanza interface {
	Kind() string
	StanzaID() string
	StanzaFrom() string
	StanzaTo() string
	MarshalXML(*xml.Encoder, xml.StartElement) error
}

// LangText is a localized stanza text value.
type LangText struct{ Lang, Value string }

// MessageType is the type attribute of a message stanza.
type MessageType string

const (
	MessageNormal    MessageType = "normal"
	MessageChat      MessageType = "chat"
	MessageGroupChat MessageType = "groupchat"
	MessageHeadline  MessageType = "headline"
	MessageError     MessageType = "error"
)

// Message is an XMPP <message/> stanza.
type Message struct {
	From, To, ID                        string
	Type                                MessageType
	Lang, Subject, Body, Thread, Parent string
	Subjects, Bodies                    []LangText
	Extensions                          []Node
	Error                               *StanzaError
}

func (Message) Kind() string                       { return "message" }
func (m Message) StanzaID() string                 { return m.ID }
func (m Message) StanzaFrom() string               { return m.From }
func (m Message) StanzaTo() string                 { return m.To }
func (m Message) GetID() string                    { return m.ID }
func (m Message) GetFrom() string                  { return m.From }
func (m Message) GetTo() string                    { return m.To }
func (m Message) Extension(ns, local string) *Node { return findExtension(m.Extensions, ns, local) }

// Reply creates a reply addressed to the sender or group-chat room.
func (m Message) Reply(body string) Message {
	kind := m.Type
	if kind == "" {
		kind = MessageNormal
	}
	to := m.From
	if kind == MessageGroupChat {
		to = BareJIDString(m.From)
	}
	return Message{To: to, Type: kind, Body: body, Thread: m.Thread}
}

// PresenceType is the type attribute of a presence stanza.
type PresenceType string

const (
	PresenceAvailable    PresenceType = ""
	PresenceUnavailable  PresenceType = "unavailable"
	PresenceSubscribe    PresenceType = "subscribe"
	PresenceSubscribed   PresenceType = "subscribed"
	PresenceUnsubscribe  PresenceType = "unsubscribe"
	PresenceUnsubscribed PresenceType = "unsubscribed"
	PresenceProbe        PresenceType = "probe"
	PresenceError        PresenceType = "error"
)

// Presence is an XMPP <presence/> stanza.
type Presence struct {
	From, To, ID       string
	Type               PresenceType
	Lang, Show, Status string
	Statuses           []LangText
	Priority           *int
	Extensions         []Node
	Error              *StanzaError
}

func (Presence) Kind() string                       { return "presence" }
func (p Presence) StanzaID() string                 { return p.ID }
func (p Presence) StanzaFrom() string               { return p.From }
func (p Presence) StanzaTo() string                 { return p.To }
func (p Presence) GetID() string                    { return p.ID }
func (p Presence) GetFrom() string                  { return p.From }
func (p Presence) GetTo() string                    { return p.To }
func (p Presence) Extension(ns, local string) *Node { return findExtension(p.Extensions, ns, local) }

// IQType is the type attribute of an IQ stanza.
type IQType string

const (
	IQGet    IQType = "get"
	IQSet    IQType = "set"
	IQResult IQType = "result"
	IQError  IQType = "error"
)

// IQ is an XMPP <iq/> stanza.
type IQ struct {
	From, To, ID string
	Type         IQType
	Payloads     []Node
	Error        *StanzaError
}

func (IQ) Kind() string         { return "iq" }
func (i IQ) StanzaID() string   { return i.ID }
func (i IQ) StanzaFrom() string { return i.From }
func (i IQ) StanzaTo() string   { return i.To }
func (i IQ) GetID() string      { return i.ID }
func (i IQ) GetFrom() string    { return i.From }
func (i IQ) GetTo() string      { return i.To }
func (i IQ) Payload() *Node {
	if len(i.Payloads) == 0 {
		return nil
	}
	n := i.Payloads[0].Clone()
	return &n
}
func (i IQ) Result(payloads ...Node) IQ {
	return IQ{To: i.From, From: i.To, ID: i.ID, Type: IQResult, Payloads: payloads}
}
func (i IQ) ErrorResult(e *StanzaError) IQ {
	return IQ{To: i.From, From: i.To, ID: i.ID, Type: IQError, Payloads: cloneNodes(i.Payloads), Error: e}
}

// StanzaXML serializes a stanza without an XML declaration.
func StanzaXML(stanza Stanza) ([]byte, error) {
	var b bytes.Buffer
	enc := xml.NewEncoder(&b)
	if err := enc.Encode(stanza); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (m Message) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	start := stanzaStart("message", m.From, m.To, m.ID, string(m.Type), m.Lang)
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	if m.Subject != "" {
		if err := encodeLangText(enc, "subject", LangText{Value: m.Subject}); err != nil {
			return err
		}
	}
	for _, v := range m.Subjects {
		if v.Lang == "" && v.Value == m.Subject {
			continue
		}
		if err := encodeLangText(enc, "subject", v); err != nil {
			return err
		}
	}
	if m.Body != "" {
		if err := encodeLangText(enc, "body", LangText{Value: m.Body}); err != nil {
			return err
		}
	}
	for _, v := range m.Bodies {
		if v.Lang == "" && v.Value == m.Body {
			continue
		}
		if err := encodeLangText(enc, "body", v); err != nil {
			return err
		}
	}
	if m.Thread != "" {
		s := xml.StartElement{Name: xml.Name{Local: "thread"}}
		if m.Parent != "" {
			s.Attr = append(s.Attr, xml.Attr{Name: xml.Name{Local: "parent"}, Value: m.Parent})
		}
		if err := enc.EncodeElement(m.Thread, s); err != nil {
			return err
		}
	}
	if err := encodeNodes(enc, m.Extensions); err != nil {
		return err
	}
	if err := encodeStanzaError(enc, m.Error); err != nil {
		return err
	}
	return enc.EncodeToken(start.End())
}

func (p Presence) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	start := stanzaStart("presence", p.From, p.To, p.ID, string(p.Type), p.Lang)
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	if p.Show != "" {
		if err := enc.EncodeElement(p.Show, xml.StartElement{Name: xml.Name{Local: "show"}}); err != nil {
			return err
		}
	}
	if p.Status != "" {
		if err := encodeLangText(enc, "status", LangText{Value: p.Status}); err != nil {
			return err
		}
	}
	for _, v := range p.Statuses {
		if v.Lang == "" && v.Value == p.Status {
			continue
		}
		if err := encodeLangText(enc, "status", v); err != nil {
			return err
		}
	}
	if p.Priority != nil {
		if *p.Priority < -128 || *p.Priority > 127 {
			return fmt.Errorf("xmpp: presence priority %d is outside -128..127", *p.Priority)
		}
		if err := enc.EncodeElement(strconv.Itoa(*p.Priority), xml.StartElement{Name: xml.Name{Local: "priority"}}); err != nil {
			return err
		}
	}
	if err := encodeNodes(enc, p.Extensions); err != nil {
		return err
	}
	if err := encodeStanzaError(enc, p.Error); err != nil {
		return err
	}
	return enc.EncodeToken(start.End())
}

func (i IQ) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	if i.ID == "" {
		return fmt.Errorf("xmpp: IQ must have an id")
	}
	if i.Type == "" {
		return fmt.Errorf("xmpp: IQ must have a type")
	}
	start := stanzaStart("iq", i.From, i.To, i.ID, string(i.Type), "")
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	if err := encodeNodes(enc, i.Payloads); err != nil {
		return err
	}
	if err := encodeStanzaError(enc, i.Error); err != nil {
		return err
	}
	return enc.EncodeToken(start.End())
}

// DecodeStanza decodes one already-open stanza element.
func DecodeStanza(dec *xml.Decoder, start xml.StartElement) (Stanza, error) {
	var root Node
	if err := dec.DecodeElement(&root, &start); err != nil {
		return nil, err
	}
	return stanzaFromNode(root)
}

// ParseStanza parses exactly one stanza.
func ParseStanza(data []byte) (Stanza, error) {
	node, err := ParseNode(data)
	if err != nil {
		return nil, err
	}
	return stanzaFromNode(node)
}

func stanzaFromNode(root Node) (Stanza, error) {
	attr := func(name string) string { v, _ := root.AttrValue(name); return v }
	switch root.Name.Local {
	case "message":
		m := Message{From: attr("from"), To: attr("to"), ID: attr("id"), Type: MessageType(attr("type"))}
		m.Lang, _ = root.AttrValueNS("http://www.w3.org/XML/1998/namespace", "lang")
		for _, c := range root.Children() {
			switch c.Name.Local {
			case "subject":
				v := langTextFromNode(c)
				m.Subjects = append(m.Subjects, v)
				if v.Lang == "" && m.Subject == "" {
					m.Subject = v.Value
				}
			case "body":
				v := langTextFromNode(c)
				m.Bodies = append(m.Bodies, v)
				if v.Lang == "" && m.Body == "" {
					m.Body = v.Value
				}
			case "thread":
				m.Thread = c.Text()
				m.Parent, _ = c.AttrValue("parent")
			case "error":
				m.Error = stanzaErrorFromNode(c)
			default:
				m.Extensions = append(m.Extensions, c)
			}
		}
		return m, nil
	case "presence":
		p := Presence{From: attr("from"), To: attr("to"), ID: attr("id"), Type: PresenceType(attr("type"))}
		p.Lang, _ = root.AttrValueNS("http://www.w3.org/XML/1998/namespace", "lang")
		for _, c := range root.Children() {
			switch c.Name.Local {
			case "show":
				p.Show = c.Text()
			case "status":
				v := langTextFromNode(c)
				p.Statuses = append(p.Statuses, v)
				if v.Lang == "" && p.Status == "" {
					p.Status = v.Value
				}
			case "priority":
				if v, e := strconv.Atoi(strings.TrimSpace(c.Text())); e == nil {
					p.Priority = &v
				}
			case "error":
				p.Error = stanzaErrorFromNode(c)
			default:
				p.Extensions = append(p.Extensions, c)
			}
		}
		return p, nil
	case "iq":
		i := IQ{From: attr("from"), To: attr("to"), ID: attr("id"), Type: IQType(attr("type"))}
		for _, c := range root.Children() {
			if c.Name.Local == "error" {
				i.Error = stanzaErrorFromNode(c)
			} else {
				i.Payloads = append(i.Payloads, c)
			}
		}
		return i, nil
	default:
		return nil, fmt.Errorf("xmpp: unsupported stanza <%s>", root.Name.Local)
	}
}

func stanzaStart(local, from, to, id, kind, lang string) xml.StartElement {
	s := xml.StartElement{Name: xml.Name{Local: local}}
	add := func(name, value string) {
		if value != "" {
			s.Attr = append(s.Attr, xml.Attr{Name: xml.Name{Local: name}, Value: value})
		}
	}
	add("from", from)
	add("to", to)
	add("id", id)
	add("type", kind)
	if lang != "" {
		s.Attr = append(s.Attr, xml.Attr{Name: xml.Name{Space: "http://www.w3.org/XML/1998/namespace", Local: "lang"}, Value: lang})
	}
	return s
}
func encodeLangText(enc *xml.Encoder, local string, v LangText) error {
	s := xml.StartElement{Name: xml.Name{Local: local}}
	if v.Lang != "" {
		s.Attr = append(s.Attr, xml.Attr{Name: xml.Name{Space: "http://www.w3.org/XML/1998/namespace", Local: "lang"}, Value: v.Lang})
	}
	return enc.EncodeElement(v.Value, s)
}
func langTextFromNode(n Node) LangText {
	l, _ := n.AttrValueNS("http://www.w3.org/XML/1998/namespace", "lang")
	return LangText{Lang: l, Value: n.Text()}
}
func encodeNodes(enc *xml.Encoder, nodes []Node) error {
	for i := range nodes {
		if err := enc.Encode(nodes[i]); err != nil {
			return err
		}
	}
	return nil
}
func encodeStanzaError(enc *xml.Encoder, e *StanzaError) error {
	if e == nil {
		return nil
	}
	n := NewNode("", "error")
	n.SetAttr("type", e.Type)
	if e.By != "" {
		n.SetAttr("by", e.By)
	}
	if e.Condition != "" {
		n.AddChild(NewNode(stanzaErrorNS, e.Condition))
	}
	if e.Text != "" {
		n.AddChild(NewTextNode(stanzaErrorNS, "text", e.Text))
	}
	for _, a := range e.Application {
		n.AddChild(a)
	}
	return enc.Encode(n)
}
func stanzaErrorFromNode(n Node) *StanzaError {
	e := &StanzaError{}
	e.Type, _ = n.AttrValue("type")
	e.By, _ = n.AttrValue("by")
	for _, c := range n.Children() {
		if c.Name.Space == stanzaErrorNS {
			if c.Name.Local == "text" {
				e.Text = c.Text()
			} else {
				e.Condition = c.Name.Local
			}
		} else {
			e.Application = append(e.Application, c)
		}
	}
	return e
}
func findExtension(nodes []Node, ns, local string) *Node {
	for _, n := range nodes {
		if n.Name.Local == local && (ns == "" || n.Name.Space == ns) {
			c := n.Clone()
			return &c
		}
	}
	return nil
}
func cloneNodes(nodes []Node) []Node {
	out := make([]Node, len(nodes))
	for i := range nodes {
		out[i] = nodes[i].Clone()
	}
	return out
}

// BareJIDString removes the first resource separator from a JID string.
func BareJIDString(value string) string {
	if slash := strings.IndexByte(value, '/'); slash >= 0 {
		return value[:slash]
	}
	return value
}
