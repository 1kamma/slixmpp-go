package xmpp

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

// Logger is the logging subset used by the library.
type Logger interface{ Printf(string, ...any) }

// Config controls a client or XEP-0114 component connection.
type Config struct {
	JID, Password, Resource string
	// ComponentSecret switches to XEP-0114 mode. JID must be a domain JID and Address is required.
	ComponentSecret                               string
	Address                                       string
	DirectTLS, RequireTLS, InsecureAllowPlainAuth bool
	TLSConfig                                     *tls.Config
	SASLMechanisms                                []string
	AuthorizationID                               string
	Resolver                                      *net.Resolver
	Dialer                                        *net.Dialer
	ConnectTimeout, IQTimeout, KeepAlive          time.Duration
	AutoPresence, DisableAutoPresence             bool
	InitialPresence                               Presence
	Language, UserAgent                           string
	Debug, DebugXML                               bool
	Logger                                        Logger
}

// DefaultConfig returns secure client defaults.
func DefaultConfig(jid, password string) Config {
	return Config{JID: jid, Password: password, RequireTLS: true, AutoPresence: true, ConnectTimeout: 20 * time.Second, IQTimeout: 30 * time.Second, Language: "en", UserAgent: "slixmpp-go"}
}
func (c Config) normalized() Config {
	if c.ConnectTimeout <= 0 {
		c.ConnectTimeout = 20 * time.Second
	}
	if c.IQTimeout <= 0 {
		c.IQTimeout = 30 * time.Second
	}
	if c.Language == "" {
		c.Language = "en"
	}
	if c.UserAgent == "" {
		c.UserAgent = "slixmpp-go"
	}
	if c.Resolver == nil {
		c.Resolver = net.DefaultResolver
	}
	if c.Dialer == nil {
		c.Dialer = &net.Dialer{Timeout: c.ConnectTimeout, KeepAlive: 30 * time.Second}
	}
	if len(c.SASLMechanisms) == 0 {
		c.SASLMechanisms = []string{"SCRAM-SHA-256", "SCRAM-SHA-1", "EXTERNAL", "PLAIN", "ANONYMOUS"}
	}
	if c.ComponentSecret != "" || c.DisableAutoPresence {
		c.AutoPresence = false
	} else {
		c.AutoPresence = true
	}
	if !c.RequireTLS && !c.InsecureAllowPlainAuth && !c.DirectTLS && c.ComponentSecret == "" {
		c.RequireTLS = true
	}
	return c
}
func (c Config) validate() (JID, error) {
	jid, err := ParseJID(c.JID)
	if err != nil {
		return JID{}, err
	}
	if c.Resource != "" {
		jid.Resource = c.Resource
	}
	if c.ComponentSecret != "" {
		if !jid.IsDomain() {
			return JID{}, fmt.Errorf("xmpp: component JID must be a domain JID")
		}
		if c.Address == "" {
			return JID{}, fmt.Errorf("xmpp: component Address is required")
		}
	}
	if c.Address != "" {
		if _, _, err := net.SplitHostPort(c.Address); err != nil {
			return JID{}, fmt.Errorf("xmpp: invalid address %q: %w", c.Address, err)
		}
	}
	return jid, nil
}
