# Testing

## Standard verification

```bash
go test ./...
go vet ./...
go test -race ./...
```

The module has no required third-party dependencies, so the core suite can run
offline after the Go toolchain is installed.

## What the tests cover

- JID parsing and XEP-0106 escaping;
- mixed-content XML node round trips;
- message, presence, IQ, and stanza-error serialization;
- event priority/once/wildcard/error behavior;
- API specificity;
- PBKDF2/SCRAM helpers;
- a local end-to-end stream covering STARTTLS, SASL PLAIN, resource binding, initial presence, and inbound message delivery;
- data forms and RSM;
- message extension and payload round trips;
- MUC, PubSub, and MAM parsers;
- OMEMO device list, bundle, and envelope XML;
- offline OMEMO integration through the test backend;
- persistent store behavior;
- race-free access across packages.

## Test backend warning

`omemo/testkit` exists to test the protocol integration without importing an
external cryptographic implementation. It is deliberately unsuitable for
security or interoperability tests.

A production backend should add its own suite covering:

1. published bundle compatibility;
2. prekey and normal Signal message decoding;
3. session creation in both directions;
4. skipped message keys and out-of-order delivery;
5. identity-key changes;
6. one-time prekey consumption;
7. cross-implementation test vectors;
8. restart/persistence behavior.

## Regenerating documentation

```bash
go run ./cmd/xep-matrix > docs/xep-support.md
```

Commit matrix changes together with the implementation and coverage metadata.

## Integration testing against a server

Use dedicated test accounts and enable diagnostics:

```bash
go run ./examples/echo \
  -jid test@example.org \
  -password "$XMPP_PASSWORD" \
  -debug -debug-xml
```

Do not use production accounts with XML logging. Test STARTTLS, direct TLS,
SCRAM, roster pushes, bare-JID routing, MUC presence, PubSub events, MAM paging,
stream acknowledgements, and server disconnect behavior separately.

## Generated documentation

`make docs` regenerates both the XEP support matrix and complete exported API
references under `docs/api`. `make check` fails when generated documentation is
stale.
