package xep

import (
	"fmt"
	"github.com/1kamma/slixmpp-go/xmpp"
	"strconv"
)

const RSMNS = "http://jabber.org/protocol/rsm"

type RSM struct {
	After, Before, First, Last string
	BeforeSet                  bool
	FirstIndex                 *int
	Max, Count, Index          *int
}

func (n RSM) ToNode() xmpp.Node {
	set := xmpp.NewNode(RSMNS, "set")
	if n.Max != nil {
		set.AddChild(xmpp.NewTextNode(RSMNS, "max", strconv.Itoa(*n.Max)))
	}
	if n.After != "" {
		set.AddChild(xmpp.NewTextNode(RSMNS, "after", n.After))
	}
	if n.BeforeSet {
		set.AddChild(xmpp.NewTextNode(RSMNS, "before", n.Before))
	}
	if n.Index != nil {
		set.AddChild(xmpp.NewTextNode(RSMNS, "index", strconv.Itoa(*n.Index)))
	}
	if n.First != "" || n.FirstIndex != nil {
		f := xmpp.NewTextNode(RSMNS, "first", n.First)
		if n.FirstIndex != nil {
			f.SetAttr("index", strconv.Itoa(*n.FirstIndex))
		}
		set.AddChild(f)
	}
	if n.Last != "" {
		set.AddChild(xmpp.NewTextNode(RSMNS, "last", n.Last))
	}
	if n.Count != nil {
		set.AddChild(xmpp.NewTextNode(RSMNS, "count", strconv.Itoa(*n.Count)))
	}
	return set
}
func ParseRSM(n xmpp.Node) (RSM, error) {
	if n.Name.Local != "set" || n.Name.Space != RSMNS {
		return RSM{}, fmt.Errorf("xep: expected {%s}set", RSMNS)
	}
	var r RSM
	for _, c := range n.Children() {
		parseInt := func() (*int, error) {
			v, err := strconv.Atoi(c.Text())
			if err != nil {
				return nil, err
			}
			return &v, nil
		}
		switch c.Name.Local {
		case "after":
			r.After = c.Text()
		case "before":
			r.BeforeSet = true
			r.Before = c.Text()
		case "first":
			r.First = c.Text()
			if x, ok := c.AttrValue("index"); ok {
				v, err := strconv.Atoi(x)
				if err != nil {
					return RSM{}, err
				}
				r.FirstIndex = &v
			}
		case "last":
			r.Last = c.Text()
		case "max":
			v, e := parseInt()
			if e != nil {
				return RSM{}, e
			}
			r.Max = v
		case "count":
			v, e := parseInt()
			if e != nil {
				return RSM{}, e
			}
			r.Count = v
		case "index":
			v, e := parseInt()
			if e != nil {
				return RSM{}, e
			}
			r.Index = v
		}
	}
	return r, nil
}
func Int(value int) *int { return &value }

type rsmPlugin struct{ xmpp.BasicPlugin }

func newRSMPlugin() xmpp.Plugin {
	return &rsmPlugin{BasicPlugin: xmpp.BasicPlugin{PluginName: "xep_0059", PluginDescription: "XEP-0059 Result Set Management", PluginFeatures: []string{RSMNS}}}
}
func init() { registerSpecialized(59, newRSMPlugin) }
