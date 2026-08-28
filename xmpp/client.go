package xmpp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	clientNS        = "jabber:client"
	streamNS        = "http://etherx.jabber.org/streams"
	startTLSNS      = "urn:ietf:params:xml:ns:xmpp-tls"
	bindNS          = "urn:ietf:params:xml:ns:xmpp-bind"
	legacySessionNS = "urn:ietf:params:xml:ns:xmpp-session"
)

// Client is an RFC 6120/6121 client or XEP-0114 external component.
type Client struct {
	cfg                Config
	stateMu            sync.RWMutex
	jid                JID
	conn               net.Conn
	decoder            *xml.Decoder
	connected, closing bool
	lastFeatures       Node
	writeMu            sync.Mutex
	pendingMu          sync.Mutex
	pending            map[string]chan IQ
	nextID             atomic.Uint64
	handlers           handlerRegistry
	Events             *EventBus
	API                *APIRegistry
	Plugins            *PluginManager
	ctx                context.Context
	cancel             context.CancelFunc
	done               chan struct{}
	doneOnce           sync.Once
	keepaliveDone      chan struct{}
	OnMessage          func(Message)
	OnPresence         func(Presence)
	OnIQ               func(IQ)
	OnError            func(error)
}

// NewClient validates config and constructs a disconnected client.
func NewClient(config Config) (*Client, error) {
	config = config.normalized()
	jid, err := config.validate()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{cfg: config, jid: jid, pending: make(map[string]chan IQ), Events: NewEventBus(), API: NewAPIRegistry(), ctx: ctx, cancel: cancel, done: make(chan struct{}), keepaliveDone: make(chan struct{})}
	c.Plugins = newPluginManager(c)
	return c, nil
}

// MustNewClient panics if config is invalid.
func MustNewClient(config Config) *Client {
	c, err := NewClient(config)
	if err != nil {
		panic(err)
	}
	return c
}
func (c *Client) JID() JID        { c.stateMu.RLock(); defer c.stateMu.RUnlock(); return c.jid }
func (c *Client) BoundJID() JID   { return c.JID() }
func (c *Client) Connected() bool { c.stateMu.RLock(); defer c.stateMu.RUnlock(); return c.connected }
func (c *Client) Features() Node {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.lastFeatures.Clone()
}
func (c *Client) NextID() string     { return "go-" + strconv.FormatUint(c.nextID.Add(1), 36) }
func (c *Client) Use(p Plugin) error { return c.Plugins.Use(p) }
func (c *Client) AddHandler(name string, m Matcher, cb StanzaHandler, options ...HandlerOptions) *Handler {
	if cb == nil {
		panic("xmpp: nil stanza handler")
	}
	var o HandlerOptions
	if len(options) > 0 {
		o = options[0]
	}
	return c.handlers.add(name, m, cb, o)
}
func (c *Client) RemoveHandler(h *Handler) { c.handlers.remove(h) }

