# Stateful XEPs

## Multi-User Chat (XEP-0045)

```go
plugin, _ := xep.Load(client, 45)
muc := plugin.(*xep.MUC)

state, err := muc.Join(ctx, "room@conference.example.org", "go-bot", xep.JoinOptions{
    Password: password,
    History:  &xep.History{MaxStanzas: 20},
})
```

The plugin tracks joined rooms and occupants, handles self-presence status
codes, emits join/leave/occupant events, sends room/private messages, changes
subjects/nicks, processes invitations, and exposes owner/admin IQ helpers.

## PubSub and PEP (XEP-0060/XEP-0163)

```go
plugin, _ := xep.Load(client, 60)
pubsub := plugin.(*xep.PubSub)

item, err := pubsub.Publish(ctx, service, node, "item-id", payload, nil)
items, err := pubsub.Items(ctx, service, node, 50, nil)
```

PEP is a thin specialization using the account's own service:

```go
plugin, _ := xep.Load(client, 163)
pep := plugin.(*xep.PEP)
_, err := pep.Publish(ctx, node, itemID, payload, options)
```

Incoming event messages emit `pubsub_event` and `pep_event`.

## Message Archive Management (XEP-0313)

```go
plugin, _ := xep.Load(client, 313)
mam := plugin.(*xep.MAM)

max := 100
page, err := mam.Query(ctx, "", xep.MAMQuery{
    With: "juliet@example.org",
    Start: startTime,
    RSM:   &xep.RSMSet{Max: &max},
})
```

Results are correlated by query ID. The returned page contains forwarded
messages, delays, completion state, and RSM cursors. The plugin also emits
`mam_result` for streamed results and supports MAM preferences.

## Blocking and carbons

```go
blockingPlugin, _ := xep.Load(client, 191)
blocking := blockingPlugin.(*xep.Blocking)
list, err := blocking.List(ctx)
err = blocking.Block(ctx, "spam@example.org")

carbonPlugin, _ := xep.Load(client, 280)
carbons := carbonPlugin.(*xep.Carbons)
err = carbons.Enable(ctx)
```

Carbon copies are parsed as forwarded stanzas and emitted through
`carbon_received` or `carbon_sent`.

## Stream management

The XEP-0198 plugin enables the stream, counts handled stanzas, responds to
acknowledgement requests, and retains the unacknowledged stanza queue:

```go
plugin, _ := xep.Load(client, 198)
sm := plugin.(*xep.StreamManagement)
state, err := sm.Enable(ctx, true)
```

Automatic transport reconnection/resumption is deliberately not hidden inside
the client. Persist `SMState`, reconnect under application policy, send the
resume node, and replay the returned unacknowledged queue only after successful
resumption.

## HTTP upload

```go
plugin, _ := xep.Load(client, 363)
upload := plugin.(*xep.HTTPUpload)
service, maxSize, err := upload.Discover(ctx, client.JID().Domain)
slot, err := upload.RequestSlot(ctx, service, "photo.jpg", "image/jpeg", size)
err = upload.Upload(ctx, slot, file, size, "image/jpeg")
```

HTTPS is required by default. Set `AllowInsecureHTTP` only in a local test
environment.
