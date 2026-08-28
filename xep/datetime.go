package xep

import (
	"fmt"
	"github.com/saret/slixmpp-go/xmpp"
	"time"
)

const DelayNS = "urn:xmpp:delay"

// FormatDateTime formats an XEP-0082 date-time in UTC.
func FormatDateTime(value time.Time) string { return value.UTC().Format("2006-01-02T15:04:05.000Z") }

// ParseDateTime parses the date-time forms accepted by XEP-0082.
func ParseDateTime(value string) (time.Time, error) {
	layouts := []string{time.RFC3339Nano, "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05Z", "2006-01-02"}
	var last error
	for _, layout := range layouts {
		v, err := time.Parse(layout, value)
		if err == nil {
			return v, nil
		}
		last = err
	}
	return time.Time{}, fmt.Errorf("xep: invalid XMPP date/time %q: %w", value, last)
}

// Delay is an XEP-0203 delayed-delivery marker.
type Delay struct {
	From, Reason string
	Stamp        time.Time
}

func (d Delay) ToNode() xmpp.Node {
	n := xmpp.NewNode(DelayNS, "delay")
	if d.From != "" {
		n.SetAttr("from", d.From)
	}
	if !d.Stamp.IsZero() {
		n.SetAttr("stamp", FormatDateTime(d.Stamp))
	}
	n.AddText(d.Reason)
	return n
}
func ParseDelay(n xmpp.Node) (Delay, error) {
	if n.Name.Space != DelayNS || n.Name.Local != "delay" {
		return Delay{}, fmt.Errorf("xep: expected {%s}delay", DelayNS)
	}
	d := Delay{Reason: n.Text()}
	d.From, _ = n.AttrValue("from")
	if stamp, ok := n.AttrValue("stamp"); ok {
		v, err := ParseDateTime(stamp)
		if err != nil {
			return Delay{}, err
		}
		d.Stamp = v
	}
	return d, nil
}
func MessageDelay(message xmpp.Message) (Delay, bool, error) {
	node := message.Extension(DelayNS, "delay")
	if node == nil {
		return Delay{}, false, nil
	}
	v, err := ParseDelay(*node)
	return v, true, err
}

type delayPlugin struct{ xmpp.BasicPlugin }

func newDelayPlugin() xmpp.Plugin {
	return &delayPlugin{BasicPlugin: xmpp.BasicPlugin{PluginName: "xep_0203", PluginDescription: "XEP-0203 Delayed Delivery", PluginFeatures: []string{DelayNS}}}
}
func init() {
	registerSpecialized(82, staticPlugin(82))
	registerSpecialized(203, newDelayPlugin)
}
