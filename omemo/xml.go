package omemo

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/1kamma/slixmpp-go/xmpp"
)

func BundleNode(device DeviceID) string { return BundleNodePrefix + device.String() }

func (list DeviceList) ToNode() xmpp.Node {
	n := xmpp.NewNode(Namespace, "list")
	for _, id := range uniqueDevices(list.Devices) {
		d := xmpp.NewNode(Namespace, "device")
		d.SetAttr("id", id.String())
		n.AddChild(d)
	}
	return n
}
func ParseDeviceList(node xmpp.Node) ([]DeviceID, error) {
	if node.Name.Space != Namespace || node.Name.Local != "list" {
		return nil, fmt.Errorf("%w: invalid device list", ErrInvalidEnvelope)
	}
	var out []DeviceID
	for _, child := range node.Children() {
		if child.Name.Local != "device" {
			continue
		}
		value, _ := child.AttrValue("id")
		id, err := parseDeviceID(value)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return uniqueDevices(out), nil
}

func (bundle Bundle) ToNode() xmpp.Node {
	n := xmpp.NewNode(Namespace, "bundle")
	signed := xmpp.NewTextNode(Namespace, "signedPreKeyPublic", base64.StdEncoding.EncodeToString(bundle.SignedPreKey))
	signed.SetAttr("signedPreKeyId", strconv.FormatUint(uint64(bundle.SignedPreKeyID), 10))
	n.AddChild(signed)
	n.AddChild(xmpp.NewTextNode(Namespace, "signedPreKeySignature", base64.StdEncoding.EncodeToString(bundle.SignedPreKeySignature)))
	n.AddChild(xmpp.NewTextNode(Namespace, "identityKey", base64.StdEncoding.EncodeToString(bundle.IdentityKey)))
	prekeys := xmpp.NewNode(Namespace, "prekeys")
	for _, key := range bundle.PreKeys {
		p := xmpp.NewTextNode(Namespace, "preKeyPublic", base64.StdEncoding.EncodeToString(key.Public))
		p.SetAttr("preKeyId", strconv.FormatUint(uint64(key.ID), 10))
		prekeys.AddChild(p)
	}
	n.AddChild(prekeys)
	return n
}
func ParseBundle(node xmpp.Node) (Bundle, error) {
	if node.Name.Space != Namespace || node.Name.Local != "bundle" {
		return Bundle{}, fmt.Errorf("omemo: invalid bundle payload")
	}
	var b Bundle
	decode := func(child *xmpp.Node) ([]byte, error) {
		if child == nil {
			return nil, fmt.Errorf("omemo: missing bundle element")
		}
		return base64.StdEncoding.DecodeString(strings.TrimSpace(child.Text()))
	}
	signed := node.Child(Namespace, "signedPreKeyPublic")
	if signed == nil {
		return b, fmt.Errorf("omemo: bundle has no signed pre-key")
	}
	idText, _ := signed.AttrValue("signedPreKeyId")
	id, err := strconv.ParseUint(idText, 10, 32)
	if err != nil {
		return b, fmt.Errorf("omemo: invalid signed pre-key ID: %w", err)
	}
	b.SignedPreKeyID = uint32(id)
	if b.SignedPreKey, err = decode(signed); err != nil {
		return b, err
	}
	if b.SignedPreKeySignature, err = decode(node.Child(Namespace, "signedPreKeySignature")); err != nil {
		return b, err
	}
	if b.IdentityKey, err = decode(node.Child(Namespace, "identityKey")); err != nil {
		return b, err
	}
	if prekeys := node.Child(Namespace, "prekeys"); prekeys != nil {
		for _, child := range prekeys.Children() {
			if child.Name.Local != "preKeyPublic" {
				continue
			}
			value, _ := child.AttrValue("preKeyId")
			id, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return b, err
			}
			public, err := base64.StdEncoding.DecodeString(strings.TrimSpace(child.Text()))
			if err != nil {
				return b, err
			}
			b.PreKeys = append(b.PreKeys, PreKey{ID: uint32(id), Public: public})
		}
	}
	if len(b.IdentityKey) == 0 || len(b.SignedPreKey) == 0 || len(b.SignedPreKeySignature) == 0 {
		return b, fmt.Errorf("omemo: incomplete bundle")
	}
	return b, nil
}

func (envelope Envelope) ToNode() xmpp.Node {
	n := xmpp.NewNode(Namespace, "encrypted")
	header := xmpp.NewNode(Namespace, "header")
	header.SetAttr("sid", envelope.Sender.String())
	for _, key := range envelope.Keys {
		k := xmpp.NewTextNode(Namespace, "key", base64.StdEncoding.EncodeToString(key.Data))
		k.SetAttr("rid", key.Recipient.String())
		if key.PreKey {
			k.SetAttr("prekey", "true")
		}
		header.AddChild(k)
	}
	header.AddChild(xmpp.NewTextNode(Namespace, "iv", base64.StdEncoding.EncodeToString(envelope.IV)))
	n.AddChild(header)
	if len(envelope.Payload) > 0 {
		n.AddChild(xmpp.NewTextNode(Namespace, "payload", base64.StdEncoding.EncodeToString(envelope.Payload)))
	}
	return n
}
func ParseEnvelope(node xmpp.Node) (Envelope, error) {
	if node.Name.Space != Namespace || node.Name.Local != "encrypted" {
		return Envelope{}, ErrInvalidEnvelope
	}
	header := node.Child(Namespace, "header")
	if header == nil {
		return Envelope{}, fmt.Errorf("%w: missing header", ErrInvalidEnvelope)
	}
	sid, _ := header.AttrValue("sid")
	sender, err := parseDeviceID(sid)
	if err != nil {
		return Envelope{}, err
	}
	e := Envelope{Sender: sender}
	for _, child := range header.Children() {
		switch child.Name.Local {
		case "key":
			rid, _ := child.AttrValue("rid")
			device, err := parseDeviceID(rid)
			if err != nil {
				return e, err
			}
			data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(child.Text()))
			if err != nil {
				return e, fmt.Errorf("%w: invalid key encoding", ErrInvalidEnvelope)
			}
			pre := false
			if value, ok := child.AttrValue("prekey"); ok {
				pre = value == "true" || value == "1"
			}
			e.Keys = append(e.Keys, WrappedKey{Recipient: device, PreKey: pre, Data: data})
		case "iv":
			e.IV, err = base64.StdEncoding.DecodeString(strings.TrimSpace(child.Text()))
			if err != nil {
				return e, fmt.Errorf("%w: invalid IV encoding", ErrInvalidEnvelope)
			}
		}
	}
	if payload := node.Child(Namespace, "payload"); payload != nil {
		e.Payload, err = base64.StdEncoding.DecodeString(strings.TrimSpace(payload.Text()))
		if err != nil {
			return e, fmt.Errorf("%w: invalid payload encoding", ErrInvalidEnvelope)
		}
	}
	if len(e.IV) == 0 || len(e.Keys) == 0 {
		return e, fmt.Errorf("%w: incomplete envelope", ErrInvalidEnvelope)
	}
	return e, nil
}
func (envelope Envelope) KeyFor(device DeviceID) (WrappedKey, bool) {
	for _, key := range envelope.Keys {
		if key.Recipient == device {
			return key, true
		}
	}
	return WrappedKey{}, false
}
func parseDeviceID(value string) (DeviceID, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("omemo: invalid device ID %q", value)
	}
	return DeviceID(id), nil
}
