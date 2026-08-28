package xmpp

import "strings"

// Matcher decides whether a stanza should be delivered to a handler.
type Matcher interface{ Match(Stanza) bool }

// MatcherFunc adapts a function to Matcher.
type MatcherFunc func(Stanza) bool

func (f MatcherFunc) Match(s Stanza) bool { return f(s) }

// MatchAll matches every stanza.
var MatchAll Matcher = MatcherFunc(func(Stanza) bool { return true })

// MatchKind matches message, presence, or iq.
func MatchKind(kind string) Matcher {
	return MatcherFunc(func(s Stanza) bool { return s != nil && s.Kind() == kind })
}

// MatchID matches a stanza ID.
func MatchID(id string) Matcher {
	return MatcherFunc(func(s Stanza) bool { return s != nil && s.StanzaID() == id })
}

// MatchFrom matches a full or bare sender JID.
func MatchFrom(value string, bare bool) Matcher {
	return MatcherFunc(func(s Stanza) bool {
		if s == nil {
			return false
		}
		from, want := s.StanzaFrom(), value
		if bare {
			from, want = BareJIDString(from), BareJIDString(want)
		}
		return strings.EqualFold(from, want)
	})
}

// MatchTo matches a full or bare recipient JID.
func MatchTo(value string, bare bool) Matcher {
	return MatcherFunc(func(s Stanza) bool {
		if s == nil {
			return false
		}
		to, want := s.StanzaTo(), value
		if bare {
			to, want = BareJIDString(to), BareJIDString(want)
		}
		return strings.EqualFold(to, want)
	})
}

// MatchPayload matches a direct extension or IQ payload QName.
func MatchPayload(ns, local string) Matcher {
	return MatcherFunc(func(s Stanza) bool {
		for _, n := range stanzaPayloads(s) {
			if n.Name.Local == local && (ns == "" || n.Name.Space == ns) {
				return true
			}
		}
		return false
	})
}

// MatchIQType matches an IQ type.
func MatchIQType(kind IQType) Matcher {
	return MatcherFunc(func(s Stanza) bool {
		switch v := s.(type) {
		case IQ:
			return v.Type == kind
		case *IQ:
			return v.Type == kind
		}
		return false
	})
}
func MatchAnd(ms ...Matcher) Matcher {
	return MatcherFunc(func(s Stanza) bool {
		for _, m := range ms {
			if m != nil && !m.Match(s) {
				return false
			}
		}
		return true
	})
}
func MatchOr(ms ...Matcher) Matcher {
	return MatcherFunc(func(s Stanza) bool {
		for _, m := range ms {
			if m != nil && m.Match(s) {
				return true
			}
		}
		return false
	})
}
func MatchNot(m Matcher) Matcher {
	return MatcherFunc(func(s Stanza) bool { return m == nil || !m.Match(s) })
}
func stanzaPayloads(s Stanza) []Node {
	switch v := s.(type) {
	case Message:
		return v.Extensions
	case *Message:
		return v.Extensions
	case Presence:
		return v.Extensions
	case *Presence:
		return v.Extensions
	case IQ:
		return v.Payloads
	case *IQ:
		return v.Payloads
	}
	return nil
}
