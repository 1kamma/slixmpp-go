// Package omemo integrates legacy OMEMO encrypted messaging with slixmpp-go.
//
// It owns XMPP/PEP protocol integration, trust policy, device lists, bundle XML,
// message envelopes, and payload encryption. Signal/Double-Ratchet session
// operations are intentionally supplied through SessionBackend, matching the
// separation between slixmpp-omemo and python-omemo.
package omemo

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	// Namespace is the legacy OMEMO namespace used by slixmpp-omemo 2.x.
	Namespace        = "eu.siacs.conversations.axolotl"
	DeviceListNode   = Namespace + ".devicelist"
	BundleNodePrefix = Namespace + ".bundles:"
	FallbackBody     = "This message is encrypted with OMEMO and could not be decrypted by this client."
)

var (
	ErrNoDevices        = errors.New("omemo: recipient has no devices")
	ErrNoTrustedDevices = errors.New("omemo: no trusted recipient devices")
	ErrNoKeyForDevice   = errors.New("omemo: message is not encrypted for this device")
	ErrInvalidEnvelope  = errors.New("omemo: invalid encrypted envelope")
	ErrMissingSession   = errors.New("omemo: no cryptographic session")
)

type DeviceID uint32

func (id DeviceID) String() string { return fmt.Sprint(uint32(id)) }

type Address struct {
	JID      string
	DeviceID DeviceID
}

func (a Address) String() string { return fmt.Sprintf("%s:%d", a.JID, a.DeviceID) }

type TrustLevel string

const (
	TrustUndecided  TrustLevel = "undecided"
	TrustTrusted    TrustLevel = "trusted"
	TrustVerified   TrustLevel = "verified"
	TrustDistrusted TrustLevel = "distrusted"
	TrustIgnored    TrustLevel = "ignored"
)

type TrustPolicy string

const (
	TrustManual                  TrustPolicy = "manual"
	TrustBlindBeforeVerification TrustPolicy = "blind-before-verification"
	TrustAll                     TrustPolicy = "trust-all"
)

// PreKey is one public one-time pre-key.
type PreKey struct {
	ID     uint32
	Public []byte
}

// Bundle is the public OMEMO device bundle published over PEP.
type Bundle struct {
	IdentityKey           []byte
	SignedPreKeyID        uint32
	SignedPreKey          []byte
	SignedPreKeySignature []byte
	PreKeys               []PreKey
}

// DeviceList is a PEP-published set of active device IDs.
type DeviceList struct {
	JID       string
	Devices   []DeviceID
	FetchedAt time.Time
}

// DeviceInfo combines protocol, trust, and backend session state.
type DeviceInfo struct {
	Address                       Address
	Trust                         TrustLevel
	Fingerprint                   string
	Active, HasBundle, HasSession bool
	LastSeen                      time.Time
}

// SessionCiphertext is an encrypted key blob produced by a Signal backend.
type SessionCiphertext struct {
	Data   []byte
	PreKey bool
}

// SessionBackend supplies wire-compatible Signal/Double-Ratchet operations.
// Implementations must persist their identity keys and sessions independently
// or through their own storage adapter.
type SessionBackend interface {
	OwnBundle(context.Context) (Bundle, error)
	HasSession(context.Context, Address) (bool, error)
	BuildSession(context.Context, Address, Bundle) error
	Encrypt(context.Context, Address, []byte) (SessionCiphertext, error)
	Decrypt(context.Context, Address, SessionCiphertext) ([]byte, error)
}

type WrappedKey struct {
	Recipient DeviceID
	PreKey    bool
	Data      []byte
}
type Envelope struct {
	Sender      DeviceID
	IV, Payload []byte
	Keys        []WrappedKey
}
type SkippedDevice struct {
	Address Address
	Reason  string
	Trust   TrustLevel
}
type EncryptionResult struct {
	Recipients []Address
	Skipped    []SkippedDevice
	Envelope   Envelope
}
type DecryptionResult struct {
	Sender      Address
	Trust       TrustLevel
	Fingerprint string
	PreKey      bool
}

// Logger is the optional logging interface used by Manager.
type Logger interface{ Printf(string, ...any) }