// Connect performs transport, TLS, SASL, resource binding, and initial presence.
func (c *Client) Connect(parent context.Context) error {
	if parent == nil {
		parent = context.Background()
	}
	if c.cfg.ComponentSecret != "" {
		return c.connectComponent(parent)
	}
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
	address, err := c.resolveAddress(ctx)
	if err != nil {
		return err
	}
	c.debugf("connecting TCP to %s", address)
	conn, err := c.cfg.Dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("xmpp: connect to %s: %w", address, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	c.debugf("TCP connected: local=%s remote=%s", conn.LocalAddr(), conn.RemoteAddr())
	if c.cfg.DirectTLS {
		tlsConn := tls.Client(conn, c.tlsConfig())
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return fmt.Errorf("xmpp: direct TLS handshake: %w", err)
		}
		conn = tlsConn
		c.debugf("direct TLS established")
	}
	c.setConn(conn)
	features, err := c.openAndReadFeatures()
	if err != nil {
		return c.failConnect(conn, fmt.Errorf("xmpp: open stream: %w", err))
	}
	if !c.cfg.DirectTLS {
		if features.Child(startTLSNS, "starttls") != nil {
			c.debugf("server offers STARTTLS; negotiating")
			if err := c.startTLS(ctx); err != nil {
				return c.failConnect(conn, err)
			}
			features, err = c.openAndReadFeatures()
			if err != nil {
				return c.failConnect(c.currentConn(), fmt.Errorf("xmpp: reopen stream after TLS: %w", err))
			}
		} else if c.cfg.RequireTLS {
			return c.failConnect(conn, ErrTLSRequired)
		}
	}
	if err := c.authenticate(features); err != nil {
		return c.failConnect(c.currentConn(), err)
	}
	c.debugf("SASL authentication succeeded")
	features, err = c.openAndReadFeatures()
	if err != nil {
		return c.failConnect(c.currentConn(), fmt.Errorf("xmpp: reopen stream after authentication: %w", err))
	}
	if features.Child(bindNS, "bind") == nil {
		return c.failConnect(c.currentConn(), fmt.Errorf("xmpp: server did not advertise resource binding"))
	}
	if err := c.bindResource(ctx); err != nil {
		return c.failConnect(c.currentConn(), err)
	}
	if features.Child(legacySessionNS, "session") != nil {
		if err := c.startLegacySession(ctx); err != nil {
			return c.failConnect(c.currentConn(), err)
		}
	}
	return c.finishConnect()
}
func (c *Client) finishConnect() error {
	conn := c.currentConn()
	if conn == nil {
		return ErrNotConnected
	}
	_ = conn.SetDeadline(time.Time{})
	c.stateMu.Lock()
	c.connected = true
	c.stateMu.Unlock()
	go c.readLoop()
	if c.cfg.KeepAlive > 0 {
		go c.keepaliveLoop(c.cfg.KeepAlive)
	}
	_ = c.Events.Emit(c.ctx, "connected", c.JID())
	_ = c.Events.Emit(c.ctx, "session_start", c.JID())
	if c.cfg.AutoPresence {
		p := c.cfg.InitialPresence
		if err := c.SendPresence(p); err != nil {
			_ = c.Close()
			return fmt.Errorf("xmpp: send initial presence: %w", err)
		}
	}
	c.debugf("connected as %s", c.JID().String())
	return nil
}
func (c *Client) failConnect(conn net.Conn, err error) error {
	if conn != nil {
		_ = conn.Close()
	}
	c.clearConn()
	return err
}
func (c *Client) resolveAddress(ctx context.Context) (string, error) {
	if c.cfg.Address != "" {
		return c.cfg.Address, nil
	}
	jid := c.JID()
	service, port := "xmpp-client", "5222"
	if c.cfg.DirectTLS {
		service, port = "xmpps-client", "5223"
	}
	_, records, err := c.cfg.Resolver.LookupSRV(ctx, service, "tcp", jid.Domain)
	if err == nil {
		for _, record := range records {
			target := strings.TrimSuffix(record.Target, ".")
			if target != "" && target != "." {
				address := net.JoinHostPort(target, strconv.Itoa(int(record.Port)))
				c.debugf("resolved _%s._tcp.%s to %s", service, jid.Domain, address)
				return address, nil
			}
		}
	}
	return net.JoinHostPort(jid.Domain, port), nil
}

