package xmpp

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidJID is returned when a Jabber ID cannot be parsed.
var ErrInvalidJID = errors.New("xmpp: invalid JID")

// JID is a Jabber ID split into localpart, domainpart, and resourcepart.
type JID struct {
	Local    string
	Domain   string
	Resource string
}

// ParseJID parses a domain, bare, or full JID. It performs structural
// validation and ASCII case folding of the domain. Full PRECIS/IDNA processing
// is intentionally left to applications that require it.
func ParseJID(value string) (JID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return JID{}, ErrInvalidJID
	}
	var resource string
	if slash := strings.IndexByte(value, '/'); slash >= 0 {
		resource = value[slash+1:]
		value = value[:slash]
		if resource == "" {
			return JID{}, fmt.Errorf("%w: empty resource", ErrInvalidJID)
		}
	}
	var local, domain string
	if at := strings.LastIndexByte(value, '@'); at >= 0 {
		local, domain = value[:at], value[at+1:]
		if local == "" {
			return JID{}, fmt.Errorf("%w: empty localpart", ErrInvalidJID)
		}
	} else {
		domain = value
	}
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	if domain == "" || strings.ContainsAny(domain, "@/ \t\r\n") {
		return JID{}, fmt.Errorf("%w: invalid domainpart", ErrInvalidJID)
	}
	if strings.ContainsAny(local, "@/\x00\r\n") || strings.ContainsAny(resource, "\x00\r\n") {
		return JID{}, ErrInvalidJID
	}
	return JID{Local: local, Domain: domain, Resource: resource}, nil
}

// MustParseJID parses value and panics on failure.
func MustParseJID(value string) JID {
	jid, err := ParseJID(value)
	if err != nil {
		panic(err)
	}
	return jid
}

// String returns the domain, bare, or full JID.
func (j JID) String() string {
	base := j.Domain
	if j.Local != "" {
		base = j.Local + "@" + base
	}
	if j.Resource != "" {
		base += "/" + j.Resource
	}
	return base
}

// Bare returns the JID without its resourcepart.
func (j JID) Bare() string {
	if j.Local != "" {
		return j.Local + "@" + j.Domain
	}
	return j.Domain
}

// IsBare reports whether the JID has no resourcepart.
func (j JID) IsBare() bool { return j.Resource == "" }

// IsDomain reports whether the JID contains only a domainpart.
func (j JID) IsDomain() bool { return j.Local == "" && j.Resource == "" }

// WithResource returns a copy of j using resource.
func (j JID) WithResource(resource string) JID { j.Resource = resource; return j }

// EqualBare reports whether two JIDs have the same localpart and domainpart.
func (j JID) EqualBare(other JID) bool {
	return j.Local == other.Local && strings.EqualFold(j.Domain, other.Domain)
}

var escapeLocalReplacer = strings.NewReplacer(
	"\\", "\\5c", " ", "\\20", `"`, "\\22", "&", "\\26", "'", "\\27",
	"/", "\\2f", ":", "\\3a", "<", "\\3c", ">", "\\3e", "@", "\\40",
)
var unescapeLocalReplacer = strings.NewReplacer(
	"\\20", " ", "\\22", `"`, "\\26", "&", "\\27", "'", "\\2f", "/",
	"\\3a", ":", "\\3c", "<", "\\3e", ">", "\\40", "@", "\\5c", "\\",
)

// EscapeLocal applies XEP-0106 localpart escaping.
func EscapeLocal(local string) string { return escapeLocalReplacer.Replace(local) }

// UnescapeLocal reverses XEP-0106 localpart escaping.
func UnescapeLocal(local string) string { return unescapeLocalReplacer.Replace(local) }
