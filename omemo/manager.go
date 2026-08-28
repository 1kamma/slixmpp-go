package omemo

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/saret/slixmpp-go/xep"
	"github.com/saret/slixmpp-go/xmpp"
)

// Options configures Manager.
type Options struct {
	Store             Store
	Backend           SessionBackend
	TrustPolicy       TrustPolicy
	DeviceListTTL     time.Duration
	DisableOwnDevices bool
	AutoDecrypt       bool
	PublishOnConnect  bool
	FallbackBody      string
	Logger            Logger
	// SenderJID resolves a real bare JID for MUC or gateway messages.
	SenderJID func(xmpp.Message) string
}

// DecryptedMessage is emitted as the omemo_message event.
type DecryptedMessage struct {
	Encrypted xmpp.Message
	Plain     xmpp.Message
	Result    DecryptionResult
}

// Manager is the unified slixmpp-omemo plugin.
type Manager struct {
	options       Options
	client        *xmpp.Client
	pubsub        *xep.PubSub
	mu            sync.Mutex
	handler       *xmpp.Handler
	subscriptions []*xmpp.Subscription
}

func New(options Options) *Manager {
	if options.Store == nil {
		options.Store = NewMemoryStore()
	}
	if options.TrustPolicy == "" {
		options.TrustPolicy = TrustBlindBeforeVerification
	}
	if options.DeviceListTTL <= 0 {
		options.DeviceListTTL = 5 * time.Minute
	}
	if options.FallbackBody == "" {
		options.FallbackBody = FallbackBody
	}
	return &Manager{options: options}
}
func (m *Manager) Name() string        { return "slixmpp_omemo" }
func (m *Manager) Description() string { return "slixmpp-omemo compatible OMEMO integration" }
func (m *Manager) Dependencies() []string {
	return []string{"xep_0060", "xep_0163", "xep_0334", "xep_0380"}
}
func (m *Manager) Features() []string { return []string{Namespace, DeviceListNode} }
func (m *Manager) Init(c *xmpp.Client) error {
	if m.options.Backend == nil {
		return fmt.Errorf("omemo: SessionBackend is required")
	}
	m.client = c
	plugin, ok := c.Plugins.Get("xep_0060")
	if !ok {
		return fmt.Errorf("omemo: xep_0060 is not loaded")
	}
	m.pubsub, ok = plugin.(*xep.PubSub)
	if !ok {
		return fmt.Errorf("omemo: unexpected xep_0060 plugin type")
	}
	m.handler = c.AddHandler("omemo", xmpp.MatchAnd(xmpp.MatchKind("message"), xmpp.MatchPayload(Namespace, "encrypted")), m.handleEncrypted, xmpp.HandlerOptions{Priority: -100})
	m.subscriptions = append(m.subscriptions, c.Events.On("pubsub_event", m.handlePubSub))
	if m.options.PublishOnConnect {
		m.subscriptions = append(m.subscriptions, c.Events.On("session_start", func(ctx context.Context, event xmpp.Event) error {
			go func() {
				publishCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := m.PublishOwn(publishCtx); err != nil {
					m.logf("publish own OMEMO state: %v", err)
				}
			}()
			return nil
		}, xmpp.EventOptions{Async: false}))
	}
	return nil
}
func (m *Manager) Shutdown(context.Context) error {
	if m.client != nil {
		m.client.RemoveHandler(m.handler)
		for _, s := range m.subscriptions {
			m.client.Events.Off(s)
		}
	}
	return nil
}

// Install loads Manager directly into client. Call xep.RegisterAll first.
func Install(client *xmpp.Client, options Options) (*Manager, error) {
	manager := New(options)
	if err := client.Use(manager); err != nil {
		return nil, err
	}
	return manager, nil
}

// RegisterFactory registers a configurable slixmpp_omemo factory.
func RegisterFactory(client *xmpp.Client, options Options) error {
	return client.Plugins.RegisterFactory("slixmpp_omemo", func() xmpp.Plugin { return New(options) })
}

