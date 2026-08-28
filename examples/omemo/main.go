// Command omemo demonstrates the OMEMO protocol integration with the
// deliberately non-interoperable testkit backend. It never connects to a
// network and must not be used as a production encryption backend.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/saret/slixmpp-go/omemo"
	"github.com/saret/slixmpp-go/omemo/testkit"
	"github.com/saret/slixmpp-go/xep"
	"github.com/saret/slixmpp-go/xmpp"
)

func main() {
	log.Println("WARNING: omemo/testkit is for local integration tests only; it is not Signal/OMEMO interoperable")
	ctx := context.Background()

	aliceClient, aliceStore, aliceBackend, alice := newParty(ctx, "alice@example.org/laptop", 101)
	_, bobStore, bobBackend, bob := newParty(ctx, "bob@example.org/phone", 202)

	aliceBundle, _ := aliceBackend.OwnBundle(ctx)
	bobBundle, _ := bobBackend.OwnBundle(ctx)
	_ = aliceStore.PutDeviceList(ctx, omemo.DeviceList{JID: "bob@example.org", Devices: []omemo.DeviceID{202}, FetchedAt: time.Now()})
	_ = aliceStore.PutBundle(ctx, omemo.Address{JID: "bob@example.org", DeviceID: 202}, bobBundle)
	_ = bobStore.PutBundle(ctx, omemo.Address{JID: "alice@example.org", DeviceID: 101}, aliceBundle)

	message := xmpp.Message{From: aliceClient.JID().String(), To: "bob@example.org", Type: xmpp.MessageChat, Body: "hello through the OMEMO integration layer"}
	encrypted, result, err := alice.EncryptMessage(ctx, message)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("encrypted for %d device(s); payload body=%q\n", len(result.Recipients), encrypted.Body)

	plain, _, err := bob.DecryptFrom(ctx, encrypted, "alice@example.org")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("decrypted body=%q\n", plain.Body)
}

func newParty(ctx context.Context, jid string, deviceID omemo.DeviceID) (*xmpp.Client, *omemo.MemoryStore, *testkit.Backend, *omemo.Manager) {
	client := xmpp.MustNewClient(xmpp.DefaultConfig(jid, "unused"))
	if err := xep.RegisterAll(client); err != nil {
		log.Fatal(err)
	}
	store := omemo.NewMemoryStore()
	if err := store.SetOwnDeviceID(ctx, deviceID); err != nil {
		log.Fatal(err)
	}
	backend, err := testkit.NewBackend()
	if err != nil {
		log.Fatal(err)
	}
	manager, err := omemo.Install(client, omemo.Options{
		Store:             store,
		Backend:           backend,
		TrustPolicy:       omemo.TrustAll,
		DisableOwnDevices: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	return client, store, backend, manager
}
