# Architecture

## Layers

### `xmpp`: stream kernel

The core package owns connection state and protocol framing:

1. resolve an endpoint;
2. establish TCP or direct TLS;
3. open the XML stream;
4. negotiate STARTTLS when applicable;
5. select and execute SASL;
6. bind a resource;
7. start the receive loop;
8. send initial presence;
9. dispatch stanzas and stream elements.

Writes are serialized. IQ requests use a concurrency-safe ID/channel map.
Incoming XML is decoded into `xmpp.Node`, then converted to typed stanza values.
Unknown extensions remain lossless generic nodes.

### Events and handlers

Stanza handlers are optimized for protocol modules: a matcher receives a stanza
before convenience callbacks. The event bus is optimized for application-level
fan-out and plugin notifications.

### Plugins

Plugins register handlers/events/APIs during `Init` and remove them during
`Shutdown`. Dependencies are resolved by name. `Features` is the source for
root service-discovery advertising.

### `xep`: protocol modules

Typed payload modules avoid connection state when possible. Stateful modules
store a client pointer, register handlers, correlate requests, maintain caches,
and emit semantic events.

### `omemo`: integration and crypto boundary

OMEMO is split into four contracts:

- XMPP/PEP protocol integration (`Manager`);
- durable metadata/trust storage (`Store`);
- Signal session and key wrapping (`SessionBackend`);
- application policy (`TrustPolicy`).

This prevents the XMPP layer from silently substituting a home-grown ratchet for
a security-reviewed Signal implementation.

## Concurrency

- Client writes are protected by one mutex.
- Connection state, pending IQs, handlers, events, plugins, roster, and each
  stateful XEP cache have independent locks.
- Handlers run synchronously unless `Async` is selected.
- Plugin background work is tied to client events or call contexts.
- The race test suite covers all packages.

## Error model

- Stream/transport failures are returned or emitted as `error` events.
- IQ error responses become `*xmpp.IQResponseError` and retain the full IQ.
- Stanza errors are structured as `xmpp.StanzaError`.
- Plugin parse failures wrap protocol context.
- Context cancellation controls all request/response operations.

## Extensibility

A missing XEP does not require a core fork. Applications can:

1. construct arbitrary `xmpp.Node` payloads;
2. add a stanza matcher;
3. package the behavior as `xmpp.Plugin`;
4. advertise a feature only after behavior is implemented;
5. optionally contribute a typed entry and coverage metadata to `xep`.