func (c *Client) openAndReadFeatures() (Node, error) {
	c.resetDecoder()
	jid := c.JID()
	opening := "<stream:stream to='" + escapeXMLAttribute(jid.Domain) + "' version='1.0' xmlns='" + clientNS + "' xmlns:stream='" + streamNS + "'"
	if c.cfg.Language != "" {
		opening += " xml:lang='" + escapeXMLAttribute(c.cfg.Language) + "'"
	}
	opening += ">"
	if err := c.writeRaw([]byte(opening), false); err != nil {
		return Node{}, err
	}
	c.debugf("opening XMPP stream")
	decoder := c.currentDecoder()
	for {
		token, err := decoder.Token()
		if err != nil {
			return Node{}, err
		}
		if start, ok := token.(xml.StartElement); ok {
			if start.Name.Local == "stream" && start.Name.Space == streamNS {
				break
			}
			return Node{}, fmt.Errorf("xmpp: expected stream opening, received <%s>", start.Name.Local)
		}
	}
	features, err := c.readTopLevelNode()
	if err != nil {
		return Node{}, err
	}
	if features.Name.Local != "features" || features.Name.Space != streamNS {
		return Node{}, fmt.Errorf("xmpp: expected stream features, received <%s>", features.Name.Local)
	}
	c.stateMu.Lock()
	c.lastFeatures = features.Clone()
	c.stateMu.Unlock()
	c.debugf("received stream features")
	return features, nil
}
func (c *Client) startTLS(ctx context.Context) error {
	if err := c.writeNode(NewNode(startTLSNS, "starttls")); err != nil {
		return err
	}
	response, err := c.readTopLevelNode()
	if err != nil {
		return err
	}
	if response.Name.Space != startTLSNS || response.Name.Local != "proceed" {
		return fmt.Errorf("xmpp: STARTTLS rejected with <%s>", response.Name.Local)
	}
	plain := c.currentConn()
	tlsConn := tls.Client(plain, c.tlsConfig())
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return fmt.Errorf("xmpp: STARTTLS handshake: %w", err)
	}
	c.setConn(tlsConn)
	c.debugf("STARTTLS established")
	return nil
}
func (c *Client) bindResource(ctx context.Context) error {
	bind := NewNode(bindNS, "bind")
	resource := c.JID().Resource
	if resource == "" {
		resource = "slixmpp-go"
	}
	bind.AddChild(NewTextNode(bindNS, "resource", resource))
	request := IQ{ID: c.NextID(), Type: IQSet, Payloads: []Node{bind}}
	c.debugf("binding resource %q", resource)
	response, err := c.syncIQ(ctx, request)
	if err != nil {
		return fmt.Errorf("xmpp: bind resource: %w", err)
	}
	payload := response.Payload()
	if payload == nil || payload.Name.Space != bindNS || payload.Name.Local != "bind" {
		return fmt.Errorf("xmpp: invalid bind response")
	}
	boundValue := payload.ChildText(bindNS, "jid")
	if boundValue == "" {
		boundValue = payload.ChildText("", "jid")
	}
	bound, err := ParseJID(boundValue)
	if err != nil {
		return fmt.Errorf("xmpp: invalid bound JID %q: %w", boundValue, err)
	}
	c.stateMu.Lock()
	c.jid = bound
	c.stateMu.Unlock()
	c.debugf("resource bound as %s", bound.String())
	return nil
}
func (c *Client) startLegacySession(ctx context.Context) error {
	_, err := c.syncIQ(ctx, IQ{ID: c.NextID(), Type: IQSet, Payloads: []Node{NewNode(legacySessionNS, "session")}})
	if err != nil {
		return fmt.Errorf("xmpp: establish legacy session: %w", err)
	}
	return nil
}
func (c *Client) syncIQ(ctx context.Context, request IQ) (IQ, error) {
	if err := c.writeStanza(request); err != nil {
		return IQ{}, err
	}
	for {
		select {
		case <-ctx.Done():
			return IQ{}, ctx.Err()
		default:
		}
		node, err := c.readTopLevelNode()
		if err != nil {
			return IQ{}, err
		}
		if node.Name.Local != "iq" {
			continue
		}
		stanza, err := stanzaFromNode(node)
		if err != nil {
			return IQ{}, err
		}
		response := stanza.(IQ)
		if response.ID != request.ID {
			continue
		}
		if response.Type == IQError {
			return response, &IQResponseError{IQ: response}
		}
		if response.Type != IQResult {
			return response, fmt.Errorf("xmpp: unexpected IQ response type %q", response.Type)
		}
		return response, nil
	}
}

// Send sends a stanza and emits sent_stanza after a successful write.
func (c *Client) Send(stanza Stanza) error {
	if stanza == nil {
		return fmt.Errorf("xmpp: nil stanza")
	}
	if !c.hasConn() {
		return ErrNotConnected
	}
	if err := c.writeStanza(stanza); err != nil {
		return err
	}
	_ = c.Events.Emit(c.ctx, "sent_stanza", stanza)
	return nil
}
func (c *Client) SendNode(node Node) error {
	if !c.hasConn() {
		return ErrNotConnected
	}
	if err := c.writeNode(node); err != nil {
		return err
	}
	_ = c.Events.Emit(c.ctx, "sent_stream_element", node)
	return nil
}
func (c *Client) RawSend(data []byte) error {
	if !c.hasConn() {
		return ErrNotConnected
	}
	return c.writeRaw(data, true)
}
func (c *Client) SendMessage(to, body string) error {
	return c.Send(Message{To: to, ID: c.NextID(), Type: MessageChat, Body: body})
}
func (c *Client) SendPresence(p Presence) error {
	if p.ID == "" {
		p.ID = c.NextID()
	}
	return c.Send(p)
}
func (c *Client) RequestIQ(ctx context.Context, request IQ) (IQ, error) {
	if request.Type != IQGet && request.Type != IQSet {
		return IQ{}, fmt.Errorf("xmpp: IQ request must use type get or set")
	}
	if request.ID == "" {
		request.ID = c.NextID()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok && c.cfg.IQTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.cfg.IQTimeout)
		defer cancel()
	}
	ch := make(chan IQ, 1)
	c.pendingMu.Lock()
	if _, ok := c.pending[request.ID]; ok {
		c.pendingMu.Unlock()
		return IQ{}, fmt.Errorf("xmpp: duplicate IQ id %q", request.ID)
	}
	c.pending[request.ID] = ch
	c.pendingMu.Unlock()
	defer func() { c.pendingMu.Lock(); delete(c.pending, request.ID); c.pendingMu.Unlock() }()
	if err := c.Send(request); err != nil {
		return IQ{}, err
	}
	select {
	case response := <-ch:
		if response.Type == IQError {
			return response, &IQResponseError{IQ: response}
		}
		return response, nil
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return IQ{}, fmt.Errorf("%w: %s", ErrIQTimeout, request.ID)
		}
		return IQ{}, ctx.Err()
	case <-c.done:
		return IQ{}, ErrClosed
	}
}
func (c *Client) ReplyIQ(request IQ, payloads ...Node) error {
	return c.Send(request.Result(payloads...))
}

