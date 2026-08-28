# Configuration and transport

## Secure defaults

`xmpp.DefaultConfig(jid, password)` sets:

- TLS required;
- TLS 1.2 minimum;
- SASL preference: SCRAM-SHA-256, SCRAM-SHA-1, EXTERNAL, PLAIN, ANONYMOUS;
- 20-second connection timeout;
- 30-second IQ timeout;
- English stream language;
- automatic initial presence.

`PLAIN` is rejected on an unencrypted stream unless
`InsecureAllowPlainAuth` is explicitly enabled. That switch is intended only
for isolated test environments.

## Address discovery

When `Address` is empty, the client tries DNS SRV:

- `_xmpp-client._tcp` for STARTTLS connections;
- `_xmpps-client._tcp` for direct TLS.

It falls back to the JID domain on port 5222 or 5223. Set an override for local
or non-standard deployments:

```go
cfg.Address = "xmpp.internal.example:5222"
```

Certificate verification still uses the account JID domain unless
`TLSConfig.ServerName` is explicitly set.

## TLS modes

STARTTLS is the default:

```go
cfg.DirectTLS = false
cfg.RequireTLS = true
```

Direct TLS:

```go
cfg.DirectTLS = true
cfg.Address = "xmpp.example.org:5223"
```

Custom trust roots or client certificates:

```go
cfg.TLSConfig = &tls.Config{
    MinVersion:   tls.VersionTLS13,
    RootCAs:      roots,
    Certificates: []tls.Certificate{certificate},
}
```

Do not set `InsecureSkipVerify` in production.

## SASL

Set an explicit mechanism order when required:

```go
cfg.SASLMechanisms = []string{"SCRAM-SHA-256", "PLAIN"}
```

Supported mechanisms:

- SCRAM-SHA-256;
- SCRAM-SHA-1 for older deployments;
- EXTERNAL when a client certificate is configured;
- PLAIN over TLS;
- ANONYMOUS when offered by the server.

The implementation verifies SCRAM server signatures and rejects malformed or
unreasonably large iteration counts.

## Presence

Initial presence is enabled by default. Customize it:

```go
priority := 10
cfg.InitialPresence = xmpp.Presence{
    Show:     "chat",
    Status:   "available",
    Priority: &priority,
}
```

Disable it for a passive resource:

```go
cfg.DisableAutoPresence = true
```

## Timeouts and keepalive

```go
cfg.ConnectTimeout = 15 * time.Second
cfg.IQTimeout = 20 * time.Second
cfg.KeepAlive = 60 * time.Second
```

The keepalive is XML whitespace. For semantic liveness use XEP-0199 ping and,
where supported, XEP-0198 stream management.

## Diagnostics

```go
cfg.Debug = true
cfg.DebugXML = true
cfg.Logger = log.Default()
```

SASL `<auth/>` and `<response/>` payloads are redacted. Stanza bodies are not;
therefore XML logging can expose message content and should be disabled in
normal production operation.

## External components

Set `ComponentSecret` and an explicit server endpoint:

```go
cfg := xmpp.DefaultConfig("component.example.org", "")
cfg.ComponentSecret = secret
cfg.Address = "example.org:5347"
```

Component mode uses XEP-0114 authentication instead of SASL/resource binding.
See [components.md](components.md).
