# XEP support matrix

This file is generated from `xep.Catalog` by `go run ./cmd/xep-matrix`.

Coverage meanings:

- **operational** — typed implementation plus client behavior and protocol tests within the documented scope.
- **client** — typed client operations and/or incoming event handling; some optional portions of the XEP may remain.
- **payload** — typed XML builders/parsers; application code supplies the surrounding workflow.
- **catalog** — the Slixmpp plugin name exists for migration/discovery, but loading it advertises no wire support.

| XEP | Slixmpp plugin | Title | Coverage | Namespaces | Notes |
|---|---|---|---|---|---|
| XEP-0004 | `xep_0004` | Data Forms | **operational** | jabber:x:data | Typed forms, validation, and media elements. |
| XEP-0009 | `xep_0009` | Jabber-RPC | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0012 | `xep_0012` | Last Activity | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0013 | `xep_0013` | Flexible Offline Message Retrieval | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0016 | `xep_0016` | Privacy Lists | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0020 | `xep_0020` | Feature Negotiation | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0027 | `xep_0027` | Current Jabber OpenPGP Usage | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0030 | `xep_0030` | Service Discovery | **operational** | http://jabber.org/protocol/disco#info<br>http://jabber.org/protocol/disco#items | Query and responder APIs. |
| XEP-0033 | `xep_0033` | Extended Stanza Addressing | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0045 | `xep_0045` | Multi-User Chat | **client** | http://jabber.org/protocol/muc<br>http://jabber.org/protocol/muc#user<br>http://jabber.org/protocol/muc#admin<br>http://jabber.org/protocol/muc#owner | Join, leave, invitations, occupants, configuration, and room administration. |
| XEP-0047 | `xep_0047` | In-band Bytestreams | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0048 | `xep_0048` | Bookmarks | **client** | storage:bookmarks | Legacy conference bookmark read/write through private XML storage. |
| XEP-0049 | `xep_0049` | Private XML Storage | **client** | jabber:iq:private | Private XML storage get/set API. |
| XEP-0050 | `xep_0050` | Ad-Hoc Commands | **payload** | http://jabber.org/protocol/commands | Typed command actions, status, notes, and forms. |
| XEP-0054 | `xep_0054` | vcard-temp | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0055 | `xep_0055` | Jabber Search | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0059 | `xep_0059` | Result Set Management | **operational** | http://jabber.org/protocol/rsm | Typed request/response sets and paging helpers. |
| XEP-0060 | `xep_0060` | Publish-Subscribe | **client** | http://jabber.org/protocol/pubsub<br>http://jabber.org/protocol/pubsub#event<br>http://jabber.org/protocol/pubsub#owner | Publish, retract, subscribe, configuration, affiliations, and events. |
| XEP-0065 | `xep_0065` | SOCKS5 Bytestreams | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0066 | `xep_0066` | Out of Band Data | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0070 | `xep_0070` | Verifying HTTP Requests via XMPP | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0071 | `xep_0071` | XHTML-IM | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0077 | `xep_0077` | In-Band Registration | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0078 | `xep_0078` | Non-SASL Authentication | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0079 | `xep_0079` | Advanced Message Processing | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0080 | `xep_0080` | User Location | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0082 | `xep_0082` | XMPP Date and Time Profiles | **operational** | — | Date and time profile helpers. |
| XEP-0084 | `xep_0084` | User Avatar | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0085 | `xep_0085` | Chat State Notifications | **payload** | http://jabber.org/protocol/chatstates | Typed chat-state extensions. |
| XEP-0086 | `xep_0086` | Error Condition Mappings | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0091 | `xep_0091` | Legacy Delayed Delivery | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0092 | `xep_0092` | Software Version | **operational** | jabber:iq:version | Automatic responder and query API. |
| XEP-0095 | `xep_0095` | Stream Initiation | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0096 | `xep_0096` | SI File Transfer | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0100 | `xep_0100` | Gateway Interaction | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0106 | `xep_0106` | JID Escaping | **operational** | urn:xmpp:jid:0 | Implemented by xmpp.EscapeLocal and xmpp.UnescapeLocal. |
| XEP-0107 | `xep_0107` | User Mood | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0108 | `xep_0108` | User Activity | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0115 | `xep_0115` | Entity Capabilities | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0118 | `xep_0118` | User Tune | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0122 | `xep_0122` | Data Forms Validation | **payload** | http://jabber.org/protocol/xdata-validate | Data-form validation rules are integrated with XEP-0004 forms. |
| XEP-0128 | `xep_0128` | Service Discovery Extensions | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0131 | `xep_0131` | Stanza Headers and Internet Metadata | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0133 | `xep_0133` | Service Administration | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0152 | `xep_0152` | Reachability Addresses | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0153 | `xep_0153` | vCard-Based Avatars | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0163 | `xep_0163` | Personal Eventing Protocol | **client** | http://jabber.org/protocol/pubsub#event | PEP publish, item retrieval, and event handling over XEP-0060. |
| XEP-0172 | `xep_0172` | User Nickname | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0184 | `xep_0184` | Message Delivery Receipts | **client** | urn:xmpp:receipts | Request, acknowledge, and automatic receipt handling. |
| XEP-0186 | `xep_0186` | Invisible Command | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0191 | `xep_0191` | Blocking Command | **client** | urn:xmpp:blocking | List, block, unblock, and push events. |
| XEP-0196 | `xep_0196` | User Gaming | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0198 | `xep_0198` | Stream Management | **client** | urn:xmpp:sm:3 | Enable and acknowledgement tracking; reconnect/resume remains application-driven. |
| XEP-0199 | `xep_0199` | XMPP Ping | **operational** | urn:xmpp:ping | Automatic responder and latency query. |
| XEP-0202 | `xep_0202` | Entity Time | **operational** | urn:xmpp:time | Automatic responder and query API. |
| XEP-0203 | `xep_0203` | Delayed Delivery | **operational** | urn:xmpp:delay | Typed delayed-delivery marker. |
| XEP-0221 | `xep_0221` | Data Forms Media Element | **payload** | urn:xmpp:media-element | Media elements are integrated with XEP-0004 fields. |
| XEP-0222 | `xep_0222` | Persistent Storage of Public Data via PubSub | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0223 | `xep_0223` | Persistent Storage of Private Data via PubSub | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0224 | `xep_0224` | Attention | **payload** | urn:xmpp:attention:0 | Attention message payload helpers. |
| XEP-0231 | `xep_0231` | Bits of Binary | **payload** | urn:xmpp:bob | CID generation and Bits of Binary payloads. |
| XEP-0234 | `xep_0234` | Jingle File Transfer | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0235 | `xep_0235` | OAuth Over XMPP | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0242 | `xep_0242` | XMPP Client Compliance 2009 | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0249 | `xep_0249` | Direct MUC Invitations | **payload** | jabber:x:conference | Direct MUC invitation payload helpers. |
| XEP-0256 | `xep_0256` | Last Activity in Presence | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0257 | `xep_0257` | Client Certificate Management for SASL EXTERNAL | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0258 | `xep_0258` | Security Labels in XMPP | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0264 | `xep_0264` | Jingle Content Thumbnails | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0270 | `xep_0270` | XMPP Compliance Suites 2010 | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0279 | `xep_0279` | Server IP Check | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0280 | `xep_0280` | Message Carbons | **client** | urn:xmpp:carbons:2 | Enable, disable, and forwarded-carbon parsing. |
| XEP-0292 | `xep_0292` | vCard4 Over XMPP | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0297 | `xep_0297` | Stanza Forwarding | **payload** | urn:xmpp:forward:0 | Typed forwarded stanza container. |
| XEP-0300 | `xep_0300` | Use of Cryptographic Hash Functions in XMPP | **operational** | urn:xmpp:hashes:2 | Hash registry and payload helpers. |
| XEP-0302 | `xep_0302` | XMPP Compliance Suites 2012 | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0308 | `xep_0308` | Last Message Correction | **operational** | urn:xmpp:message-correct:0 | Message correction helpers. |
| XEP-0313 | `xep_0313` | Message Archive Management | **client** | urn:xmpp:mam:2 | Archive queries, result events, fin parsing, and RSM. |
| XEP-0317 | `xep_0317` | Hats | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0319 | `xep_0319` | Last User Interaction in Presence | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0323 | `xep_0323` | Internet of Things - Sensor Data | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0325 | `xep_0325` | Internet of Things - Control | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0332 | `xep_0332` | HTTP over XMPP Transport | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0333 | `xep_0333` | Chat Markers | **payload** | urn:xmpp:chat-markers:0 | Chat marker payloads. |
| XEP-0334 | `xep_0334` | Message Processing Hints | **operational** | urn:xmpp:hints | Message-processing hints. |
| XEP-0335 | `xep_0335` | JSON Containers | **operational** | urn:xmpp:json:0 | Validated JSON containers. |
| XEP-0352 | `xep_0352` | Client State Indication | **client** | urn:xmpp:csi:0 | Active/inactive stream elements. |
| XEP-0353 | `xep_0353` | Jingle Message Initiation | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0356 | `xep_0356` | Privileged Entity | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0359 | `xep_0359` | Unique and Stable Stanza IDs | **operational** | urn:xmpp:sid:0 | Origin and stanza identifiers. |
| XEP-0363 | `xep_0363` | HTTP File Upload | **client** | urn:xmpp:http:upload:0 | Service discovery, slot requests, and HTTP upload. |
| XEP-0369 | `xep_0369` | MIX-CORE | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0372 | `xep_0372` | References | **payload** | urn:xmpp:reference:0 | Typed references. |
| XEP-0377 | `xep_0377` | Spam Reporting | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0380 | `xep_0380` | Explicit Message Encryption | **operational** | urn:xmpp:eme:0 | Encryption metadata. |
| XEP-0382 | `xep_0382` | Spoiler Messages | **operational** | urn:xmpp:spoiler:0 | Spoiler marker and localized hint. |
| XEP-0385 | `xep_0385` | Stateless Inline Media Sharing | **payload** | urn:xmpp:sims:1 | SIMS references and sources. |
| XEP-0394 | `xep_0394` | Message Markup | **payload** | urn:xmpp:markup:0 | Message markup spans, blocks, lists, and code ranges. |
| XEP-0402 | `xep_0402` | PEP Native Bookmarks | **client** | urn:xmpp:bookmarks:1 | PEP bookmark helpers. |
| XEP-0403 | `xep_0403` | MIX-Presence | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0404 | `xep_0404` | MIX-ANON | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0405 | `xep_0405` | MIX-PAM | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0410 | `xep_0410` | MUC Self-Ping | **client** | urn:xmpp:ping | MUC self-ping. |
| XEP-0421 | `xep_0421` | Anonymous Unique Occupant Identifiers for MUCs | **operational** | urn:xmpp:occupant-id:0 | Occupant identifiers. |
| XEP-0422 | `xep_0422` | Message Fastening | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0424 | `xep_0424` | Message Retraction | **operational** | urn:xmpp:message-retract:1 | Message retraction and fallback. |
| XEP-0425 | `xep_0425` | Moderated Message Retraction | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0428 | `xep_0428` | Fallback Indication | **operational** | urn:xmpp:fallback:0 | Fallback ranges and stripping. |
| XEP-0437 | `xep_0437` | Room Activity Indicators | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0439 | `xep_0439` | Quick Response | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0441 | `xep_0441` | Message Archive Management Preferences | **client** | urn:xmpp:mam:2 | MAM preference get/set payloads and client API. |
| XEP-0444 | `xep_0444` | Message Reactions | **operational** | urn:xmpp:reactions:0 | Message reactions. |
| XEP-0446 | `xep_0446` | File Metadata Element | **payload** | urn:xmpp:file:metadata:0 | File metadata, hashes, dimensions, and thumbnails. |
| XEP-0447 | `xep_0447` | Stateless File Sharing | **payload** | urn:xmpp:sfs:0 | Stateless file sharing. |
| XEP-0449 | `xep_0449` | Stickers | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0454 | `xep_0454` | OMEMO Media Sharing | **payload** | urn:xmpp:omemo:2 | OMEMO media fragment helpers. |
| XEP-0455 | `xep_0455` | Service Outage Status | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0461 | `xep_0461` | Message Replies | **operational** | urn:xmpp:reply:0 | Message replies and fallback. |
| XEP-0462 | `xep_0462` | PubSub Type Filtering | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0463 | `xep_0463` | MUC Affiliation Versioning | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0469 | `xep_0469` | Bookmark Pinning | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0482 | `xep_0482` | Call Invites | **payload** | urn:xmpp:call-invites:0 | Call invitation actions. |
| XEP-0490 | `xep_0490` | Message Displayed Synchronization | **payload** | urn:xmpp:mds:displayed:0 | Displayed synchronization. |
| XEP-0492 | `xep_0492` | Chat Notification Settings | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0494 | `xep_0494` | Client Access Management | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0502 | `xep_0502` | MUC Activity Indicator | **catalog** | — | Metadata compatibility only; use `xmpp.Node` for custom XML. |
| XEP-0511 | `xep_0511` | Link Metadata | **payload** | urn:xmpp:link-metadata:0 | Link metadata. |
| XEP-0513 | `xep_0513` | Explicit Mentions | **operational** | urn:xmpp:mention:0 | Explicit mentions. |
