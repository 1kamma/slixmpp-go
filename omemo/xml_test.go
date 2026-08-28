package omemo

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	e := Envelope{Sender: 1, IV: []byte("123456789012"), Payload: []byte("cipher"), Keys: []WrappedKey{{Recipient: 2, PreKey: true, Data: []byte("key")}}}
	got, err := ParseEnvelope(e.ToNode())
	if err != nil || got.Sender != 1 || !bytes.Equal(got.Payload, e.Payload) || !got.Keys[0].PreKey {
		t.Fatal(got, err)
	}
}
func TestBundleRoundTrip(t *testing.T) {
	b := Bundle{IdentityKey: []byte{1}, SignedPreKeyID: 2, SignedPreKey: []byte{3}, SignedPreKeySignature: []byte{4}, PreKeys: []PreKey{{ID: 5, Public: []byte{6}}}}
	got, err := ParseBundle(b.ToNode())
	if err != nil || got.SignedPreKeyID != 2 || len(got.PreKeys) != 1 {
		t.Fatal(got, err)
	}
}
func TestJSONStore(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "store.json")
	s, err := OpenJSONStore(path)
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.OwnDeviceID(ctx)
	if err != nil || id == 0 {
		t.Fatal(id, err)
	}
	if err := s.SetTrust(ctx, Address{JID: "a@example", DeviceID: 2}, TrustVerified); err != nil {
		t.Fatal(err)
	}
	loaded, err := OpenJSONStore(path)
	if err != nil {
		t.Fatal(err)
	}
	trust, err := loaded.Trust(ctx, Address{JID: "a@example", DeviceID: 2})
	if err != nil || trust != TrustVerified {
		t.Fatal(trust, err)
	}
}