func (m *Manager) OwnDeviceID(ctx context.Context) (DeviceID, error) {
	return m.options.Store.OwnDeviceID(ctx)
}
func (m *Manager) PublishOwn(ctx context.Context) error {
	if err := m.PublishBundle(ctx); err != nil {
		return err
	}
	return m.PublishDeviceList(ctx)
}
func (m *Manager) PublishBundle(ctx context.Context) error {
	id, err := m.OwnDeviceID(ctx)
	if err != nil {
		return err
	}
	bundle, err := m.options.Backend.OwnBundle(ctx)
	if err != nil {
		return fmt.Errorf("omemo: own bundle: %w", err)
	}
	if err := validateBundle(bundle); err != nil {
		return err
	}
	options := pepOptions()
	_, err = m.pubsub.Publish(ctx, "", BundleNode(id), "current", []xmpp.Node{bundle.ToNode()}, &options)
	return err
}
func (m *Manager) PublishDeviceList(ctx context.Context) error {
	ownJID := m.client.JID().Bare()
	id, err := m.OwnDeviceID(ctx)
	if err != nil {
		return err
	}
	list, ok, _ := m.options.Store.DeviceList(ctx, ownJID)
	if !ok || time.Since(list.FetchedAt) > m.options.DeviceListTTL {
		if remote, e := m.RefreshDeviceList(ctx, ownJID); e == nil {
			list = remote
		}
	}
	list.JID = ownJID
	list.Devices = append(list.Devices, id)
	list.Devices = uniqueDevices(list.Devices)
	list.FetchedAt = time.Now()
	options := pepOptions()
	if _, err := m.pubsub.Publish(ctx, "", DeviceListNode, "current", []xmpp.Node{list.ToNode()}, &options); err != nil {
		return err
	}
	return m.options.Store.PutDeviceList(ctx, list)
}
func pepOptions() xep.Form {
	return xep.Form{Fields: []xep.Field{{Var: "pubsub#persist_items", Type: xep.FieldBoolean, Values: []string{"true"}}, {Var: "pubsub#access_model", Type: xep.FieldListSingle, Values: []string{"open"}}, {Var: "pubsub#max_items", Type: xep.FieldTextSingle, Values: []string{"1"}}, {Var: "pubsub#send_last_published_item", Type: xep.FieldListSingle, Values: []string{"on_sub_and_presence"}}}}
}

func (m *Manager) RefreshDeviceList(ctx context.Context, jid string) (DeviceList, error) {
	jid = xmpp.BareJIDString(jid)
	items, _, err := m.pubsub.GetItems(ctx, jid, DeviceListNode, 1)
	if err != nil {
		return DeviceList{}, err
	}
	list := DeviceList{JID: jid, FetchedAt: time.Now()}
	if len(items) > 0 {
		for _, payload := range items[0].Payloads {
			if payload.Name.Space == Namespace && payload.Name.Local == "list" {
				devices, e := ParseDeviceList(payload)
				if e != nil {
					return list, e
				}
				list.Devices = devices
				break
			}
		}
	}
	if err := m.options.Store.PutDeviceList(ctx, list); err != nil {
		return list, err
	}
	_ = m.client.Events.Emit(ctx, "omemo_device_list", list)
	return list, nil
}
func (m *Manager) DeviceList(ctx context.Context, jid string, refresh bool) (DeviceList, error) {
	jid = xmpp.BareJIDString(jid)
	if !refresh {
		if list, ok, err := m.options.Store.DeviceList(ctx, jid); err != nil {
			return DeviceList{}, err
		} else if ok && time.Since(list.FetchedAt) <= m.options.DeviceListTTL {
			return list, nil
		}
	}
	return m.RefreshDeviceList(ctx, jid)
}
func (m *Manager) FetchBundle(ctx context.Context, address Address) (Bundle, error) {
	if bundle, ok, err := m.options.Store.Bundle(ctx, address); err != nil {
		return Bundle{}, err
	} else if ok {
		return bundle, nil
	}
	items, _, err := m.pubsub.GetItems(ctx, address.JID, BundleNode(address.DeviceID), 1)
	if err != nil {
		return Bundle{}, err
	}
	if len(items) == 0 {
		return Bundle{}, fmt.Errorf("omemo: no bundle for %s", address)
	}
	for _, payload := range items[0].Payloads {
		if payload.Name.Space == Namespace && payload.Name.Local == "bundle" {
			bundle, e := ParseBundle(payload)
			if e != nil {
				return Bundle{}, e
			}
			if e = m.options.Store.PutBundle(ctx, address, bundle); e != nil {
				return Bundle{}, e
			}
			return bundle, nil
		}
	}
	return Bundle{}, fmt.Errorf("omemo: bundle payload missing for %s", address)
}
func (m *Manager) EnsureSession(ctx context.Context, address Address) error {
	ok, err := m.options.Backend.HasSession(ctx, address)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	bundle, err := m.FetchBundle(ctx, address)
	if err != nil {
		return err
	}
	if err := m.options.Backend.BuildSession(ctx, address, bundle); err != nil {
		return fmt.Errorf("omemo: build session for %s: %w", address, err)
	}
	return nil
}

