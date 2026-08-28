package xep

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/saret/slixmpp-go/xmpp"
)

const (
	MAMNS            = "urn:xmpp:mam:2"
	MAMPreferencesNS = "urn:xmpp:mam:2"
)

type MAMQuery struct {
	Service, Node, With, Search string
	Start, End                  *time.Time
	IDs                         []string
	Form                        *Form
	RSM                         *RSM
}
type MAMResult struct {
	QueryID, ID string
	Forwarded   Forwarded
	Message     xmpp.Message
}
type MAMFin struct {
	Complete, Stable bool
	RSM              *RSM
}
type MAMPage struct {
	Results []MAMResult
	Fin     MAMFin
}
type mamPending struct{ results []MAMResult }
type MAM struct {
	client  *xmpp.Client
	mu      sync.Mutex
	pending map[string]*mamPending
	handler *xmpp.Handler
}

func NewMAM() *MAM                    { return &MAM{pending: make(map[string]*mamPending)} }
func (m *MAM) Name() string           { return "xep_0313" }
func (m *MAM) Description() string    { return "XEP-0313 Message Archive Management" }
func (m *MAM) Dependencies() []string { return []string{"xep_0004", "xep_0059", "xep_0297"} }
func (m *MAM) Features() []string     { return []string{MAMNS} }
func (m *MAM) Init(c *xmpp.Client) error {
	m.client = c
	if m.pending == nil {
		m.pending = make(map[string]*mamPending)
	}
	m.handler = c.AddHandler("mam-result", xmpp.MatchAnd(xmpp.MatchKind("message"), xmpp.MatchPayload(MAMNS, "result")), m.handleResult)
	return nil
}
func (m *MAM) Shutdown(context.Context) error {
	if m.client != nil {
		m.client.RemoveHandler(m.handler)
	}
	return nil
}
func (m *MAM) Query(ctx context.Context, query MAMQuery) (MAMPage, error) {
	queryID := m.client.NextID()
	pending := &mamPending{}
	m.mu.Lock()
	m.pending[queryID] = pending
	m.mu.Unlock()
	defer func() { m.mu.Lock(); delete(m.pending, queryID); m.mu.Unlock() }()
	payload := xmpp.NewNode(MAMNS, "query")
	payload.SetAttr("queryid", queryID)
	if query.Node != "" {
		payload.SetAttr("node", query.Node)
	}
	form := buildMAMForm(query)
	payload.AddChild(form.ToNode())
	if query.RSM != nil {
		payload.AddChild(query.RSM.ToNode())
	}
	response, err := m.client.RequestIQ(ctx, xmpp.IQ{To: query.Service, Type: xmpp.IQSet, Payloads: []xmpp.Node{payload}})
	if err != nil {
		return MAMPage{}, err
	}
	finNode := response.Payload()
	if finNode == nil || finNode.Name.Space != MAMNS || finNode.Name.Local != "fin" {
		return MAMPage{}, fmt.Errorf("xep: missing MAM fin response")
	}
	fin, err := parseMAMFin(*finNode)
	if err != nil {
		return MAMPage{}, err
	}
	m.mu.Lock()
	results := append([]MAMResult(nil), pending.results...)
	m.mu.Unlock()
	return MAMPage{Results: results, Fin: fin}, nil
}
func buildMAMForm(q MAMQuery) Form {
	if q.Form != nil {
		f := *q.Form
		f.Type = FormTypeSubmit
		if field := f.Field("FORM_TYPE"); field == nil {
			f.Fields = append([]Field{{Var: "FORM_TYPE", Type: FieldHidden, Values: []string{MAMNS}}}, f.Fields...)
		}
		return f
	}
	f := Form{Type: FormTypeSubmit, Fields: []Field{{Var: "FORM_TYPE", Type: FieldHidden, Values: []string{MAMNS}}}}
	if q.With != "" {
		f.Fields = append(f.Fields, Field{Var: "with", Type: FieldJIDSingle, Values: []string{q.With}})
	}
	if q.Start != nil {
		f.Fields = append(f.Fields, Field{Var: "start", Type: FieldTextSingle, Values: []string{FormatDateTime(*q.Start)}})
	}
	if q.End != nil {
		f.Fields = append(f.Fields, Field{Var: "end", Type: FieldTextSingle, Values: []string{FormatDateTime(*q.End)}})
	}
	if q.Search != "" {
		f.Fields = append(f.Fields, Field{Var: "full-text-search", Type: FieldTextSingle, Values: []string{q.Search}})
	}
	if len(q.IDs) > 0 {
		f.Fields = append(f.Fields, Field{Var: "ids", Type: FieldTextMulti, Values: append([]string(nil), q.IDs...)})
	}
	return f
}
func (m *MAM) handleResult(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
	message := asMessage(s)
	node := message.Extension(MAMNS, "result")
	result, err := parseMAMResult(*node)
	if err != nil {
		return err
	}
	result.Message = message
	m.mu.Lock()
	if pending := m.pending[result.QueryID]; pending != nil {
		pending.results = append(pending.results, result)
	}
	m.mu.Unlock()
	_ = c.Events.Emit(ctx, "mam_result", result)
	return nil
}
func parseMAMResult(n xmpp.Node) (MAMResult, error) {
	if n.Name.Space != MAMNS || n.Name.Local != "result" {
		return MAMResult{}, fmt.Errorf("xep: invalid MAM result")
	}
	r := MAMResult{}
	r.QueryID, _ = n.AttrValue("queryid")
	r.ID, _ = n.AttrValue("id")
	forwarded := n.Child(ForwardNS, "forwarded")
	if forwarded == nil {
		return r, fmt.Errorf("xep: MAM result has no forwarded stanza")
	}
	v, err := ParseForwarded(*forwarded)
	if err != nil {
		return r, err
	}
	r.Forwarded = v
	return r, nil
}
func parseMAMFin(n xmpp.Node) (MAMFin, error) {
	f := MAMFin{}
	if v, ok := n.AttrValue("complete"); ok {
		f.Complete = v == "true" || v == "1"
	}
	if v, ok := n.AttrValue("stable"); ok {
		f.Stable = v == "true" || v == "1"
	}
	if set := n.Child(RSMNS, "set"); set != nil {
		r, err := ParseRSM(*set)
		if err != nil {
			return f, err
		}
		f.RSM = &r
	}
	return f, nil
}

