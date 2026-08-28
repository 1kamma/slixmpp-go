package xmpp

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Plugin is an XMPP extension module.
type Plugin interface {
	Name() string
	Description() string
	Dependencies() []string
	Features() []string
	Init(*Client) error
	Shutdown(context.Context) error
}

// PluginFactory constructs a plugin.
type PluginFactory func() Plugin

// BasicPlugin supplies metadata and no-op lifecycle methods.
type BasicPlugin struct {
	PluginName, PluginDescription      string
	PluginDependencies, PluginFeatures []string
	OnInit                             func(*Client) error
	OnShutdown                         func(context.Context) error
}

func (p *BasicPlugin) Name() string           { return p.PluginName }
func (p *BasicPlugin) Description() string    { return p.PluginDescription }
func (p *BasicPlugin) Dependencies() []string { return append([]string(nil), p.PluginDependencies...) }
func (p *BasicPlugin) Features() []string     { return append([]string(nil), p.PluginFeatures...) }
func (p *BasicPlugin) Init(c *Client) error {
	if p.OnInit != nil {
		return p.OnInit(c)
	}
	return nil
}
func (p *BasicPlugin) Shutdown(ctx context.Context) error {
	if p.OnShutdown != nil {
		return p.OnShutdown(ctx)
	}
	return nil
}

// PluginManager owns plugin factories and loaded instances.
type PluginManager struct {
	mu        sync.RWMutex
	client    *Client
	factories map[string]PluginFactory
	loaded    map[string]Plugin
	loading   map[string]bool
	order     []string
}

func newPluginManager(c *Client) *PluginManager {
	return &PluginManager{client: c, factories: make(map[string]PluginFactory), loaded: make(map[string]Plugin), loading: make(map[string]bool)}
}
func (m *PluginManager) RegisterFactory(name string, f PluginFactory) error {
	if name == "" || f == nil {
		return fmt.Errorf("xmpp: plugin name and factory are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.factories[name]; ok {
		return fmt.Errorf("xmpp: plugin factory %q already registered", name)
	}
	m.factories[name] = f
	return nil
}
func (m *PluginManager) Use(p Plugin) error {
	if p == nil || p.Name() == "" {
		return fmt.Errorf("xmpp: invalid plugin")
	}
	m.mu.Lock()
	if _, ok := m.loaded[p.Name()]; ok {
		m.mu.Unlock()
		return nil
	}
	if m.loading[p.Name()] {
		m.mu.Unlock()
		return fmt.Errorf("xmpp: cyclic plugin dependency involving %q", p.Name())
	}
	m.loading[p.Name()] = true
	m.mu.Unlock()
	defer func() { m.mu.Lock(); delete(m.loading, p.Name()); m.mu.Unlock() }()
	for _, dep := range p.Dependencies() {
		if _, err := m.Load(dep); err != nil {
			return fmt.Errorf("xmpp: initialize %s dependency %s: %w", p.Name(), dep, err)
		}
	}
	if err := p.Init(m.client); err != nil {
		return fmt.Errorf("xmpp: initialize plugin %s: %w", p.Name(), err)
	}
	m.mu.Lock()
	m.loaded[p.Name()] = p
	m.order = append(m.order, p.Name())
	m.mu.Unlock()
	_ = m.client.Events.Emit(context.Background(), "plugin_loaded", p)
	return nil
}
func (m *PluginManager) Load(name string) (Plugin, error) {
	m.mu.RLock()
	if p, ok := m.loaded[name]; ok {
		m.mu.RUnlock()
		return p, nil
	}
	f := m.factories[name]
	m.mu.RUnlock()
	if f == nil {
		return nil, fmt.Errorf("xmpp: unknown plugin %q", name)
	}
	p := f()
	if err := m.Use(p); err != nil {
		return nil, err
	}
	return p, nil
}
func (m *PluginManager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.loaded[name]
	return p, ok
}
func (m *PluginManager) MustGet(name string) Plugin {
	p, ok := m.Get(name)
	if !ok {
		panic("xmpp: plugin not loaded: " + name)
	}
	return p
}
func (m *PluginManager) Names() []string {
	m.mu.RLock()
	names := make([]string, 0, len(m.loaded))
	for n := range m.loaded {
		names = append(names, n)
	}
	m.mu.RUnlock()
	sort.Strings(names)
	return names
}
func (m *PluginManager) Features() []string {
	seen := map[string]struct{}{}
	m.mu.RLock()
	for _, p := range m.loaded {
		for _, f := range p.Features() {
			if f != "" {
				seen[f] = struct{}{}
			}
		}
	}
	m.mu.RUnlock()
	out := make([]string, 0, len(seen))
	for f := range seen {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}
func (m *PluginManager) shutdown(ctx context.Context) error {
	m.mu.RLock()
	order := append([]string(nil), m.order...)
	loaded := make(map[string]Plugin, len(m.loaded))
	for k, v := range m.loaded {
		loaded[k] = v
	}
	m.mu.RUnlock()
	var errs []error
	for i := len(order) - 1; i >= 0; i-- {
		if err := loaded[order[i]].Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return joinErrors(errs)
}
