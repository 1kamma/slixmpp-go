package xep

import (
	"context"
	"fmt"
	"github.com/saret/slixmpp-go/xmpp"
)

// GenericPlugin preserves a Slixmpp plugin identity without claiming wire support.
type GenericPlugin struct{ Descriptor Descriptor }

func (p *GenericPlugin) Name() string                   { return p.Descriptor.Name() }
func (p *GenericPlugin) Description() string            { return p.Descriptor.XEP() + ": " + p.Descriptor.Title }
func (p *GenericPlugin) Dependencies() []string         { return nil }
func (p *GenericPlugin) Features() []string             { return nil }
func (p *GenericPlugin) Init(*xmpp.Client) error        { return nil }
func (p *GenericPlugin) Shutdown(context.Context) error { return nil }

var specializedFactories = map[int]xmpp.PluginFactory{}

func registerSpecialized(number int, f xmpp.PluginFactory) {
	if _, ok := specializedFactories[number]; ok {
		panic(fmt.Sprintf("xep: duplicate factory for XEP-%04d", number))
	}
	specializedFactories[number] = f
}

// RegisterAll registers every Slixmpp 1.17 xep_NNNN plugin name.
func RegisterAll(client *xmpp.Client) error {
	if client == nil {
		return fmt.Errorf("xep: nil client")
	}
	for _, descriptor := range Catalog {
		d := descriptor
		f := specializedFactories[d.Number]
		if f == nil {
			f = func() xmpp.Plugin { return &GenericPlugin{Descriptor: d} }
		}
		if err := client.Plugins.RegisterFactory(d.Name(), f); err != nil {
			return err
		}
	}
	return nil
}

// Load loads one XEP plugin by number.
func Load(client *xmpp.Client, number int) (xmpp.Plugin, error) {
	d, ok := Lookup(number)
	if !ok {
		return nil, fmt.Errorf("xep: XEP-%04d is not in the Slixmpp catalog", number)
	}
	return client.Plugins.Load(d.Name())
}

// LoadDefaults loads a practical baseline.
func LoadDefaults(client *xmpp.Client) error {
	for _, n := range []int{4, 30, 92, 184, 199, 202, 203, 334, 359, 380} {
		if _, err := Load(client, n); err != nil {
			return err
		}
	}
	return nil
}