type MAMPreferences struct {
	Default       string
	Always, Never []string
}

func (p MAMPreferences) ToNode(kind xmpp.IQType) xmpp.Node {
	n := xmpp.NewNode(MAMPreferencesNS, "prefs")
	if p.Default != "" {
		n.SetAttr("default", p.Default)
	}
	if len(p.Always) > 0 {
		a := xmpp.NewNode(MAMPreferencesNS, "always")
		for _, jid := range p.Always {
			a.AddChild(xmpp.NewTextNode(MAMPreferencesNS, "jid", jid))
		}
		n.AddChild(a)
	}
	if len(p.Never) > 0 {
		a := xmpp.NewNode(MAMPreferencesNS, "never")
		for _, jid := range p.Never {
			a.AddChild(xmpp.NewTextNode(MAMPreferencesNS, "jid", jid))
		}
		n.AddChild(a)
	}
	return n
}
func (m *MAM) GetPreferences(ctx context.Context, service string) (MAMPreferences, error) {
	response, err := m.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQGet, Payloads: []xmpp.Node{xmpp.NewNode(MAMPreferencesNS, "prefs")}})
	if err != nil {
		return MAMPreferences{}, err
	}
	p := response.Payload()
	if p == nil {
		return MAMPreferences{}, fmt.Errorf("xep: missing MAM preferences")
	}
	return parseMAMPreferences(*p), nil
}
func (m *MAM) SetPreferences(ctx context.Context, service string, prefs MAMPreferences) error {
	_, err := m.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQSet, Payloads: []xmpp.Node{prefs.ToNode(xmpp.IQSet)}})
	return err
}
func parseMAMPreferences(n xmpp.Node) MAMPreferences {
	p := MAMPreferences{}
	p.Default, _ = n.AttrValue("default")
	if a := n.Child(MAMPreferencesNS, "always"); a != nil {
		for _, j := range a.Children() {
			if j.Name.Local == "jid" {
				p.Always = append(p.Always, j.Text())
			}
		}
	}
	if a := n.Child(MAMPreferencesNS, "never"); a != nil {
		for _, j := range a.Children() {
			if j.Name.Local == "jid" {
				p.Never = append(p.Never, j.Text())
			}
		}
	}
	return p
}
func init() {
	registerSpecialized(313, func() xmpp.Plugin { return NewMAM() })
	registerSpecialized(441, staticPlugin(441, MAMPreferencesNS))
}
