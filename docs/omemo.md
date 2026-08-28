# OMEMO integration

## Scope

The package targets the legacy OMEMO namespace used by the referenced
`slixmpp-omemo` release:

```text
eu.siacs.conversations.axolotl
```

It implements the XMPP-facing responsibilities:

- PEP device-list publication and retrieval;
- per-device bundle publication and retrieval;
- cached device and bundle metadata;
- trust state and trust-policy decisions;
- session establishment requests through a backend;
- random AES-GCM content keys and IVs;
- per-device key wrapping through the backend;
- legacy `<encrypted/>` envelope generation/parsing;
- EME and message-processing hint integration;
- encrypted-message detection, decryption, and semantic events;
- JSON or application-defined metadata storage.

It does **not** implement Signal's X3DH/Double Ratchet internally. That is an
explicit backend boundary, matching the separation between `slixmpp-omemo` and
python-omemo.

## Backend contract

```go
type SessionBackend interface {
    OwnBundle(context.Context) (Bundle, error)
    BuildSession(context.Context, Address, Bundle) error
    HasSession(context.Context, Address) (bool, error)
    EncryptKey(context.Context, Address, []byte) (wrapped []byte, preKey bool, err error)
    DecryptKey(context.Context, Address, []byte, bool) ([]byte, error)
    Fingerprint(context.Context, Address, *Bundle) (string, error)
}
```

A production implementation must provide wire-compatible Signal messages and
persist all cryptographic state. `Store` does not persist backend sessions.

## Installation

Register XEP factories first because OMEMO depends on PubSub/PEP, message hints,
and EME:

```go
if err := xep.RegisterAll(client); err != nil {
    return err
}

store, err := omemo.OpenJSONStore("./omemo-state.json")
if err != nil {
    return err
}

manager, err := omemo.Install(client, omemo.Options{
    Store:             store,
    Backend:           signalBackend,
    TrustPolicy:       omemo.TrustBlindBeforeVerification,
    AutoDecrypt:       true,
    PublishOnConnect:  true,
    IncludeOwnDevices: true,
    DeviceListTTL:     15 * time.Minute,
})
```

`Install` loads required XEP plugins and installs the OMEMO manager as
`slixmpp_omemo`.

## Publication

```go
if err := manager.PublishDeviceList(ctx); err != nil {
    return err
}
if err := manager.PublishBundle(ctx); err != nil {
    return err
}
```

With `PublishOnConnect`, both operations run after `session_start`.
Applications should surface publication failures and retry under bounded policy.

## Encryption

```go
message := xmpp.Message{
    To:   "juliet@example.org",
    Type: xmpp.MessageChat,
    Body: "secret",
}

encrypted, result, err := manager.EncryptMessage(ctx, message)
if err != nil {
    return err
}
log.Printf("encrypted for %d devices", len(result.Recipients))
return client.Send(encrypted)
```

Or encrypt and send in one call:

```go
result, err := manager.SendEncrypted(ctx, message)
```

Recipients are selected from fresh or refreshed device lists. Disabled devices
are skipped. Own devices can be included for multi-device synchronization.

## Decryption events

Every successfully decrypted incoming message emits `omemo_message` with
`omemo.DecryptionResult`. When `AutoDecrypt` is set, `message_decrypted` is also
emitted with the plaintext `xmpp.Message`.

```go
client.Events.On("omemo_message", func(ctx context.Context, event xmpp.Event) error {
    result := event.Data.(omemo.DecryptionResult)
    log.Printf("%s/%d: %s", result.Sender.JID, result.Sender.DeviceID, result.Message.Body)
    return nil
})
```

The original encrypted stanza is retained in `DecryptionResult.Encrypted`.

## Trust

```go
addresses, err := manager.Devices(ctx, "juliet@example.org")
fingerprint, err := manager.Fingerprint(ctx, addresses[0])
err = manager.SetTrust(ctx, addresses[0], omemo.TrustVerified)
```

Trust levels:

- undecided;
- blindly trusted;
- verified;
- distrusted.

Policies:

- `TrustAll` — encrypt to all non-distrusted devices;
- `TrustBlindBeforeVerification` — first-use trust until a verified identity
  exists for that JID;
- `TrustManual` — only explicitly trusted/verified devices.

A user interface should display fingerprints and make identity changes visible.

## Persistence

`MemoryStore` is suitable for tests. `JSONStore` persists device IDs, device
lists, bundles, trust decisions, and disabled-device flags using atomic mode
`0600` writes.

The backend must separately persist:

- local identity key pair;
- signed prekey and signature;
- one-time prekeys and consumption state;
- remote identity keys;
- sending and receiving ratchets;
- skipped message keys;
- registration/device metadata required by its Signal format.

## Testkit

`omemo/testkit.Backend` only demonstrates the interface and exercises XML,
trust, recipient selection, and AES-GCM envelope flow. It derives simple keys
and does not implement Signal. The example prints a warning and never connects
to a server.

```bash
go run ./examples/omemo
```

Do not substitute this backend in a network client.