func (c *Client) readLoop() {
	defer c.finish(nil)
	for {
		node, err := c.readTopLevelNode()
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) && !errors.Is(err, ErrClosed) {
				c.reportError(err)
			}
			return
		}
		if node.Name.Local == "error" && node.Name.Space == streamNS {
			c.reportError(fmt.Errorf("xmpp: stream error: %s", node.MustXML()))
			return
		}
		if node.Name.Local != "message" && node.Name.Local != "presence" && node.Name.Local != "iq" {
			_ = c.Events.Emit(c.ctx, "stream_element", node)
			continue
		}
		stanza, err := stanzaFromNode(node)
		if err != nil {
			c.reportError(err)
			continue
		}
		if iq, ok := stanza.(IQ); ok && (iq.Type == IQResult || iq.Type == IQError) {
			c.pendingMu.Lock()
			ch := c.pending[iq.ID]
			c.pendingMu.Unlock()
			if ch != nil {
				select {
				case ch <- iq:
				default:
				}
			}
		}
		c.dispatchStanza(stanza)
	}
}
func (c *Client) dispatchStanza(stanza Stanza) {
	_ = c.Events.Emit(c.ctx, "received_stanza", stanza)
	for _, h := range c.handlers.snapshot() {
		if h.Matcher == nil || !h.Matcher.Match(stanza) {
			continue
		}
		if h.Once {
			c.RemoveHandler(h)
		}
		run := func(item *Handler) {
			if err := item.Callback(c.ctx, c, stanza); err != nil {
				c.reportError(fmt.Errorf("xmpp handler %s: %w", item.Name, err))
			}
		}
		if h.Async {
			go run(h)
		} else {
			run(h)
		}
	}
	_ = c.Events.Emit(c.ctx, "stanza", stanza)
	switch v := stanza.(type) {
	case Message:
		if c.OnMessage != nil {
			c.OnMessage(v)
		}
		_ = c.Events.Emit(c.ctx, "message", v)
		if v.Type != "" {
			_ = c.Events.Emit(c.ctx, "message_"+string(v.Type), v)
		}
	case Presence:
		if c.OnPresence != nil {
			c.OnPresence(v)
		}
		_ = c.Events.Emit(c.ctx, "presence", v)
		if v.Type != "" {
			_ = c.Events.Emit(c.ctx, "presence_"+string(v.Type), v)
		}
	case IQ:
		if c.OnIQ != nil {
			c.OnIQ(v)
		}
		_ = c.Events.Emit(c.ctx, "iq", v)
	}
}
func (c *Client) keepaliveLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.writeRaw([]byte(" "), false); err != nil {
				c.reportError(err)
				return
			}
		case <-c.keepaliveDone:
			return
		case <-c.done:
			return
		}
	}
}
func (c *Client) reportError(err error) {
	if err == nil {
		return
	}
	c.debugf("error: %v", err)
	if c.OnError != nil {
		c.OnError(err)
	}
	_ = c.Events.Emit(c.ctx, "error", err)
}