func (m *Manager) SetTrust(ctx context.Context, address Address, level TrustLevel) error {
	return m.options.Store.SetTrust(ctx, address, level)
}
func (m *Manager) Trust(ctx context.Context, address Address) (TrustLevel, error) {
	return m.options.Store.Trust(ctx, address)
}
func (m *Manager) Devices(ctx context.Context, jid string, refresh bool) ([]DeviceInfo, error) {
	list, err := m.DeviceList(ctx, jid, refresh)
	if err != nil {
		return nil, err
	}
	out := make([]DeviceInfo, 0, len(list.Devices))
	for _, id := range list.Devices {
		address := Address{JID: list.JID, DeviceID: id}
		trust, err := m.effectiveTrust(ctx, address)
		if err != nil {
			return nil, err
		}
		bundle, hasBundle, err := m.options.Store.Bundle(ctx, address)
		if err != nil {
			return nil, err
		}
		session, err := m.options.Backend.HasSession(ctx, address)
		if err != nil {
			return nil, err
		}
		info := DeviceInfo{Address: address, Trust: trust, Active: true, HasBundle: hasBundle, HasSession: session, LastSeen: list.FetchedAt}
		if hasBundle {
			info.Fingerprint = Fingerprint(bundle.IdentityKey)
		}
		out = append(out, info)
	}
	return out, nil
}
func Fingerprint(identityKey []byte) string {
	sum := sha256.Sum256(identityKey)
	encoded := strings.ToUpper(hex.EncodeToString(sum[:]))
	var groups []string
	for len(encoded) > 0 {
		n := 8
		if len(encoded) < n {
			n = len(encoded)
		}
		groups = append(groups, encoded[:n])
		encoded = encoded[n:]
	}
	return strings.Join(groups, " ")
}

