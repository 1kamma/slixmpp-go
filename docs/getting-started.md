# Getting started

## 1. Create a client

`xmpp.DefaultConfig` supplies secure defaults and enables initial presence:

```go
cfg := xmpp.DefaultConfig("romeo@example.org", password)
cfg.Resource = "desktop"
cfg.Logger = log.Default()

client, err := xmpp.NewClient(cfg)
if err != nil {
    return err
}
```

A JID can be a bare account JID or a full JID. `Config.Resource`, when set,
overrides the resource in `Config.JID`.

## 2. Register XEP factories

```go
if err := xep.RegisterAll(client); err != nil {
    return err
}
```

This registers every Slixmpp 1.17 plugin name. It does not load all plugins and
it does not advertise catalog-only XEPs.

Load a baseline:

```go
if err := xep.LoadDefaults(client); err != nil {
    return err
}
```

Or load one plugin:

```go
plugin, err := xep.Load(client, 313)
if err != nil {
    return err
}
mam := plugin.(*xep.MAM)
```

## 3. Add message handling

The callback fields are convenient for small programs:

```go
client.OnMessage = func(message xmpp.Message) {
    log.Printf("%s: %s", message.From, message.Body)
}
client.OnError = func(err error) {
    log.Printf("XMPP error: %v", err)
}
```

For reusable modules, prefer matchers and named handlers:

```go
handler := client.AddHandler(
    "chat-only",
    xmpp.MatchAnd(
        xmpp.MatchKind("message"),
        xmpp.MatcherFunc(func(stanza xmpp.Stanza) bool {
            message, ok := stanza.(xmpp.Message)
            return ok && message.Type == xmpp.MessageChat
        }),
    ),
    func(ctx context.Context, client *xmpp.Client, stanza xmpp.Stanza) error {
        message := stanza.(xmpp.Message)
        return client.SendMessage(message.From, "received")
    },
)
defer client.RemoveHandler(handler)
```

## 4. Connect and shut down

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

if err := client.Connect(ctx); err != nil {
    return err
}

select {
case <-ctx.Done():
    return client.Close()
case <-client.Done():
    return nil
}
```

`Connect` returns after authentication, resource binding, receive-loop startup,
and initial presence. `Done` closes when the stream terminates.

## 5. Query the roster

```go
items, err := client.Roster.Get(ctx)
if err != nil {
    return err
}
for _, item := range items {
    log.Printf("%s groups=%v", item.JID, item.Groups)
}
```

Roster pushes update the in-memory cache and emit `roster_update` events.

## 6. Inspect raw extensions

Typed coverage is not required to exchange an XEP payload. `xmpp.Node` is a
namespace-aware generic XML tree:

```go
request := xmpp.NewNode("urn:example:feature", "request")
request.SetAttr("mode", "fast")
request.AddChild(xmpp.NewTextNode("urn:example:feature", "value", "42"))

response, err := client.RequestIQ(ctx, xmpp.IQ{
    To:       "service.example.org",
    Type:     xmpp.IQGet,
    Payloads: []xmpp.Node{request},
})
```

`Node` preserves mixed text/child order, which is required by XHTML-like
payloads.
