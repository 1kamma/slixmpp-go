# Security

## Transport

- TLS is required by default.
- TLS 1.2 is the minimum unless a stricter `TLSConfig` is supplied.
- SASL PLAIN is selected only on an encrypted stream, except when the explicit
  test-only override is enabled.
- SCRAM verifies the server signature.
- Certificate verification uses the account domain by default, even when SRV
  or `Address` points to another host.

Never use `tls.Config.InsecureSkipVerify` or
`Config.InsecureAllowPlainAuth` in production.

## XML and stanza handling

Outgoing XML is produced by `encoding/xml`; stanza fields and node text are
escaped. `RawSend` bypasses typed validation and should only receive trusted,
fully formed XML.

The decoder preserves unknown extensions rather than executing them. Plugins
must validate lengths, counts, URLs, hashes, and identifiers before using them
outside the XML layer.

## Logging

SASL payloads are redacted from debug XML logs. Message bodies, JIDs, room
subjects, URLs, and extension payloads are not. Treat `DebugXML` as sensitive.

## HTTP upload

`xep.HTTPUpload.Upload` rejects non-HTTPS slot URLs by default. Redirect and
certificate policy are governed by the supplied `http.Client`. Consider a
restricted client that rejects cross-origin redirects for high-assurance use.

## Storage

`omemo.JSONStore` writes atomically with mode `0600`, but it stores metadata,
bundles, and trust decisions—not cryptographic session state. A production
backend must protect identity keys, signed prekeys, one-time prekeys, and
ratchet state with equivalent or stronger controls.

## OMEMO

The `omemo` package's AES-GCM content encryption and legacy envelope handling do
not replace Signal session security. Confidentiality and forward secrecy depend
on `SessionBackend`.

`omemo/testkit`:

- is deterministic enough for integration testing;
- is not X3DH;
- is not Double Ratchet;
- is not wire-compatible with OMEMO clients;
- must never be used for real messages.

Use a reviewed, interoperable backend and add cross-client test vectors before
production deployment.

## Trust policy

Available policies include trust-all, blind-trust-before-verification, and
manual trust. BTFV is convenient but changes security state when first seeing a
fingerprint. Applications should expose fingerprint verification and persist
explicit trust decisions.

## Reporting vulnerabilities

See the repository-level `SECURITY.md`. Do not include live credentials,
private keys, raw OMEMO sessions, or message content in a public issue.