func (m *Manager) EncryptMessage(ctx context.Context, message xmpp.Message, recipients ...string) (xmpp.Message, EncryptionResult, error) {
	if len(recipients) == 0 && message.To != "" {
		recipients = []string{message.To}
	}
	if len(recipients) == 0 {
		return message, EncryptionResult{}, fmt.Errorf("omemo: no recipients")
	}
	ownID, err := m.OwnDeviceID(ctx)
	if err != nil {
		return message, EncryptionResult{}, err
	}
	targets := map[string]Address{}
	for _, recipient := range recipients {
		bare := xmpp.BareJIDString(recipient)
		list, e := m.DeviceList(ctx, bare, false)
		if e != nil {
			continue
		}
		for _, id := range list.Devices {
			address := Address{JID: bare, DeviceID: id}
			targets[address.String()] = address
		}
	}
	if !m.options.DisableOwnDevices {
		own := m.client.JID().Bare()
		list, e := m.DeviceList(ctx, own, false)
		if e == nil {
			for _, id := range list.Devices {
				if id == ownID {
					continue
				}
				address := Address{JID: own, DeviceID: id}
				targets[address.String()] = address
			}
		}
	}
	if len(targets) == 0 {
		return message, EncryptionResult{}, ErrNoDevices
	}
	keyMaterial, iv, payload, err := seal([]byte(message.Body))
	if err != nil {
		return message, EncryptionResult{}, err
	}
	result := EncryptionResult{}
	envelope := Envelope{Sender: ownID, IV: iv, Payload: payload}
	for _, address := range targets {
		trust, e := m.effectiveTrust(ctx, address)
		if e != nil {
			result.Skipped = append(result.Skipped, SkippedDevice{Address: address, Trust: trust, Reason: e.Error()})
			continue
		}
		if trust != TrustTrusted && trust != TrustVerified {
			result.Skipped = append(result.Skipped, SkippedDevice{Address: address, Trust: trust, Reason: "device is not trusted"})
			continue
		}
		if e := m.EnsureSession(ctx, address); e != nil {
			result.Skipped = append(result.Skipped, SkippedDevice{Address: address, Trust: trust, Reason: e.Error()})
			continue
		}
		wrapped, e := m.options.Backend.Encrypt(ctx, address, keyMaterial)
		if e != nil {
			result.Skipped = append(result.Skipped, SkippedDevice{Address: address, Trust: trust, Reason: e.Error()})
			continue
		}
		envelope.Keys = append(envelope.Keys, WrappedKey{Recipient: address.DeviceID, PreKey: wrapped.PreKey, Data: wrapped.Data})
		result.Recipients = append(result.Recipients, address)
	}
	if len(envelope.Keys) == 0 {
		return message, result, ErrNoTrustedDevices
	}
	result.Envelope = envelope
	out := cloneMessage(message)
	out.Body = m.options.FallbackBody
	out.Extensions = withoutOMEMO(out.Extensions)
	out.Extensions = append(out.Extensions, envelope.ToNode())
	xep.SetEncryptionInfo(&out, xep.EncryptionInfo{Namespace: Namespace, Name: "OMEMO"})
	xep.AddHint(&out, xep.HintStore)
	if out.ID == "" {
		out.ID = m.client.NextID()
	}
	if _, ok := xep.GetOriginID(out); !ok {
		xep.SetOriginID(&out, out.ID)
	}
	return out, result, nil
}
func (m *Manager) SendEncrypted(ctx context.Context, message xmpp.Message, recipients ...string) (EncryptionResult, error) {
	encrypted, result, err := m.EncryptMessage(ctx, message, recipients...)
	if err != nil {
		return result, err
	}
	return result, m.client.Send(encrypted)
}
func (m *Manager) DecryptMessage(ctx context.Context, message xmpp.Message) (xmpp.Message, DecryptionResult, error) {
	sender := xmpp.BareJIDString(message.From)
	if m.options.SenderJID != nil {
		if resolved := m.options.SenderJID(message); resolved != "" {
			sender = xmpp.BareJIDString(resolved)
		}
	}
	return m.DecryptFrom(ctx, message, sender)
}
func (m *Manager) DecryptFrom(ctx context.Context, message xmpp.Message, senderJID string) (xmpp.Message, DecryptionResult, error) {
	node := message.Extension(Namespace, "encrypted")
	if node == nil {
		return message, DecryptionResult{}, fmt.Errorf("omemo: message is not encrypted")
	}
	envelope, err := ParseEnvelope(*node)
	if err != nil {
		return message, DecryptionResult{}, err
	}
	ownID, err := m.OwnDeviceID(ctx)
	if err != nil {
		return message, DecryptionResult{}, err
	}
	wrapped, ok := envelope.KeyFor(ownID)
	if !ok {
		return message, DecryptionResult{}, ErrNoKeyForDevice
	}
	address := Address{JID: xmpp.BareJIDString(senderJID), DeviceID: envelope.Sender}
	material, err := m.options.Backend.Decrypt(ctx, address, SessionCiphertext{Data: wrapped.Data, PreKey: wrapped.PreKey})
	if err != nil {
		return message, DecryptionResult{}, fmt.Errorf("omemo: decrypt key from %s: %w", address, err)
	}
	plain, err := open(material, envelope.IV, envelope.Payload)
	if err != nil {
		return message, DecryptionResult{}, err
	}
	trust, err := m.effectiveTrust(ctx, address)
	if err != nil {
		return message, DecryptionResult{}, err
	}
	result := DecryptionResult{Sender: address, Trust: trust, PreKey: wrapped.PreKey}
	if bundle, ok, _ := m.options.Store.Bundle(ctx, address); ok {
		result.Fingerprint = Fingerprint(bundle.IdentityKey)
	}
	out := cloneMessage(message)
	out.Body = string(plain)
	out.Extensions = withoutOMEMO(out.Extensions)
	return out, result, nil
}
func (m *Manager) handleEncrypted(ctx context.Context, c *xmpp.Client, s xmpp.Stanza) error {
	message := asMessage(s)
	plain, result, err := m.DecryptMessage(ctx, message)
	if err != nil {
		_ = c.Events.Emit(ctx, "omemo_decryption_failed", struct {
			Message xmpp.Message
			Err     error
		}{message, err})
		return nil
	}
	event := DecryptedMessage{Encrypted: message, Plain: plain, Result: result}
	if m.options.AutoDecrypt {
		_ = c.Events.Emit(ctx, "message_decrypted", plain)
	}
	return c.Events.Emit(ctx, "omemo_message", event)
}
func (m *Manager) handlePubSub(ctx context.Context, event xmpp.Event) error {
	value, ok := event.Data.(xep.PubSubEvent)
	if !ok || value.Node != DeviceListNode {
		return nil
	}
	for _, item := range value.Items {
		for _, payload := range item.Payloads {
			if payload.Name.Space == Namespace && payload.Name.Local == "list" {
				devices, err := ParseDeviceList(payload)
				if err != nil {
					return err
				}
				list := DeviceList{JID: xmpp.BareJIDString(value.From), Devices: devices, FetchedAt: time.Now()}
				if err := m.options.Store.PutDeviceList(ctx, list); err != nil {
					return err
				}
				return m.client.Events.Emit(ctx, "omemo_device_list", list)
			}
		}
	}
	return nil
}
func (m *Manager) effectiveTrust(ctx context.Context, address Address) (TrustLevel, error) {
	level, err := m.options.Store.Trust(ctx, address)
	if err != nil {
		return level, err
	}
	if level != TrustUndecided {
		return level, nil
	}
	switch m.options.TrustPolicy {
	case TrustAll:
		return TrustTrusted, nil
	case TrustManual:
		return TrustUndecided, nil
	case TrustBlindBeforeVerification:
		addresses, err := m.options.Store.KnownAddresses(ctx)
		if err != nil {
			return level, err
		}
		for _, other := range addresses {
			if other.JID != address.JID {
				continue
			}
			trust, err := m.options.Store.Trust(ctx, other)
			if err != nil {
				return level, err
			}
			if trust == TrustVerified {
				return TrustUndecided, nil
			}
		}
		return TrustTrusted, nil
	default:
		return level, fmt.Errorf("omemo: invalid trust policy %q", m.options.TrustPolicy)
	}
}
func (m *Manager) logf(format string, args ...any) {
	if m.options.Logger != nil {
		m.options.Logger.Printf(format, args...)
	}
}

