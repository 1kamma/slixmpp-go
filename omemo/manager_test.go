package omemo_test

import (
	"context"
	"testing"
	"time"

	"github.com/saret/slixmpp-go/omemo"
	"github.com/saret/slixmpp-go/omemo/testkit"
	"github.com/saret/slixmpp-go/xep"
	"github.com/saret/slixmpp-go/xmpp"
)

func newOfflineManager(t *testing.T, jid string, deviceID omemo.DeviceID, store *omemo.MemoryStore, backend omemo.SessionBackend) (*xmpp.Client, *omemo.Manager) {
	t.Helper()
	client, err := xmpp.NewClient(xmpp.DefaultConfig(jid, "unused"))
	if err != nil {
		t.Fatal(err)
	}
	if err := xep.RegisterAll(client); err != nil {
		t.Fatal(err)
	}
	if err := store.SetOwnDeviceID(context.Background(), deviceID); err != nil {
		t.Fatal(err)
	}
	manager, err := omemo.Install(client, omemo.Options{
		Store:             store,
		Backend:           backend,
		TrustPolicy:       omemo.TrustAll,
		DisableOwnDevices: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client, manager
}

func TestEncryptDecryptWithTestBackend(t *testing.T) {
	ctx := context.Background()
	aliceStore := omemo.NewMemoryStore()
	bobStore := omemo.NewMemoryStore()
	aliceBackend, err := testkit.NewBackend()
	if err != nil {
		t.Fatal(err)
	}
	bobBackend, err := testkit.NewBackend()
	if err != nil {
		t.Fatal(err)
	}

	aliceID := omemo.DeviceID(111)
	bobID := omemo.DeviceID(222)
	aliceClient, alice := newOfflineManager(t, "alice@example.org/laptop", aliceID, aliceStore, aliceBackend)
	_, bob := newOfflineManager(t, "bob@example.org/phone", bobID, bobStore, bobBackend)

	aliceBundle, err := aliceBackend.OwnBundle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bobBundle, err := bobBackend.OwnBundle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := aliceStore.PutDeviceList(ctx, omemo.DeviceList{JID: "bob@example.org", Devices: []omemo.DeviceID{bobID}, FetchedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := aliceStore.PutBundle(ctx, omemo.Address{JID: "bob@example.org", DeviceID: bobID}, bobBundle); err != nil {
		t.Fatal(err)
	}
	if err := bobStore.PutBundle(ctx, omemo.Address{JID: "alice@example.org", DeviceID: aliceID}, aliceBundle); err != nil {
		t.Fatal(err)
	}

	plain := xmpp.Message{
		From: "alice@example.org/laptop",
		To:   "bob@example.org",
		Type: xmpp.MessageChat,
		Body: "secret message",
	}
	encrypted, result, err := alice.EncryptMessage(ctx, plain)
	if err != nil {
		t.Fatal(err)
	}
	if encrypted.Body == plain.Body {
		t.Fatal("plaintext body was not replaced")
	}
	if len(result.Recipients) != 1 || result.Recipients[0].DeviceID != bobID {
		t.Fatalf("recipients = %#v", result.Recipients)
	}
	if encrypted.ID == "" || encrypted.Extension(omemo.Namespace, "encrypted") == nil {
		t.Fatalf("invalid encrypted message: %#v", encrypted)
	}

	decrypted, decryption, err := bob.DecryptFrom(ctx, encrypted, aliceClient.JID().Bare())
	if err != nil {
		t.Fatal(err)
	}
	if decrypted.Body != plain.Body {
		t.Fatalf("decrypted body = %q, want %q", decrypted.Body, plain.Body)
	}
	if decryption.Sender.DeviceID != aliceID || decryption.Sender.JID != "alice@example.org" {
		t.Fatalf("sender = %#v", decryption.Sender)
	}
}
