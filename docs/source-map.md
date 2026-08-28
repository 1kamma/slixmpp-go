# Source and API map

This repository is an independent Go implementation. It does not embed Python
source files, but its package boundaries deliberately cover the same working
areas as Slixmpp and slixmpp-omemo so migrations have an obvious destination.

| Slixmpp / slixmpp-omemo area | Go destination | Notes |
|---|---|---|
| `slixmpp.jid` | `xmpp.JID` | Parsing, bare/full forms, resource handling, and XEP-0106 escaping. |
| `slixmpp.xmlstream` | `xmpp.Node`, `xmpp.Client` | Namespace-aware XML, stream negotiation, serialized writes, receive loop, and dispatch. |
| `slixmpp.stanza` | `xmpp.Message`, `xmpp.Presence`, `xmpp.IQ`, `xmpp.StanzaError` | Typed core stanzas plus lossless extension nodes. |
| `BaseXMPP` event system | `xmpp.EventBus`, `xmpp.Handler`, `xmpp.Matcher` | Priorities, once/async handlers, wildcard events, and typed stanza matchers. |
| `ClientXMPP` | `xmpp.Client` | TCP, SRV, STARTTLS/direct TLS, SASL, binding, initial presence, IQ correlation, and roster APIs. |
| `ComponentXMPP` | `xmpp.Client` with `Config.ComponentSecret` | XEP-0114 external-component handshake through the same stanza/plugin surface. |
| Slixmpp plugin manager | `xmpp.PluginManager`, `xep.RegisterAll`, `xep.Load` | Dependency loading and familiar `xep_NNNN` names. |
| Slixmpp plugin API registry | `xmpp.APIRegistry`, `xmpp.APIProxy` | Global, JID, node, and exact JID+node operation scopes. |
| `slixmpp.plugins.xep_*` | `xep` | Typed implementations where listed in `xep-support.md`; metadata-only registrations otherwise. |
| Slixmpp data forms and stanza plugins | `xep.DataForm`, payload structs, `xmpp.Node` | Compile-time types for implemented protocols; generic XML remains available for every extension. |
| `slixmpp-omemo` plugin/session integration | `omemo.Manager` | Device lists, bundles, PEP publication, trust policy, encrypted envelopes, and message events. |
| `slixmpp-omemo` storage hooks | `omemo.Store`, `omemo.MemoryStore`, `omemo.JSONStore` | Pluggable persistence with an atomic mode-0600 JSON implementation. |
| python-omemo backend dependency | `omemo.SessionBackend` | Production Signal/X3DH/Double-Ratchet cryptography is intentionally supplied behind this interface. |
| slixmpp-omemo trust levels | `omemo.TrustLevel`, `omemo.TrustPolicy` | Explicit trusted, undecided, distrusted, and verified policy decisions. |

## Behavioral coverage

The package map is complete, but protocol behavior is intentionally described
per XEP rather than with a blanket parity claim. See
[`xep-support.md`](xep-support.md):

- **operational** entries have typed protocol behavior and tests;
- **client** entries have stateful request/response/event support with stated
  limits;
- **payload** entries provide typed XML builders/parsers;
- **catalog** entries preserve the Slixmpp plugin identity only and advertise no
  feature.