func seal(plain []byte) (material, iv, payload []byte, err error) {
	key := make([]byte, 16)
	iv = make([]byte, 12)
	if _, err = rand.Read(key); err != nil {
		return nil, nil, nil, err
	}
	if _, err = rand.Read(iv); err != nil {
		return nil, nil, nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, err
	}
	sealed := gcm.Seal(nil, iv, plain, nil)
	tagStart := len(sealed) - gcm.Overhead()
	payload = append([]byte(nil), sealed[:tagStart]...)
	material = append(append([]byte(nil), key...), sealed[tagStart:]...)
	return material, iv, payload, nil
}
func open(material, iv, payload []byte) ([]byte, error) {
	if len(material) < 16 {
		return nil, fmt.Errorf("%w: key material too short", ErrInvalidEnvelope)
	}
	key := material[:16]
	tag := material[16:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(tag) != gcm.Overhead() {
		return nil, fmt.Errorf("%w: invalid authentication tag length", ErrInvalidEnvelope)
	}
	sealed := append(append([]byte(nil), payload...), tag...)
	plain, err := gcm.Open(nil, iv, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("omemo: authenticate payload: %w", err)
	}
	return plain, nil
}
func validateBundle(bundle Bundle) error {
	if len(bundle.IdentityKey) == 0 || len(bundle.SignedPreKey) == 0 || len(bundle.SignedPreKeySignature) == 0 {
		return fmt.Errorf("omemo: backend returned an incomplete bundle")
	}
	return nil
}
func withoutOMEMO(nodes []xmpp.Node) []xmpp.Node {
	out := make([]xmpp.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.Name.Space == Namespace && n.Name.Local == "encrypted" {
			continue
		}
		out = append(out, n.Clone())
	}
	return out
}
func cloneMessage(message xmpp.Message) xmpp.Message {
	out := message
	out.Subjects = append([]xmpp.LangText(nil), message.Subjects...)
	out.Bodies = append([]xmpp.LangText(nil), message.Bodies...)
	out.Extensions = make([]xmpp.Node, len(message.Extensions))
	for i := range message.Extensions {
		out.Extensions[i] = message.Extensions[i].Clone()
	}
	return out
}
func asMessage(s xmpp.Stanza) xmpp.Message {
	switch v := s.(type) {
	case xmpp.Message:
		return v
	case *xmpp.Message:
		return *v
	}
	panic("omemo: stanza is not a message")
}
