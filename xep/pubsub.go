package xep

import (
	"context"
	"fmt"
	"github.com/saret/slixmpp-go/xmpp"
)

const (
	PubSubNS       = "http://jabber.org/protocol/pubsub"
	PubSubEventNS  = "http://jabber.org/protocol/pubsub#event"
	PubSubOwnerNS  = "http://jabber.org/protocol/pubsub#owner"
	PubSubErrorsNS = "http://jabber.org/protocol/pubsub#errors"
)

type PubSubItem struct {
	ID, Publisher string
	Payloads      []xmpp.Node
}
type PubSubSubscription struct{ JID, Node, SubID, State string }
type PubSubAffiliation struct{ JID, Node, Affiliation string }
type PubSubEvent struct {
	From, Node    string
	Items         []PubSubItem
	Retracted     []string
	Purge, Delete bool
	Redirect      string
	Subscription  *PubSubSubscription
	Configuration *Form
}

type PubSub struct {
	client  *xmpp.Client
	handler *xmpp.Handler
}

func NewPubSub() *PubSub                 { return &PubSub{} }
func (p *PubSub) Name() string           { return "xep_0060" }
func (p *PubSub) Description() string    { return "XEP-0060 Publish-Subscribe" }
func (p *PubSub) Dependencies() []string { return []string{"xep_0004", "xep_0030", "xep_0059"} }
func (p *PubSub) Features() []string     { return []string{PubSubNS, PubSubEventNS, PubSubOwnerNS} }
func (p *PubSub) Init(c *xmpp.Client) error {
	p.client = c
	p.handler = c.AddHandler("pubsub-event", xmpp.MatchAnd(xmpp.MatchKind("message"), xmpp.MatchPayload(PubSubEventNS, "event")), p.handleEvent)
	return nil
}
func (p *PubSub) Shutdown(context.Context) error {
	if p.client != nil {
		p.client.RemoveHandler(p.handler)
	}
	return nil
}
func (p *PubSub) GetItems(ctx context.Context, service, node string, max int, itemIDs ...string) ([]PubSubItem, *RSM, error) {
	pubsub := xmpp.NewNode(PubSubNS, "pubsub")
	items := xmpp.NewNode(PubSubNS, "items")
	items.SetAttr("node", node)
	if max > 0 {
		items.SetAttr("max_items", fmt.Sprint(max))
	}
	for _, id := range itemIDs {
		x := xmpp.NewNode(PubSubNS, "item")
		x.SetAttr("id", id)
		items.AddChild(x)
	}
	pubsub.AddChild(items)
	response, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQGet, Payloads: []xmpp.Node{pubsub}})
	if err != nil {
		return nil, nil, err
	}
	payload := response.Payload()
	if payload == nil {
		return nil, nil, fmt.Errorf("xep: missing pubsub response")
	}
	itemsNode := payload.Child(PubSubNS, "items")
	if itemsNode == nil {
		return nil, nil, fmt.Errorf("xep: missing pubsub items")
	}
	out := parsePubSubItems(*itemsNode)
	var rsm *RSM
	if set := itemsNode.Child(RSMNS, "set"); set != nil {
		v, e := ParseRSM(*set)
		if e != nil {
			return nil, nil, e
		}
		rsm = &v
	}
	return out, rsm, nil
}
func (p *PubSub) Publish(ctx context.Context, service, node, id string, payloads []xmpp.Node, options *Form) (string, error) {
	pubsub := xmpp.NewNode(PubSubNS, "pubsub")
	publish := xmpp.NewNode(PubSubNS, "publish")
	publish.SetAttr("node", node)
	item := xmpp.NewNode(PubSubNS, "item")
	if id != "" {
		item.SetAttr("id", id)
	}
	for _, payload := range payloads {
		item.AddChild(payload)
	}
	publish.AddChild(item)
	pubsub.AddChild(publish)
	if options != nil {
		form := *options
		form.Type = FormTypeSubmit
		publishOptions := xmpp.NewNode(PubSubNS, "publish-options")
		publishOptions.AddChild(form.ToNode())
		pubsub.AddChild(publishOptions)
	}
	response, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQSet, Payloads: []xmpp.Node{pubsub}})
	if err != nil {
		return "", err
	}
	payload := response.Payload()
	if payload != nil {
		if publishResult := payload.Child(PubSubNS, "publish"); publishResult != nil {
			if resultItem := publishResult.Child(PubSubNS, "item"); resultItem != nil {
				if value, ok := resultItem.AttrValue("id"); ok {
					return value, nil
				}
			}
		}
	}
	return id, nil
}
func (p *PubSub) Retract(ctx context.Context, service, node string, notify bool, ids ...string) error {
	pubsub := xmpp.NewNode(PubSubNS, "pubsub")
	retract := xmpp.NewNode(PubSubNS, "retract")
	retract.SetAttr("node", node)
	if notify {
		retract.SetAttr("notify", "true")
	}
	for _, id := range ids {
		item := xmpp.NewNode(PubSubNS, "item")
		item.SetAttr("id", id)
		retract.AddChild(item)
	}
	pubsub.AddChild(retract)
	_, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQSet, Payloads: []xmpp.Node{pubsub}})
	return err
}
func (p *PubSub) Subscribe(ctx context.Context, service, node, jid string, options *Form) (PubSubSubscription, error) {
	pubsub := xmpp.NewNode(PubSubNS, "pubsub")
	subscribe := xmpp.NewNode(PubSubNS, "subscribe")
	subscribe.SetAttr("node", node)
	subscribe.SetAttr("jid", jid)
	pubsub.AddChild(subscribe)
	if options != nil {
		form := *options
		form.Type = FormTypeSubmit
		o := xmpp.NewNode(PubSubNS, "options")
		o.SetAttr("node", node)
		o.SetAttr("jid", jid)
		o.AddChild(form.ToNode())
		pubsub.AddChild(o)
	}
	response, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQSet, Payloads: []xmpp.Node{pubsub}})
	if err != nil {
		return PubSubSubscription{}, err
	}
	payload := response.Payload()
	if payload == nil {
		return PubSubSubscription{}, fmt.Errorf("xep: missing pubsub subscription")
	}
	n := payload.Child(PubSubNS, "subscription")
	if n == nil {
		return PubSubSubscription{}, fmt.Errorf("xep: missing subscription element")
	}
	return parseSubscription(*n), nil
}
func (p *PubSub) Unsubscribe(ctx context.Context, service, node, jid, subID string) error {
	pubsub := xmpp.NewNode(PubSubNS, "pubsub")
	n := xmpp.NewNode(PubSubNS, "unsubscribe")
	n.SetAttr("node", node)
	n.SetAttr("jid", jid)
	if subID != "" {
		n.SetAttr("subid", subID)
	}
	pubsub.AddChild(n)
	_, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQSet, Payloads: []xmpp.Node{pubsub}})
	return err
}
func (p *PubSub) Subscriptions(ctx context.Context, service, node string) ([]PubSubSubscription, error) {
	pubsub := xmpp.NewNode(PubSubNS, "pubsub")
	n := xmpp.NewNode(PubSubNS, "subscriptions")
	if node != "" {
		n.SetAttr("node", node)
	}
	pubsub.AddChild(n)
	response, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQGet, Payloads: []xmpp.Node{pubsub}})
	if err != nil {
		return nil, err
	}
	payload := response.Payload()
	if payload == nil {
		return nil, fmt.Errorf("xep: missing pubsub response")
	}
	parent := payload.Child(PubSubNS, "subscriptions")
	if parent == nil {
		return nil, nil
	}
	var out []PubSubSubscription
	for _, child := range parent.Children() {
		if child.Name.Local == "subscription" {
			out = append(out, parseSubscription(child))
		}
	}
	return out, nil
}
func (p *PubSub) Affiliations(ctx context.Context, service, node string) ([]PubSubAffiliation, error) {
	pubsub := xmpp.NewNode(PubSubNS, "pubsub")
	n := xmpp.NewNode(PubSubNS, "affiliations")
	if node != "" {
		n.SetAttr("node", node)
	}
	pubsub.AddChild(n)
	response, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQGet, Payloads: []xmpp.Node{pubsub}})
	if err != nil {
		return nil, err
	}
	payload := response.Payload()
	if payload == nil {
		return nil, fmt.Errorf("xep: missing pubsub response")
	}
	parent := payload.Child(PubSubNS, "affiliations")
	if parent == nil {
		return nil, nil
	}
	var out []PubSubAffiliation
	for _, child := range parent.Children() {
		if child.Name.Local == "affiliation" {
			v := PubSubAffiliation{}
			v.JID, _ = child.AttrValue("jid")
			v.Node, _ = child.AttrValue("node")
			v.Affiliation, _ = child.AttrValue("affiliation")
			out = append(out, v)
		}
	}
	return out, nil
}
func (p *PubSub) CreateNode(ctx context.Context, service, node string, config *Form) error {
	pubsub := xmpp.NewNode(PubSubNS, "pubsub")
	create := xmpp.NewNode(PubSubNS, "create")
	if node != "" {
		create.SetAttr("node", node)
	}
	pubsub.AddChild(create)
	if config != nil {
		form := *config
		form.Type = FormTypeSubmit
		configure := xmpp.NewNode(PubSubNS, "configure")
		configure.AddChild(form.ToNode())
		pubsub.AddChild(configure)
	}
	_, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQSet, Payloads: []xmpp.Node{pubsub}})
	return err
}
func (p *PubSub) DeleteNode(ctx context.Context, service, node, redirect string) error {
	pubsub := xmpp.NewNode(PubSubOwnerNS, "pubsub")
	deleteNode := xmpp.NewNode(PubSubOwnerNS, "delete")
	deleteNode.SetAttr("node", node)
	if redirect != "" {
		r := xmpp.NewNode(PubSubOwnerNS, "redirect")
		r.SetAttr("uri", redirect)
		deleteNode.AddChild(r)
	}
	pubsub.AddChild(deleteNode)
	_, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQSet, Payloads: []xmpp.Node{pubsub}})
	return err
}
func (p *PubSub) Purge(ctx context.Context, service, node string) error {
	pubsub := xmpp.NewNode(PubSubOwnerNS, "pubsub")
	purge := xmpp.NewNode(PubSubOwnerNS, "purge")
	purge.SetAttr("node", node)
	pubsub.AddChild(purge)
	_, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQSet, Payloads: []xmpp.Node{pubsub}})
	return err
}
func (p *PubSub) GetNodeConfig(ctx context.Context, service, node string) (Form, error) {
	pubsub := xmpp.NewNode(PubSubOwnerNS, "pubsub")
	configure := xmpp.NewNode(PubSubOwnerNS, "configure")
	configure.SetAttr("node", node)
	pubsub.AddChild(configure)
	response, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQGet, Payloads: []xmpp.Node{pubsub}})
	if err != nil {
		return Form{}, err
	}
	payload := response.Payload()
	if payload == nil {
		return Form{}, fmt.Errorf("xep: missing pubsub owner response")
	}
	parent := payload.Child(PubSubOwnerNS, "configure")
	if parent == nil {
		return Form{}, fmt.Errorf("xep: missing configure response")
	}
	form := parent.Child(DataFormsNS, "x")
	if form == nil {
		return Form{}, fmt.Errorf("xep: missing configuration form")
	}
	return ParseForm(*form)
}
func (p *PubSub) SetNodeConfig(ctx context.Context, service, node string, form Form) error {
	form.Type = FormTypeSubmit
	pubsub := xmpp.NewNode(PubSubOwnerNS, "pubsub")
	configure := xmpp.NewNode(PubSubOwnerNS, "configure")
	configure.SetAttr("node", node)
	configure.AddChild(form.ToNode())
	pubsub.AddChild(configure)
	_, err := p.client.RequestIQ(ctx, xmpp.IQ{To: service, Type: xmpp.IQSet, Payloads: []xmpp.Node{pubsub}})
	return err
}
func (p *PubSub) handleEvent(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
	message := asMessage(s)
	eventNode := message.Extension(PubSubEventNS, "event")
	event, err := parsePubSubEvent(*eventNode)
	if err != nil {
		return err
	}
	event.From = message.From
	_ = c.Events.Emit(ctx, "pubsub_event", event)
	if event.Node != "" {
		_ = c.Events.Emit(ctx, "pubsub_event:"+event.Node, event)
	}
	return nil
}
func parsePubSubEvent(n xmpp.Node) (PubSubEvent, error) {
	if n.Name.Space != PubSubEventNS || n.Name.Local != "event" {
		return PubSubEvent{}, fmt.Errorf("xep: invalid pubsub event")
	}
	var event PubSubEvent
	for _, child := range n.Children() {
		switch child.Name.Local {
		case "items":
			event.Node, _ = child.AttrValue("node")
			event.Items = parsePubSubItems(child)
			for _, x := range child.Children() {
				if x.Name.Local == "retract" {
					id, _ := x.AttrValue("id")
					event.Retracted = append(event.Retracted, id)
				}
			}
		case "purge":
			event.Node, _ = child.AttrValue("node")
			event.Purge = true
		case "delete":
			event.Node, _ = child.AttrValue("node")
			event.Delete = true
			if r := child.Child(PubSubEventNS, "redirect"); r != nil {
				event.Redirect, _ = r.AttrValue("uri")
			}
		case "subscription":
			v := parseSubscription(child)
			event.Subscription = &v
			event.Node = v.Node
		case "configuration":
			event.Node, _ = child.AttrValue("node")
			if f := child.Child(DataFormsNS, "x"); f != nil {
				v, e := ParseForm(*f)
				if e != nil {
					return event, e
				}
				event.Configuration = &v
			}
		}
	}
	return event, nil
}
func parsePubSubItems(n xmpp.Node) []PubSubItem {
	var out []PubSubItem
	for _, child := range n.Children() {
		if child.Name.Local != "item" {
			continue
		}
		item := PubSubItem{}
		item.ID, _ = child.AttrValue("id")
		item.Publisher, _ = child.AttrValue("publisher")
		item.Payloads = child.Children()
		out = append(out, item)
	}
	return out
}
func parseSubscription(n xmpp.Node) PubSubSubscription {
	v := PubSubSubscription{}
	v.JID, _ = n.AttrValue("jid")
	v.Node, _ = n.AttrValue("node")
	v.SubID, _ = n.AttrValue("subid")
	v.State, _ = n.AttrValue("subscription")
	return v
}

