package xep

import (
	"context"
	"fmt"
	"github.com/saret/slixmpp-go/xmpp"
	"time"
)

const (
	VersionNS    = "jabber:iq:version"
	PingNS       = "urn:xmpp:ping"
	EntityTimeNS = "urn:xmpp:time"
)

// VersionInfo is an XEP-0092 software-version result.
type VersionInfo struct{ Name, Version, OS string }
type Version struct {
	client  *xmpp.Client
	Info    VersionInfo
	handler *xmpp.Handler
}

func NewVersion(info VersionInfo) *Version { return &Version{Info: info} }
func (v *Version) Name() string            { return "xep_0092" }
func (v *Version) Description() string     { return "XEP-0092 Software Version" }
func (v *Version) Dependencies() []string  { return nil }
func (v *Version) Features() []string      { return []string{VersionNS} }
func (v *Version) Init(c *xmpp.Client) error {
	v.client = c
	if v.Info.Name == "" {
		v.Info.Name = "slixmpp-go"
	}
	v.handler = c.AddHandler("software-version", xmpp.MatchAnd(xmpp.MatchIQType(xmpp.IQGet), xmpp.MatchPayload(VersionNS, "query")), func(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
		iq := asIQ(s)
		return c.ReplyIQ(iq, versionNode(v.Info))
	})
	return nil
}
func (v *Version) Shutdown(context.Context) error {
	if v.client != nil {
		v.client.RemoveHandler(v.handler)
	}
	return nil
}
func (v *Version) Query(ctx context.Context, jid string) (VersionInfo, error) {
	q := xmpp.NewNode(VersionNS, "query")
	response, err := v.client.RequestIQ(ctx, xmpp.IQ{To: jid, Type: xmpp.IQGet, Payloads: []xmpp.Node{q}})
	if err != nil {
		return VersionInfo{}, err
	}
	p := response.Payload()
	if p == nil {
		return VersionInfo{}, fmt.Errorf("xep: missing version response")
	}
	return VersionInfo{Name: p.ChildText(VersionNS, "name"), Version: p.ChildText(VersionNS, "version"), OS: p.ChildText(VersionNS, "os")}, nil
}
func versionNode(info VersionInfo) xmpp.Node {
	q := xmpp.NewNode(VersionNS, "query")
	if info.Name != "" {
		q.AddChild(xmpp.NewTextNode(VersionNS, "name", info.Name))
	}
	if info.Version != "" {
		q.AddChild(xmpp.NewTextNode(VersionNS, "version", info.Version))
	}
	if info.OS != "" {
		q.AddChild(xmpp.NewTextNode(VersionNS, "os", info.OS))
	}
	return q
}

type Ping struct {
	client  *xmpp.Client
	handler *xmpp.Handler
}

func NewPing() *Ping                   { return &Ping{} }
func (p *Ping) Name() string           { return "xep_0199" }
func (p *Ping) Description() string    { return "XEP-0199 XMPP Ping" }
func (p *Ping) Dependencies() []string { return nil }
func (p *Ping) Features() []string     { return []string{PingNS} }
func (p *Ping) Init(c *xmpp.Client) error {
	p.client = c
	p.handler = c.AddHandler("ping", xmpp.MatchAnd(xmpp.MatchIQType(xmpp.IQGet), xmpp.MatchPayload(PingNS, "ping")), func(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error { return c.ReplyIQ(asIQ(s)) })
	return nil
}
func (p *Ping) Shutdown(context.Context) error {
	if p.client != nil {
		p.client.RemoveHandler(p.handler)
	}
	return nil
}
func (p *Ping) Ping(ctx context.Context, jid string) (time.Duration, error) {
	start := time.Now()
	_, err := p.client.RequestIQ(ctx, xmpp.IQ{To: jid, Type: xmpp.IQGet, Payloads: []xmpp.Node{xmpp.NewNode(PingNS, "ping")}})
	return time.Since(start), err
}

type EntityTime struct {
	client  *xmpp.Client
	handler *xmpp.Handler
	Now     func() time.Time
}
type TimeInfo struct {
	UTC time.Time
	TZD string
}

func NewEntityTime() *EntityTime             { return &EntityTime{Now: time.Now} }
func (e *EntityTime) Name() string           { return "xep_0202" }
func (e *EntityTime) Description() string    { return "XEP-0202 Entity Time" }
func (e *EntityTime) Dependencies() []string { return nil }
func (e *EntityTime) Features() []string     { return []string{EntityTimeNS} }
func (e *EntityTime) Init(c *xmpp.Client) error {
	e.client = c
	if e.Now == nil {
		e.Now = time.Now
	}
	e.handler = c.AddHandler("entity-time", xmpp.MatchAnd(xmpp.MatchIQType(xmpp.IQGet), xmpp.MatchPayload(EntityTimeNS, "time")), func(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
		return c.ReplyIQ(asIQ(s), timeNode(e.Now()))
	})
	return nil
}
func (e *EntityTime) Shutdown(context.Context) error {
	if e.client != nil {
		e.client.RemoveHandler(e.handler)
	}
	return nil
}
func (e *EntityTime) Query(ctx context.Context, jid string) (TimeInfo, error) {
	response, err := e.client.RequestIQ(ctx, xmpp.IQ{To: jid, Type: xmpp.IQGet, Payloads: []xmpp.Node{xmpp.NewNode(EntityTimeNS, "time")}})
	if err != nil {
		return TimeInfo{}, err
	}
	p := response.Payload()
	if p == nil {
		return TimeInfo{}, fmt.Errorf("xep: missing entity time")
	}
	utc, err := ParseDateTime(p.ChildText(EntityTimeNS, "utc"))
	if err != nil {
		return TimeInfo{}, err
	}
	return TimeInfo{UTC: utc, TZD: p.ChildText(EntityTimeNS, "tzo")}, nil
}
func timeNode(now time.Time) xmpp.Node {
	_, offset := now.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	tzo := fmt.Sprintf("%s%02d:%02d", sign, offset/3600, (offset%3600)/60)
	n := xmpp.NewNode(EntityTimeNS, "time")
	n.AddChild(xmpp.NewTextNode(EntityTimeNS, "tzo", tzo))
	n.AddChild(xmpp.NewTextNode(EntityTimeNS, "utc", FormatDateTime(now)))
	return n
}
func init() {
	registerSpecialized(92, func() xmpp.Plugin { return NewVersion(VersionInfo{}) })
	registerSpecialized(199, func() xmpp.Plugin { return NewPing() })
	registerSpecialized(202, func() xmpp.Plugin { return NewEntityTime() })
}
