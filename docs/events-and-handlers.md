# Events, matchers, and stanza handlers

## Event bus

Every client has `client.Events`. Events are identified by strings and carry an
`any` payload:

```go
subscription := client.Events.On("message", func(ctx context.Context, event xmpp.Event) error {
    message := event.Data.(xmpp.Message)
    log.Printf("%s", message.Body)
    return nil
})
defer client.Events.Off(subscription)
```

Options:

```go
client.Events.On("message", handler, xmpp.EventOptions{
    Priority: -10,
    Once:     false,
    Async:    true,
})
```

Lower priority values run first. A handler registered for `*` receives all
events. Synchronous errors are joined and returned by `Emit`; asynchronous
handler errors cannot be returned to the emitter.

Core event names include:

- `connected`, `session_start`, `session_end`, `disconnected`;
- `sent_stanza`, `received_stanza`, `stanza`;
- `message`, `message_chat`, `message_groupchat`;
- `presence` and type-specific presence events;
- `iq`, `stream_element`, `error`;
- `plugin_loaded`, `roster_update`.

Plugins add names such as `muc_joined`, `pubsub_event`, `mam_result`,
`blocklist_changed`, and `omemo_message`.

## Stanza handlers

Handlers combine a matcher and callback:

```go
handler := client.AddHandler(
    "receipts",
    xmpp.MatchPayload("urn:xmpp:receipts", "received"),
    func(ctx context.Context, client *xmpp.Client, stanza xmpp.Stanza) error {
        message := stanza.(xmpp.Message)
        // inspect message extensions
        return nil
    },
    xmpp.HandlerOptions{Priority: 20},
)
```

Handlers run before convenience callbacks and named stanza events. A
synchronous long-running handler blocks stanza dispatch; use `Async` or start a
goroutine when ordering is not required.

## Matchers

Built-in matchers:

- `MatchAll`;
- `MatchKind("message")`;
- `MatchID(id)`;
- `MatchFrom(jid, bare)` and `MatchTo(jid, bare)`;
- `MatchPayload(namespace, local)`;
- `MatchAnd`, `MatchOr`, and `MatchNot`;
- `MatcherFunc` for custom logic.

Example:

```go
matcher := xmpp.MatchAnd(
    xmpp.MatchKind("iq"),
    xmpp.MatchPayload("urn:xmpp:ping", "ping"),
)
```

## Scoped internal APIs

`client.API` mirrors Slixmpp's replaceable plugin API concept. Handlers may be
registered globally or for a specific JID and/or node. Resolution order is:

1. exact JID and node;
2. JID only;
3. node only;
4. global.

```go
client.API.Register("my_plugin", "lookup", "service.example", "node", handler)
result, err := client.API.Run(ctx, xmpp.APICall{
    Category:  "my_plugin",
    Operation: "lookup",
    JID:       "service.example",
    Node:      "node",
    Args:      request,
})
```
