package xmpp

import (
	"context"
	"crypto/sha1" // #nosec G505 -- XEP-0114 mandates SHA-1.
	"crypto/tls"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"time"
)

const componentNS = "jabber:component:accept"

// IsComponent reports whether this client uses XEP-0114 authentication.
func (c *Client) IsComponent() bool { return c.cfg.ComponentSecret != "" }
func (c *Client) connectComponent(parent context.Context) error {
	c.stateMu.Lock()
	if c.connected || c.conn != nil {
		c.stateMu.Unlock()
		return fmt.Errorf("xmpp: client is already connected")
	}
	c.stateMu.Unlock()
	ctx := parent
	var cancel context.CancelFunc
	if _, ok := parent.Deadline(); !ok && c.cfg.ConnectTimeout > 0 {
		ctx, cancel = context.WithTimeout(parent, c.cfg.ConnectTimeout)
		defer cancel()
	}
	conn, err := c.cfg.Dialer.DialContext(ctx, "tcp", c.cfg.Address)
	if err != nil {
		return fmt.Errorf("xmpp: connect component to %s: %w", c.cfg.Address, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if c.cfg.DirectTLS {
		tlsConn := tls.Client(conn, c.tlsConfig())
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return fmt.Errorf("xmpp: component TLS handshake: %w", err)
		}
		conn = tlsConn
	}
	c.setConn(conn)
	opening := "<stream:stream xmlns='" + componentNS + "' xmlns:stream='" + streamNS + "' to='" + escapeXMLAttribute(c.JID().Domain) + "'>"
	if err := c.writeRaw([]byte(opening), false); err != nil {
		return c.failConnect(conn, err)
	}
	decoder := c.currentDecoder()
	var streamID string
	for {
		token, err := decoder.Token()
		if err != nil {
			return c.failConnect(conn, fmt.Errorf("xmpp: read component stream opening: %w", err))
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "stream" || start.Name.Space != streamNS {
			return c.failConnect(conn, fmt.Errorf("xmpp: expected component stream opening, received <%s>", start.Name.Local))
		}
		for _, a := range start.Attr {
			if a.Name.Local == "id" {
				streamID = a.Value
			}
		}
		break
	}
	if streamID == "" {
		return c.failConnect(conn, fmt.Errorf("xmpp: component stream did not include an id"))
	}
	digest := sha1.Sum([]byte(streamID + c.cfg.ComponentSecret))
	handshake := NewTextNode(componentNS, "handshake", hex.EncodeToString(digest[:]))
	if err := c.writeNode(handshake); err != nil {
		return c.failConnect(conn, err)
	}
	response, err := c.readTopLevelNode()
	if err != nil {
		return c.failConnect(conn, err)
	}
	if response.Name.Local != "handshake" || response.Name.Space != componentNS {
		return c.failConnect(conn, fmt.Errorf("xmpp: component authentication rejected with <%s>", response.Name.Local))
	}
	_ = conn.SetDeadline(time.Time{})
	return c.finishConnect()
}
