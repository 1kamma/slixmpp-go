# XEP-0114 external components

A component uses the same stanza, event, plugin, and generic XML APIs as a
normal client, but authenticates with the XEP-0114 stream handshake.

```go
cfg := xmpp.DefaultConfig("weather.example.org", "")
cfg.ComponentSecret = os.Getenv("XMPP_COMPONENT_SECRET")
cfg.Address = "example.org:5347"
cfg.Logger = log.Default()

component, err := xmpp.NewClient(cfg)
if err != nil {
    return err
}
if err := xep.RegisterAll(component); err != nil {
    return err
}
if err := component.Connect(ctx); err != nil {
    return err
}
```

`Config.JID` must be a domain JID. `Address` is mandatory because the component
domain is frequently different from the accepting server host.

For a stanza originating from the component, set `From` to a JID within the
component's delegated domain:

```go
reply := incoming.Reply("forecast: sunny")
reply.ID = component.NextID()
reply.From = incoming.To
err := component.Send(reply)
```

XEP-0114's SHA-1 handshake is retained solely because the protocol requires it.
Use a protected network path or direct TLS where supported, and use a strong,
unique component secret.

Run the example:

```bash
go run ./examples/component \
  -jid weather.example.org \
  -address example.org:5347 \
  -secret "$XMPP_COMPONENT_SECRET"
```
