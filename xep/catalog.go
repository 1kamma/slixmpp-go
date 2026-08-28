// Package xep contains typed XMPP Extension Protocol payloads and plugins.
package xep

import (
	"fmt"
	"sort"
)

// Coverage describes implementation depth.
type Coverage string

const (
	CoverageCatalog     Coverage = "catalog"
	CoveragePayload     Coverage = "payload"
	CoverageClient      Coverage = "client"
	CoverageOperational Coverage = "operational"
)

// Descriptor is one Slixmpp-compatible XEP plugin entry.
type Descriptor struct {
	Number     int
	Title      string
	Namespaces []string
	Coverage   Coverage
	Notes      string
}

func (d Descriptor) Name() string { return fmt.Sprintf("xep_%04d", d.Number) }
func (d Descriptor) XEP() string  { return fmt.Sprintf("XEP-%04d", d.Number) }

// Catalog mirrors the Slixmpp 1.17 plugin index. Catalog-only entries retain
// the familiar plugin name but advertise no service-discovery feature.
var Catalog = []Descriptor{
	{4, "Data Forms", nil, "", ""}, {9, "Jabber-RPC", nil, "", ""}, {12, "Last Activity", nil, "", ""}, {13, "Flexible Offline Message Retrieval", nil, "", ""}, {16, "Privacy Lists", nil, "", ""}, {20, "Feature Negotiation", nil, "", ""}, {27, "Current Jabber OpenPGP Usage", nil, "", ""}, {30, "Service Discovery", nil, "", ""}, {33, "Extended Stanza Addressing", nil, "", ""}, {45, "Multi-User Chat", nil, "", ""}, {47, "In-band Bytestreams", nil, "", ""}, {48, "Bookmarks", nil, "", ""}, {49, "Private XML Storage", nil, "", ""}, {50, "Ad-Hoc Commands", nil, "", ""}, {54, "vcard-temp", nil, "", ""}, {55, "Jabber Search", nil, "", ""}, {59, "Result Set Management", nil, "", ""}, {60, "Publish-Subscribe", nil, "", ""}, {65, "SOCKS5 Bytestreams", nil, "", ""}, {66, "Out of Band Data", nil, "", ""}, {70, "Verifying HTTP Requests via XMPP", nil, "", ""}, {71, "XHTML-IM", nil, "", ""}, {77, "In-Band Registration", nil, "", ""}, {78, "Non-SASL Authentication", nil, "", ""}, {79, "Advanced Message Processing", nil, "", ""}, {80, "User Location", nil, "", ""}, {82, "XMPP Date and Time Profiles", nil, "", ""}, {84, "User Avatar", nil, "", ""}, {85, "Chat State Notifications", nil, "", ""}, {86, "Error Condition Mappings", nil, "", ""}, {91, "Legacy Delayed Delivery", nil, "", ""}, {92, "Software Version", nil, "", ""}, {95, "Stream Initiation", nil, "", ""}, {96, "SI File Transfer", nil, "", ""}, {100, "Gateway Interaction", nil, "", ""}, {106, "JID Escaping", nil, "", ""}, {107, "User Mood", nil, "", ""}, {108, "User Activity", nil, "", ""}, {115, "Entity Capabilities", nil, "", ""}, {118, "User Tune", nil, "", ""}, {122, "Data Forms Validation", nil, "", ""}, {128, "Service Discovery Extensions", nil, "", ""}, {131, "Stanza Headers and Internet Metadata", nil, "", ""}, {133, "Service Administration", nil, "", ""}, {152, "Reachability Addresses", nil, "", ""}, {153, "vCard-Based Avatars", nil, "", ""}, {163, "Personal Eventing Protocol", nil, "", ""}, {172, "User Nickname", nil, "", ""}, {184, "Message Delivery Receipts", nil, "", ""}, {186, "Invisible Command", nil, "", ""}, {191, "Blocking Command", nil, "", ""}, {196, "User Gaming", nil, "", ""}, {198, "Stream Management", nil, "", ""}, {199, "XMPP Ping", nil, "", ""}, {202, "Entity Time", nil, "", ""}, {203, "Delayed Delivery", nil, "", ""}, {221, "Data Forms Media Element", nil, "", ""}, {222, "Persistent Storage of Public Data via PubSub", nil, "", ""}, {223, "Persistent Storage of Private Data via PubSub", nil, "", ""}, {224, "Attention", nil, "", ""}, {231, "Bits of Binary", nil, "", ""}, {234, "Jingle File Transfer", nil, "", ""}, {235, "OAuth Over XMPP", nil, "", ""}, {242, "XMPP Client Compliance 2009", nil, "", ""}, {249, "Direct MUC Invitations", nil, "", ""}, {256, "Last Activity in Presence", nil, "", ""}, {257, "Client Certificate Management for SASL EXTERNAL", nil, "", ""}, {258, "Security Labels in XMPP", nil, "", ""}, {264, "Jingle Content Thumbnails", nil, "", ""}, {270, "XMPP Compliance Suites 2010", nil, "", ""}, {279, "Server IP Check", nil, "", ""}, {280, "Message Carbons", nil, "", ""}, {292, "vCard4 Over XMPP", nil, "", ""}, {297, "Stanza Forwarding", nil, "", ""}, {300, "Use of Cryptographic Hash Functions in XMPP", nil, "", ""}, {302, "XMPP Compliance Suites 2012", nil, "", ""}, {308, "Last Message Correction", nil, "", ""}, {313, "Message Archive Management", nil, "", ""}, {317, "Hats", nil, "", ""}, {319, "Last User Interaction in Presence", nil, "", ""}, {323, "Internet of Things - Sensor Data", nil, "", ""}, {325, "Internet of Things - Control", nil, "", ""}, {332, "HTTP over XMPP Transport", nil, "", ""}, {333, "Chat Markers", nil, "", ""}, {334, "Message Processing Hints", nil, "", ""}, {335, "JSON Containers", nil, "", ""}, {352, "Client State Indication", nil, "", ""}, {353, "Jingle Message Initiation", nil, "", ""}, {356, "Privileged Entity", nil, "", ""}, {359, "Unique and Stable Stanza IDs", nil, "", ""}, {363, "HTTP File Upload", nil, "", ""}, {369, "MIX-CORE", nil, "", ""}, {372, "References", nil, "", ""}, {377, "Spam Reporting", nil, "", ""}, {380, "Explicit Message Encryption", nil, "", ""}, {382, "Spoiler Messages", nil, "", ""}, {385, "Stateless Inline Media Sharing", nil, "", ""}, {394, "Message Markup", nil, "", ""}, {402, "PEP Native Bookmarks", nil, "", ""}, {403, "MIX-Presence", nil, "", ""}, {404, "MIX-ANON", nil, "", ""}, {405, "MIX-PAM", nil, "", ""}, {410, "MUC Self-Ping", nil, "", ""}, {421, "Anonymous Unique Occupant Identifiers for MUCs", nil, "", ""}, {422, "Message Fastening", nil, "", ""}, {424, "Message Retraction", nil, "", ""}, {425, "Moderated Message Retraction", nil, "", ""}, {428, "Fallback Indication", nil, "", ""}, {437, "Room Activity Indicators", nil, "", ""}, {439, "Quick Response", nil, "", ""}, {441, "Message Archive Management Preferences", nil, "", ""}, {444, "Message Reactions", nil, "", ""}, {446, "File Metadata Element", nil, "", ""}, {447, "Stateless File Sharing", nil, "", ""}, {449, "Stickers", nil, "", ""}, {454, "OMEMO Media Sharing", nil, "", ""}, {455, "Service Outage Status", nil, "", ""}, {461, "Message Replies", nil, "", ""}, {462, "PubSub Type Filtering", nil, "", ""}, {463, "MUC Affiliation Versioning", nil, "", ""}, {469, "Bookmark Pinning", nil, "", ""}, {482, "Call Invites", nil, "", ""}, {490, "Message Displayed Synchronization", nil, "", ""}, {492, "Chat Notification Settings", nil, "", ""}, {494, "Client Access Management", nil, "", ""}, {502, "MUC Activity Indicator", nil, "", ""}, {511, "Link Metadata", nil, "", ""}, {513, "Explicit Mentions", nil, "", ""},
}
var byNumber map[int]int