// PEP exposes PubSub operations against the user's own PEP service.
type PEP struct {
	client *xmpp.Client
	PubSub *PubSub
}

func NewPEP() *PEP                    { return &PEP{} }
func (p *PEP) Name() string           { return "xep_0163" }
func (p *PEP) Description() string    { return "XEP-0163 Personal Eventing Protocol" }
func (p *PEP) Dependencies() []string { return []string{"xep_0060"} }
func (p *PEP) Features() []string     { return nil }
func (p *PEP) Init(c *xmpp.Client) error {
	p.client = c
	plugin, ok := c.Plugins.Get("xep_0060")
	if !ok {
		return fmt.Errorf("xep: xep_0060 not loaded")
	}
	var cast bool
	p.PubSub, cast = plugin.(*PubSub)
	if !cast {
		return fmt.Errorf("xep: xep_0060 has unexpected type")
	}
	return nil
}
func (p *PEP) Shutdown(context.Context) error { return nil }
func (p *PEP) Publish(ctx context.Context, node, id string, payload xmpp.Node, options *Form) (string, error) {
	return p.PubSub.Publish(ctx, "", node, id, []xmpp.Node{payload}, options)
}
func (p *PEP) Items(ctx context.Context, jid, node string, max int) ([]PubSubItem, error) {
	items, _, err := p.PubSub.GetItems(ctx, jid, node, max)
	return items, err
}
func init() {
	registerSpecialized(60, func() xmpp.Plugin { return NewPubSub() })
	registerSpecialized(163, func() xmpp.Plugin { return NewPEP() })
}
