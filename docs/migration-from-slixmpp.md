# Migration from Slixmpp

## Concept map

| Slixmpp | slixmpp-go |
|---|---|
| `ClientXMPP` | `*xmpp.Client` |
| `ComponentXMPP` | `*xmpp.Client` with `Config.ComponentSecret` |
| `register_plugin('xep_0030')` | `xep.RegisterAll(client)` then `xep.Load(client, 30)` |
| event handler | `client.Events.On` |
| stanza handler/matcher | `client.AddHandler` plus `xmpp.Matcher` |
| coroutine/asyncio task | goroutine plus `context.Context` |
| stanza plugin dictionary | typed struct or `xmpp.Node` |
| `Iq.send()` | `client.RequestIQ(ctx, iq)` |
| plugin API | `client.API` / `xmpp.APIProxy` |
| JID object | `xmpp.JID` |

## Connecting

Python:

```python
xmpp = ClientXMPP(jid, password)
xmpp.register_plugin('xep_0030')
xmpp.connect()
xmpp.process()
```

Go:

```go
client := xmpp.MustNewClient(xmpp.DefaultConfig(jid, password))
if err := xep.RegisterAll(client); err != nil { return err }
if _, err := xep.Load(client, 30); err != nil { return err }
if err := client.Connect(ctx); err != nil { return err }
client.Wait()
```

## Events

Python:

```python
xmpp.add_event_handler('message', self.message)
```

Go:

```go
client.Events.On("message", func(ctx context.Context, event xmpp.Event) error {
    message := event.Data.(xmpp.Message)
    return nil
})
```

Small programs can use `client.OnMessage`.

## Stanza access

Python's dictionary-like syntax is replaced with compile-time fields:

```python
body = msg['body']
```

```go
body := message.Body
```

Unknown or extension XML remains available through `xmpp.Node`:

```go
extension := message.Extension(namespace, local)
value, _ := extension.AttrValue("name")
```

## Plugin names

The complete Slixmpp 1.17 plugin-name catalog is present, so configuration can
continue to refer to `xep_0030`, `xep_0045`, and similar names. Check
`docs/xep-support.md` before assuming behavioral parity. Catalog-only plugins
are metadata placeholders and advertise no protocol feature.

## Exceptions and cancellation

Python exceptions become Go errors. Timeouts and cancellation are explicit:

```go
ctx, cancel := context.WithTimeout(parent, 15*time.Second)
defer cancel()
response, err := client.RequestIQ(ctx, request)
```

Use `errors.As(err, *xmpp.IQResponseError)` to inspect IQ failures.

## OMEMO migration

`slixmpp-omemo` delegates cryptographic sessions to python-omemo backends. The
Go design preserves that boundary:

| Python concept | Go concept |
|---|---|
| OMEMO storage | `omemo.Store` |
| backend | `omemo.SessionBackend` |
| trust levels | `omemo.TrustLevel` |
| trust policy | `omemo.TrustPolicy` |
| plugin/session object | `*omemo.Manager` |

Do not migrate by using `omemo/testkit`; it is not interoperable. Implement or
bind a production Signal backend first, then migrate device/trust metadata into
a `Store` implementation.