func init() {
	sort.Slice(Catalog, func(i, j int) bool { return Catalog[i].Number < Catalog[j].Number })
	byNumber = make(map[int]int, len(Catalog))
	for i := range Catalog {
		Catalog[i].Coverage = CoverageCatalog
		byNumber[Catalog[i].Number] = i
	}
	applyCoverageMetadata()
}
func Lookup(number int) (Descriptor, bool) {
	i, ok := byNumber[number]
	if !ok {
		return Descriptor{}, false
	}
	d := Catalog[i]
	d.Namespaces = append([]string(nil), d.Namespaces...)
	return d, true
}
func upgrade(number int, coverage Coverage, namespaces []string, notes string) {
	i, ok := byNumber[number]
	if !ok {
		return
	}
	Catalog[i].Coverage = coverage
	Catalog[i].Namespaces = append([]string(nil), namespaces...)
	Catalog[i].Notes = notes
}
func applyCoverageMetadata() {
	upgrade(48, CoverageClient, []string{"storage:bookmarks"}, "Legacy conference bookmark read/write through private XML storage.")
	upgrade(49, CoverageClient, []string{"jabber:iq:private"}, "Private XML storage get/set API.")
	upgrade(4, CoverageOperational, []string{"jabber:x:data"}, "Typed forms, validation, and media elements.")
	upgrade(30, CoverageOperational, []string{"http://jabber.org/protocol/disco#info", "http://jabber.org/protocol/disco#items"}, "Query and responder APIs.")
	upgrade(45, CoverageClient, []string{"http://jabber.org/protocol/muc", "http://jabber.org/protocol/muc#user", "http://jabber.org/protocol/muc#admin", "http://jabber.org/protocol/muc#owner"}, "Join, leave, invitations, occupants, configuration, and room administration.")
	upgrade(50, CoveragePayload, []string{"http://jabber.org/protocol/commands"}, "Typed command actions, status, notes, and forms.")
	upgrade(59, CoverageOperational, []string{"http://jabber.org/protocol/rsm"}, "Typed request/response sets and paging helpers.")
	upgrade(60, CoverageClient, []string{"http://jabber.org/protocol/pubsub", "http://jabber.org/protocol/pubsub#event", "http://jabber.org/protocol/pubsub#owner"}, "Publish, retract, subscribe, configuration, affiliations, and events.")
	upgrade(82, CoverageOperational, nil, "Date and time profile helpers.")
	upgrade(122, CoveragePayload, []string{"http://jabber.org/protocol/xdata-validate"}, "Data-form validation rules are integrated with XEP-0004 forms.")
	upgrade(163, CoverageClient, []string{"http://jabber.org/protocol/pubsub#event"}, "PEP publish, item retrieval, and event handling over XEP-0060.")
	upgrade(85, CoveragePayload, []string{"http://jabber.org/protocol/chatstates"}, "Typed chat-state extensions.")
	upgrade(92, CoverageOperational, []string{"jabber:iq:version"}, "Automatic responder and query API.")
	upgrade(106, CoverageOperational, []string{"urn:xmpp:jid:0"}, "Implemented by xmpp.EscapeLocal and xmpp.UnescapeLocal.")
	upgrade(184, CoverageClient, []string{"urn:xmpp:receipts"}, "Request, acknowledge, and automatic receipt handling.")
	upgrade(191, CoverageClient, []string{"urn:xmpp:blocking"}, "List, block, unblock, and push events.")
	upgrade(198, CoverageClient, []string{"urn:xmpp:sm:3"}, "Enable and acknowledgement tracking; reconnect/resume remains application-driven.")
	upgrade(199, CoverageOperational, []string{"urn:xmpp:ping"}, "Automatic responder and latency query.")
	upgrade(202, CoverageOperational, []string{"urn:xmpp:time"}, "Automatic responder and query API.")
	upgrade(203, CoverageOperational, []string{"urn:xmpp:delay"}, "Typed delayed-delivery marker.")
	upgrade(221, CoveragePayload, []string{"urn:xmpp:media-element"}, "Media elements are integrated with XEP-0004 fields.")
	upgrade(224, CoveragePayload, []string{"urn:xmpp:attention:0"}, "Attention message payload helpers.")
	upgrade(249, CoveragePayload, []string{"jabber:x:conference"}, "Direct MUC invitation payload helpers.")
	upgrade(231, CoveragePayload, []string{"urn:xmpp:bob"}, "CID generation and Bits of Binary payloads.")
	upgrade(280, CoverageClient, []string{"urn:xmpp:carbons:2"}, "Enable, disable, and forwarded-carbon parsing.")
	upgrade(297, CoveragePayload, []string{"urn:xmpp:forward:0"}, "Typed forwarded stanza container.")
	upgrade(300, CoverageOperational, []string{"urn:xmpp:hashes:2"}, "Hash registry and payload helpers.")
	upgrade(308, CoverageOperational, []string{"urn:xmpp:message-correct:0"}, "Message correction helpers.")
	upgrade(313, CoverageClient, []string{"urn:xmpp:mam:2"}, "Archive queries, result events, fin parsing, and RSM.")
	upgrade(333, CoveragePayload, []string{"urn:xmpp:chat-markers:0"}, "Chat marker payloads.")
	upgrade(334, CoverageOperational, []string{"urn:xmpp:hints"}, "Message-processing hints.")
	upgrade(335, CoverageOperational, []string{"urn:xmpp:json:0"}, "Validated JSON containers.")
	upgrade(352, CoverageClient, []string{"urn:xmpp:csi:0"}, "Active/inactive stream elements.")
	upgrade(359, CoverageOperational, []string{"urn:xmpp:sid:0"}, "Origin and stanza identifiers.")
	upgrade(363, CoverageClient, []string{"urn:xmpp:http:upload:0"}, "Service discovery, slot requests, and HTTP upload.")
	upgrade(372, CoveragePayload, []string{"urn:xmpp:reference:0"}, "Typed references.")
	upgrade(380, CoverageOperational, []string{"urn:xmpp:eme:0"}, "Encryption metadata.")
	upgrade(382, CoverageOperational, []string{"urn:xmpp:spoiler:0"}, "Spoiler marker and localized hint.")
	upgrade(385, CoveragePayload, []string{"urn:xmpp:sims:1"}, "SIMS references and sources.")
	upgrade(394, CoveragePayload, []string{"urn:xmpp:markup:0"}, "Message markup spans, blocks, lists, and code ranges.")
	upgrade(402, CoverageClient, []string{"urn:xmpp:bookmarks:1"}, "PEP bookmark helpers.")
	upgrade(410, CoverageClient, []string{"urn:xmpp:ping"}, "MUC self-ping.")
	upgrade(421, CoverageOperational, []string{"urn:xmpp:occupant-id:0"}, "Occupant identifiers.")
	upgrade(424, CoverageOperational, []string{"urn:xmpp:message-retract:1"}, "Message retraction and fallback.")
	upgrade(428, CoverageOperational, []string{"urn:xmpp:fallback:0"}, "Fallback ranges and stripping.")
	upgrade(441, CoverageClient, []string{"urn:xmpp:mam:2"}, "MAM preference get/set payloads and client API.")
	upgrade(444, CoverageOperational, []string{"urn:xmpp:reactions:0"}, "Message reactions.")
	upgrade(446, CoveragePayload, []string{"urn:xmpp:file:metadata:0"}, "File metadata, hashes, dimensions, and thumbnails.")
	upgrade(447, CoveragePayload, []string{"urn:xmpp:sfs:0"}, "Stateless file sharing.")
	upgrade(454, CoveragePayload, []string{"urn:xmpp:omemo:2"}, "OMEMO media fragment helpers.")
	upgrade(461, CoverageOperational, []string{"urn:xmpp:reply:0"}, "Message replies and fallback.")
	upgrade(482, CoveragePayload, []string{"urn:xmpp:call-invites:0"}, "Call invitation actions.")
	upgrade(490, CoveragePayload, []string{"urn:xmpp:mds:displayed:0"}, "Displayed synchronization.")
	upgrade(511, CoveragePayload, []string{"urn:xmpp:link-metadata:0"}, "Link metadata.")
	upgrade(513, CoverageOperational, []string{"urn:xmpp:mention:0"}, "Explicit mentions.")
}
