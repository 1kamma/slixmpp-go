# slixmpp-go

A unified, Go-native XMPP client/component library inspired by the public
architecture and plugin surfaces of **Slixmpp 1.17** and
**slixmpp-omemo 2.2**.

The repository combines:

- an RFC 6120/6121 client stream;
- XEP-0114 external components;
- Slixmpp-style events, stanza matchers, plugin dependencies, and scoped APIs;
- a complete catalog of the `xep_NNNN` names documented by Slixmpp 1.17;
- typed and tested implementations for the most commonly deployed XEPs;
- the XMPP/PEP/trust/envelope layer needed by legacy OMEMO;
- examples, migration notes, a generated support matrix, and tests.

## Read this first

This is an independent Go implementation, not a mechanical Python-to-Go
transliteration. Python coroutines and dictionary-like stanza plugins become
`context.Context`, goroutines, interfaces, typed structs, and namespace-aware
`xmpp.Node` values.

The implementation status is intentionally explicit:

- **operational** entries have typed behavior and protocol tests within the
  documented scope;
- **client** entries expose typed client operations and incoming events, but may
  omit optional or uncommon branches of the XEP;
- **payload** entries provide typed XML builders/parsers;
- **catalog** entries preserve the familiar Slixmpp plugin name but advertise no
  service-discovery feature and make no behavioral claim.

See [the generated XEP support matrix](docs/xep-support.md). The matrix is the
authoritative statement of current coverage.

### OMEMO boundary

The `omemo` package implements device lists, bundles, PEP publication,
trust policy, content encryption, legacy OMEMO envelopes, message events, and
storage contracts. Like `slixmpp-omemo`, it delegates the Signal/X3DH/Double
Ratchet session engine to a backend interface.

A production application **must provide a wire-compatible Signal backend** by
implementing `omemo.SessionBackend`. The bundled `omemo/testkit` backend is
only for local tests and examples. It is deliberately not Signal-compatible and
must never protect real messages.

## Requirements

- Go 1.23 or newer.
- An XMPP server supporting the features your application enables.
- For real OMEMO interoperability, a production `omemo.SessionBackend` adapter.

The core module uses only the Go standard library.

## Quick start

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/1kamma/slixmpp-go/xep"
    "github.com/1kamma/slixmpp-go/xmpp"
)

func main() {
    cfg := xmpp.DefaultConfig(os.Getenv("XMPP_JID"), os.Getenv("XMPP_PASSWORD"))
    cfg.Resource = "go-bot"
    cfg.Debug = true
    cfg.Logger = log.Default()

    client, err := xmpp.NewClient(cfg)
    if err != nil {
        log.Fatal(err)
    }
    if err := xep.RegisterAll(client); err != nil {
        log.Fatal(err)
    }
    if err := xep.LoadDefaults(client); err != nil {
        log.Fatal(err)
    }

    client.OnMessage = func(message xmpp.Message) {
        if message.Type != xmpp.MessageChat || message.Body == "" {
            return
        }
        reply := message.Reply("echo: " + message.Body)
        reply.ID = client.NextID()
        if err := client.Send(reply); err != nil {
            log.Printf("send: %v", err)
        }
    }

    if err := client.Connect(context.Background()); err != nil {
        log.Fatal(err)
    }
    client.Wait()
}
```

Run the complete example:

```bash
export XMPP_JID='bot@example.org'
export XMPP_PASSWORD='secret'
go run ./examples/echo --debug
```

A positional form is also accepted when flags precede it:

```bash
go run ./examples/echo --debug bot@example.org secret
```

## Package layout

| Package | Purpose |
|---|---|
| `xmpp` | Stream negotiation, TLS, SASL, JIDs, stanzas, XML nodes, events, handlers, plugins, APIs, roster, and XEP-0114 components. |
| `xep` | Typed XEP payloads and stateful client plugins. |
| `omemo` | Legacy OMEMO protocol integration, trust, storage, and backend contracts. |
| `omemo/testkit` | Insecure, non-interoperable backend for tests only. |
| `cmd/xep-matrix` | Generates `docs/xep-support.md` from the compiled catalog. |
| `examples` | Echo, MUC, component, and offline OMEMO examples. |

## Core capabilities

### Stream and authentication

- DNS SRV lookup with host/port override.
- STARTTLS and direct TLS.
- TLS 1.2 minimum by default.
- SASL SCRAM-SHA-256, SCRAM-SHA-1, PLAIN, EXTERNAL, and ANONYMOUS.
- Resource binding and legacy session establishment.
- Initial presence, keepalive whitespace, IQ correlation, and structured stanza
  errors.
- XEP-0114 component handshake.

### Slixmpp-style application model

- Named event bus with priorities, one-shot handlers, wildcard handlers, and
  optional asynchronous dispatch.
- Composable stanza matchers.
- Plugin factories with dependency loading and feature aggregation.
- Scoped internal API registry with JID/node specificity.
- Generic, mixed-content XML nodes for XEPs not yet typed.

### Stateful protocol modules

Current typed client modules include service discovery, roster/subscriptions,
MUC, PubSub/PEP, MAM, message receipts, blocking, carbons, CSI, stream
management acknowledgements, bookmarks, HTTP upload, ping, entity time, and
software version. See the matrix for exact scope.

## Loading plugins

Register factories once, then load by XEP number:

```go
if err := xep.RegisterAll(client); err != nil {
    log.Fatal(err)
}