// Close closes plugins, the XML stream, and the transport.
func (c *Client) Close() error {
	c.stateMu.Lock()
	if c.closing {
		c.stateMu.Unlock()
		<-c.done
		return nil
	}
	c.closing = true
	conn := c.conn
	c.stateMu.Unlock()
	select {
	case <-c.keepaliveDone:
	default:
		close(c.keepaliveDone)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	pluginErr := c.Plugins.shutdown(ctx)
	cancel()
	var closeErr error
	if conn != nil {
		_ = c.writeRaw([]byte("</stream:stream>"), false)
		closeErr = conn.Close()
	}
	c.cancel()
	c.finish(closeErr)
	return joinErrors([]error{pluginErr, closeErr})
}
func (c *Client) Disconnect() error     { return c.Close() }
func (c *Client) Done() <-chan struct{} { return c.done }
func (c *Client) Wait()                 { <-c.done }
func (c *Client) finish(reason error) {
	c.doneOnce.Do(func() {
		c.stateMu.Lock()
		c.connected = false
		c.conn = nil
		c.decoder = nil
		c.stateMu.Unlock()
		c.cancel()
		close(c.done)
		_ = c.Events.Emit(context.Background(), "session_end", reason)
		_ = c.Events.Emit(context.Background(), "disconnected", reason)
	})
}
func (c *Client) setConn(conn net.Conn) {
	c.stateMu.Lock()
	c.conn = conn
	c.decoder = xml.NewDecoder(conn)
	c.stateMu.Unlock()
}
func (c *Client) clearConn() {
	c.stateMu.Lock()
	c.conn = nil
	c.decoder = nil
	c.connected = false
	c.stateMu.Unlock()
}
func (c *Client) resetDecoder() {
	c.stateMu.Lock()
	if c.conn != nil {
		c.decoder = xml.NewDecoder(c.conn)
	}
	c.stateMu.Unlock()
}
func (c *Client) currentConn() net.Conn { c.stateMu.RLock(); defer c.stateMu.RUnlock(); return c.conn }
func (c *Client) currentDecoder() *xml.Decoder {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.decoder
}
func (c *Client) hasConn() bool   { return c.currentConn() != nil }
func (c *Client) tlsActive() bool { _, ok := c.currentConn().(*tls.Conn); return ok }
func (c *Client) tlsConfig() *tls.Config {
	var config *tls.Config
	if c.cfg.TLSConfig != nil {
		config = c.cfg.TLSConfig.Clone()
	} else {
		config = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	if config.ServerName == "" {
		config.ServerName = c.JID().Domain
	}
	if config.MinVersion == 0 {
		config.MinVersion = tls.VersionTLS12
	}
	return config
}
func (c *Client) writeStanza(stanza Stanza) error {
	data, err := StanzaXML(stanza)
	if err != nil {
		return err
	}
	return c.writeRaw(data, true)
}
func (c *Client) writeNode(node Node) error {
	var b bytes.Buffer
	enc := xml.NewEncoder(&b)
	if err := enc.Encode(node); err != nil {
		return err
	}
	if err := enc.Flush(); err != nil {
		return err
	}
	return c.writeRaw(b.Bytes(), true)
}
func (c *Client) writeRaw(data []byte, logXML bool) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	conn := c.currentConn()
	if conn == nil {
		return ErrNotConnected
	}
	if logXML {
		c.debugXML("TX", data)
	}
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}
func (c *Client) readTopLevelNode() (Node, error) {
	decoder := c.currentDecoder()
	if decoder == nil {
		return Node{}, ErrNotConnected
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			return Node{}, err
		}
		switch v := token.(type) {
		case xml.StartElement:
			var node Node
			if err := decoder.DecodeElement(&node, &v); err != nil {
				return Node{}, err
			}
			if raw, e := node.XML(); e == nil {
				c.debugXML("RX", []byte(raw))
			}
			return node, nil
		case xml.EndElement:
			if v.Name.Local == "stream" && v.Name.Space == streamNS {
				return Node{}, ErrClosed
			}
		}
	}
}
func (c *Client) debugf(format string, args ...any) {
	if c.cfg.Debug && c.cfg.Logger != nil {
		c.cfg.Logger.Printf(format, args...)
	}
}
func (c *Client) debugXML(direction string, data []byte) {
	if !c.cfg.DebugXML || c.cfg.Logger == nil {
		return
	}
	text := strings.TrimSpace(string(data))
	if strings.Contains(text, "<auth") || strings.Contains(text, "<response") {
		text = "<redacted-sasl/>"
	}
	c.cfg.Logger.Printf("%s %s", direction, text)
}
func escapeXMLAttribute(value string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(value))
	return strings.ReplaceAll(b.String(), "'", "&apos;")
}
