package xep

import (
	"context"
	"fmt"
	"github.com/1kamma/slixmpp-go/xmpp"
	"sort"
	"sync"
)

const (
	DiscoInfoNS  = "http://jabber.org/protocol/disco#info"
	DiscoItemsNS = "http://jabber.org/protocol/disco#items"
)

type Identity struct{ Category, Type, Name, Lang string }
type DiscoItem struct{ JID, Node, Name string }
type DiscoInfo struct {
	Node       string
	Identities []Identity
	Features   []string
	Forms      []Form
}
type DiscoItems struct {
	Node  string
	Items []DiscoItem
	RSM   *RSM
}
type Disco struct {
	client   *xmpp.Client
	mu       sync.RWMutex
	info     map[string]DiscoInfo
	items    map[string]DiscoItems
	handlers []*xmpp.Handler
}

func NewDisco() *Disco {
	return &Disco{info: make(map[string]DiscoInfo), items: make(map[string]DiscoItems)}
}
func (d *Disco) Name() string           { return "xep_0030" }
func (d *Disco) Description() string    { return "XEP-0030 Service Discovery" }
func (d *Disco) Dependencies() []string { return []string{"xep_0004", "xep_0059"} }
func (d *Disco) Features() []string     { return []string{DiscoInfoNS, DiscoItemsNS} }
func (d *Disco) Init(c *xmpp.Client) error {
	d.client = c
	if d.info == nil {
		d.info = make(map[string]DiscoInfo)
	}
	if d.items == nil {
		d.items = make(map[string]DiscoItems)
	}
	d.mu.Lock()
	root := d.info[""]
	if len(root.Identities) == 0 {
		root.Identities = []Identity{{Category: "client", Type: "bot", Name: "slixmpp-go"}}
	}
	d.info[""] = root
	d.mu.Unlock()
	d.handlers = append(d.handlers, c.AddHandler("disco-info", xmpp.MatchAnd(xmpp.MatchIQType(xmpp.IQGet), xmpp.MatchPayload(DiscoInfoNS, "query")), d.handleInfo), c.AddHandler("disco-items", xmpp.MatchAnd(xmpp.MatchIQType(xmpp.IQGet), xmpp.MatchPayload(DiscoItemsNS, "query")), d.handleItems))
	_ = c.API.Register(d.Name(), "get_info", "", "", func(ctx context.Context, call xmpp.APICall) (any, error) { return d.GetInfo(ctx, call.JID, call.Node) })
	_ = c.API.Register(d.Name(), "get_items", "", "", func(ctx context.Context, call xmpp.APICall) (any, error) {
		return d.GetItems(ctx, call.JID, call.Node, nil)
	})
	return nil
}
func (d *Disco) Shutdown(context.Context) error {
	if d.client != nil {
		for _, h := range d.handlers {
			d.client.RemoveHandler(h)
		}
		d.client.API.Purge(d.Name())
	}
	return nil
}
func (d *Disco) AddIdentity(node string, id Identity) {
	d.mu.Lock()
	info := d.info[node]
	info.Node = node
	info.Identities = append(info.Identities, id)
	d.info[node] = info
	d.mu.Unlock()
}
func (d *Disco) AddFeature(node, feature string) {
	d.mu.Lock()
	info := d.info[node]
	info.Node = node
	for _, v := range info.Features {
		if v == feature {
			d.mu.Unlock()
			return
		}
	}
	info.Features = append(info.Features, feature)
	d.info[node] = info
	d.mu.Unlock()
}
func (d *Disco) RemoveFeature(node, feature string) {
	d.mu.Lock()
	info := d.info[node]
	for i, v := range info.Features {
		if v == feature {
			info.Features = append(info.Features[:i], info.Features[i+1:]...)
			break
		}
	}
	d.info[node] = info
	d.mu.Unlock()
}
func (d *Disco) SetInfo(node string, info DiscoInfo) {
	info.Node = node
	d.mu.Lock()
	d.info[node] = cloneDiscoInfo(info)
	d.mu.Unlock()
}
func (d *Disco) SetItems(node string, items DiscoItems) {
	items.Node = node
	d.mu.Lock()
	d.items[node] = cloneDiscoItems(items)
	d.mu.Unlock()
}
func (d *Disco) AddItem(node string, item DiscoItem) {
	d.mu.Lock()
	items := d.items[node]
	items.Node = node
	items.Items = append(items.Items, item)
	d.items[node] = items
	d.mu.Unlock()
}
func (d *Disco) LocalInfo(node string) (DiscoInfo, bool) {
	d.mu.RLock()
	info, ok := d.info[node]
	d.mu.RUnlock()
	if !ok {
		return DiscoInfo{}, false
	}
	if node == "" && d.client != nil {
		for _, f := range d.client.Plugins.Features() {
			found := false
			for _, x := range info.Features {
				if x == f {
					found = true
					break
				}
			}
			if !found {
				info.Features = append(info.Features, f)
			}
		}
	}
	sort.Strings(info.Features)
	return cloneDiscoInfo(info), true
}
func (d *Disco) LocalItems(node string) (DiscoItems, bool) {
	d.mu.RLock()
	items, ok := d.items[node]
	d.mu.RUnlock()
	return cloneDiscoItems(items), ok
}
func (d *Disco) GetInfo(ctx context.Context, jid, node string) (DiscoInfo, error) {
	query := xmpp.NewNode(DiscoInfoNS, "query")
	if node != "" {
		query.SetAttr("node", node)
	}
	response, err := d.client.RequestIQ(ctx, xmpp.IQ{To: jid, Type: xmpp.IQGet, Payloads: []xmpp.Node{query}})
	if err != nil {
		return DiscoInfo{}, err
	}
	payload := response.Payload()
	if payload == nil {
		return DiscoInfo{}, fmt.Errorf("xep: missing disco#info payload")
	}
	return parseDiscoInfo(*payload)
}
func (d *Disco) GetItems(ctx context.Context, jid, node string, rsm *RSM) (DiscoItems, error) {
	query := xmpp.NewNode(DiscoItemsNS, "query")
	if node != "" {
		query.SetAttr("node", node)
	}
	if rsm != nil {
		query.AddChild(rsm.ToNode())
	}
	response, err := d.client.RequestIQ(ctx, xmpp.IQ{To: jid, Type: xmpp.IQGet, Payloads: []xmpp.Node{query}})
	if err != nil {
		return DiscoItems{}, err
	}
	payload := response.Payload()
	if payload == nil {
		return DiscoItems{}, fmt.Errorf("xep: missing disco#items payload")
	}
	return parseDiscoItems(*payload)
}
func (d *Disco) handleInfo(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
	iq := asIQ(s)
	payload := iq.Payload()
	node, _ := payload.AttrValue("node")
	info, ok := d.LocalInfo(node)
	if !ok {
		return c.Send(iq.ErrorResult(&xmpp.StanzaError{Type: "cancel", Condition: "item-not-found"}))
	}
	return c.ReplyIQ(iq, discoInfoNode(info))
}
func (d *Disco) handleItems(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
	iq := asIQ(s)
	payload := iq.Payload()
	node, _ := payload.AttrValue("node")
	items, ok := d.LocalItems(node)
	if !ok && node != "" {
		return c.Send(iq.ErrorResult(&xmpp.StanzaError{Type: "cancel", Condition: "item-not-found"}))
	}
	return c.ReplyIQ(iq, discoItemsNode(items))
}
func discoInfoNode(info DiscoInfo) xmpp.Node {
	q := xmpp.NewNode(DiscoInfoNS, "query")
	if info.Node != "" {
		q.SetAttr("node", info.Node)
	}
	for _, id := range info.Identities {
		n := xmpp.NewNode(DiscoInfoNS, "identity")
		n.SetAttr("category", id.Category)
		n.SetAttr("type", id.Type)
		if id.Name != "" {
			n.SetAttr("name", id.Name)
		}
		if id.Lang != "" {
			n.SetAttrNS("http://www.w3.org/XML/1998/namespace", "lang", id.Lang)
		}
		q.AddChild(n)
	}
	features := append([]string(nil), info.Features...)
	sort.Strings(features)
	for _, f := range features {
		n := xmpp.NewNode(DiscoInfoNS, "feature")
		n.SetAttr("var", f)
		q.AddChild(n)
	}
	for _, form := range info.Forms {
		q.AddChild(form.ToNode())
	}
	return q
}
func discoItemsNode(items DiscoItems) xmpp.Node {
	q := xmpp.NewNode(DiscoItemsNS, "query")
	if items.Node != "" {
		q.SetAttr("node", items.Node)
	}
	for _, item := range items.Items {
		n := xmpp.NewNode(DiscoItemsNS, "item")
		n.SetAttr("jid", item.JID)
		if item.Node != "" {
			n.SetAttr("node", item.Node)
		}
		if item.Name != "" {
			n.SetAttr("name", item.Name)
		}
		q.AddChild(n)
	}
	if items.RSM != nil {
		q.AddChild(items.RSM.ToNode())
	}
	return q
}
func parseDiscoInfo(q xmpp.Node) (DiscoInfo, error) {
	if q.Name.Space != DiscoInfoNS || q.Name.Local != "query" {
		return DiscoInfo{}, fmt.Errorf("xep: invalid disco#info payload")
	}
	var info DiscoInfo
	info.Node, _ = q.AttrValue("node")
	for _, n := range q.Children() {
		switch n.Name.Local {
		case "identity":
			id := Identity{}
			id.Category, _ = n.AttrValue("category")
			id.Type, _ = n.AttrValue("type")
			id.Name, _ = n.AttrValue("name")
			id.Lang, _ = n.AttrValueNS("http://www.w3.org/XML/1998/namespace", "lang")
			info.Identities = append(info.Identities, id)
		case "feature":
			if v, ok := n.AttrValue("var"); ok {
				info.Features = append(info.Features, v)
			}
		case "x":
			if n.Name.Space == DataFormsNS {
				form, err := ParseForm(n)
				if err != nil {
					return DiscoInfo{}, err
				}
				info.Forms = append(info.Forms, form)
			}
		}
	}
	return info, nil
}
func parseDiscoItems(q xmpp.Node) (DiscoItems, error) {
	if q.Name.Space != DiscoItemsNS || q.Name.Local != "query" {
		return DiscoItems{}, fmt.Errorf("xep: invalid disco#items payload")
	}
	var out DiscoItems
	out.Node, _ = q.AttrValue("node")
	for _, n := range q.Children() {
		if n.Name.Space == RSMNS && n.Name.Local == "set" {
			r, err := ParseRSM(n)
			if err != nil {
				return DiscoItems{}, err
			}
			out.RSM = &r
			continue
		}
		if n.Name.Local == "item" {
			item := DiscoItem{}
			item.JID, _ = n.AttrValue("jid")
			item.Node, _ = n.AttrValue("node")
			item.Name, _ = n.AttrValue("name")
			out.Items = append(out.Items, item)
		}
	}
	return out, nil
}
func cloneDiscoInfo(v DiscoInfo) DiscoInfo {
	v.Identities = append([]Identity(nil), v.Identities...)
	v.Features = append([]string(nil), v.Features...)
	v.Forms = append([]Form(nil), v.Forms...)
	return v
}
func cloneDiscoItems(v DiscoItems) DiscoItems {
	v.Items = append([]DiscoItem(nil), v.Items...)
	if v.RSM != nil {
		r := *v.RSM
		v.RSM = &r
	}
	return v
}
func asIQ(s xmpp.Stanza) xmpp.IQ {
	switch v := s.(type) {
	case xmpp.IQ:
		return v
	case *xmpp.IQ:
		return *v
	}
	panic("xep: stanza is not IQ")
}
func init() { registerSpecialized(30, func() xmpp.Plugin { return NewDisco() }) }