plugin, err := xep.Load(client, 45)
if err != nil {
    log.Fatal(err)
}
muc := plugin.(*xep.MUC)
```

`xep.LoadDefaults` loads a conservative baseline. Loading a catalog-only plugin
is safe: it returns metadata and advertises no unsupported feature.

## OMEMO installation

```go
store, err := omemo.OpenJSONStore("./omemo-state.json")
if err != nil {
    log.Fatal(err)
}

manager, err := omemo.Install(client, omemo.Options{
    Store:       store,
    Backend:     productionSignalBackend,
    TrustPolicy: omemo.TrustBlindBeforeVerification,
    AutoDecrypt: true,
})
if err != nil {
    log.Fatal(err)
}

client.Events.On("omemo_message", func(ctx context.Context, event xmpp.Event) error {
    result := event.Data.(omemo.DecryptionResult)
    log.Printf("decrypted from %s: %s", result.Sender, result.Message.Body)
    return nil
})
```

Read [the OMEMO guide](docs/omemo.md) before using this package. It documents
the backend contract, trust behavior, persistence, publication workflow, and
security boundary.

## Examples

```bash
# Direct chat transport smoke test
go run ./examples/echo -jid bot@example.org -password secret -debug

# Interactive room client
go run ./examples/muc -jid bot@example.org -password secret \
  -room room@conference.example.org -nick go-bot

# XEP-0114 external component
go run ./examples/component -jid weather.example.org \
  -address example.org:5347 -secret component-secret

# Offline OMEMO integration round trip; test backend only
go run ./examples/omemo
```

## Documentation

- [Getting started](docs/getting-started.md)
- [Configuration and transport](docs/configuration.md)
- [Events, matchers, and handlers](docs/events-and-handlers.md)
- [Plugin architecture](docs/plugins.md)
- [Roster and messaging](docs/roster-and-messaging.md)
- [MUC, PubSub, and MAM](docs/stateful-xeps.md)
- [OMEMO integration](docs/omemo.md)
- [External components](docs/components.md)
- [Migration from Slixmpp](docs/migration-from-slixmpp.md)
- [Source and API map](docs/source-map.md)
- [Architecture](docs/architecture.md)
- [Security](docs/security.md)
- [Testing](docs/testing.md)
- [Generated API reference](docs/api/xmpp.md) (`xmpp`, `xep`, `omemo`, and `omemo/testkit`)
- [XEP support matrix](docs/xep-support.md)

Every exported Go symbol is also available through standard Go documentation:

```bash
go doc ./xmpp
go doc ./xep
go doc ./omemo
```

## Build and verification

```bash
make check
make verify
# Regenerate the support matrix and exported API references:
make docs
```

The delivered archive is verified with `go test ./...`, `go vet ./...`,
`go test -race ./...`, a generated-matrix consistency check, the offline OMEMO
round trip, and a local STARTTLS/SASL/bind/presence/message integration test.

## Module path

The current module path is:

```text
github.com/1kamma/slixmpp-go
```

Change the `module` line and imports before publishing under another namespace.

## Licensing and attribution

The combined repository is licensed under **AGPL-3.0-or-later**. This is
compatible with the licensing boundary introduced by the AGPL-licensed
`slixmpp-omemo` project. Slixmpp and python-omemo attribution is recorded in
`NOTICE`.

No Python source code is embedded. Protocol behavior is implemented from public
XMPP RFCs/XEPs and public API documentation.
