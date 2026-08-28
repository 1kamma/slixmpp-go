# Roster and messaging

## Roster

The core client owns `client.Roster`, which implements RFC 6121 roster
management and subscription presence.

Fetch and cache:

```go
items, err := client.Roster.Get(ctx)
if err != nil {
    return err
}
```

Create or update:

```go
err := client.Roster.Set(ctx, xmpp.RosterItem{
    JID:    "juliet@example.org",
    Name:   "Juliet",
    Groups: []string{"Friends"},
})
```

Remove:

```go
err := client.Roster.Remove(ctx, "juliet@example.org")
```

Subscription helpers:

```go
client.Roster.Subscribe("juliet@example.org", "Please add me")
client.Roster.AcceptSubscription("juliet@example.org")
client.Roster.Unsubscribe("juliet@example.org")
```

Incoming roster pushes are acknowledged, merged into the cache, and emitted as
`roster_update` events.

## Messages

```go
message := xmpp.Message{
    To:   "juliet@example.org",
    ID:   client.NextID(),
    Type: xmpp.MessageChat,
    Body: "Wherefore art thou?",
}
err := client.Send(message)
```

`Message` supports localized bodies/subjects, threads, stanza errors, and an
ordered list of extension nodes.

```go
receipt := message.Extension("urn:xmpp:receipts", "received")
```

For common extensions, prefer typed helpers in `xep`: receipts, chat states,
corrections, markers, hints, stanza IDs, references, EME, spoilers, retractions,
fallbacks, reactions, replies, and mentions.

## Presence

```go
priority := 5
err := client.SendPresence(xmpp.Presence{
    Show:     "away",
    Status:   "back soon",
    Priority: &priority,
})
```

Priority must be in the RFC range -128 through 127.

## IQ requests

```go
query := xmpp.NewNode("urn:example:query", "query")
response, err := client.RequestIQ(ctx, xmpp.IQ{
    To:       "service.example.org",
    Type:     xmpp.IQGet,
    Payloads: []xmpp.Node{query},
})
```

The client assigns an ID when absent, correlates results/errors, applies the IQ
timeout, and returns `*xmpp.IQResponseError` for error responses.
