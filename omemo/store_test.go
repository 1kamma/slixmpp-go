package omemo

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestJSONStorePersistsGeneratedDeviceIDAndAddresses(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "omemo-state.json")
	store, err := OpenJSONStore(path)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := store.OwnDeviceID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if generated == 0 {
		t.Fatal("generated device ID is zero")
	}

	address := Address{JID: "user:tag@example.org", DeviceID: 7788}
	if err := store.PutBundle(ctx, address, Bundle{IdentityKey: []byte("identity")}); err != nil {
		t.Fatal(err)
	}
	if err := store.SetTrust(ctx, address, TrustVerified); err != nil {
		t.Fatal(err)
	}
	if err := store.PutDeviceList(ctx, DeviceList{JID: "user:tag@example.org", Devices: []DeviceID{7788}, FetchedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenJSONStore(path)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := reopened.OwnDeviceID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if persisted != generated {
		t.Fatalf("persisted device ID = %d, want %d", persisted, generated)
	}
	addresses, err := reopened.KnownAddresses(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(addresses) != 1 || addresses[0] != address {
		t.Fatalf("known addresses = %#v, want %#v", addresses, address)
	}
	trust, err := reopened.Trust(ctx, address)
	if err != nil {
		t.Fatal(err)
	}
	if trust != TrustVerified {
		t.Fatalf("trust = %s, want %s", trust, TrustVerified)
	}
}
