# Plugin architecture

## Lifecycle

A plugin implements:

```go
type Plugin interface {
    Name() string
    Description() string
    Dependencies() []string
    Features() []string
    Init(*Client) error
    Shutdown(context.Context) error
}
```

Dependencies are loaded before `Init`. Cycles are rejected. Shutdown runs in
reverse plugin order with a bounded context.

## Registering the Slixmpp catalog

```go
if err := xep.RegisterAll(client); err != nil {
    return err
}
```

Every documented Slixmpp 1.17 `xep_NNNN` name gets a factory. Implemented
entries construct a typed plugin. Catalog-only entries construct
`xep.GenericPlugin` and advertise no features.

This distinction matters: service discovery must never claim support merely
because a migration-compatible name exists.

## Loading and retrieving plugins

```go
plugin, err := client.Plugins.Load("xep_0030")
disco := plugin.(*xep.Disco)

same, ok := client.Plugins.Get("xep_0030")
```

Equivalent numeric helper:

```go
plugin, err := xep.Load(client, 30)
```

## Supplying configured plugins

Factories create sensible defaults. To configure a plugin instance directly,
register its dependencies and call `Use`:

```go
if err := xep.RegisterAll(client); err != nil {
    return err
}

version := xep.NewVersion("my-bot", buildVersion, runtime.GOOS)
if err := client.Use(version); err != nil {
    return err
}
```

Calling `Use` twice for the same plugin name is idempotent.

## Writing a plugin

```go
type Example struct {
    client  *xmpp.Client
    handler *xmpp.Handler
}

func (*Example) Name() string           { return "example" }
func (*Example) Description() string    { return "Example extension" }
func (*Example) Dependencies() []string { return []string{"xep_0030"} }
func (*Example) Features() []string     { return []string{"urn:example:0"} }

func (p *Example) Init(client *xmpp.Client) error {
    p.client = client
    p.handler = client.AddHandler(
        "example",
        xmpp.MatchPayload("urn:example:0", "notice"),
        p.handle,
    )
    return nil
}

func (p *Example) Shutdown(context.Context) error {
    p.client.RemoveHandler(p.handler)
    return nil
}
```

A plugin that adds a service-discovery feature should only return it from
`Features` when the advertised behavior is actually implemented.

## Generating the support matrix

```bash
go run ./cmd/xep-matrix > docs/xep-support.md
```

Coverage metadata lives beside the runtime catalog so documentation and plugin
registration cannot silently drift apart.
