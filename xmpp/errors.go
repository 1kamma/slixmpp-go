package xmpp

import (
	"errors"
	"fmt"
)

var (
	ErrNotConnected   = errors.New("xmpp: not connected")
	ErrClosed         = errors.New("xmpp: stream closed")
	ErrIQTimeout      = errors.New("xmpp: IQ timeout")
	ErrAuthentication = errors.New("xmpp: authentication failed")
	ErrTLSRequired    = errors.New("xmpp: TLS required but unavailable")
)

// StanzaError is the structured <error/> child of a stanza.
type StanzaError struct {
	Type        string
	By          string
	Condition   string
	Text        string
	Application []Node
}

func (e *StanzaError) Error() string {
	if e == nil {
		return ""
	}
	message := "xmpp stanza error"
	if e.Type != "" {
		message += " " + e.Type
	}
	if e.Condition != "" {
		message += ": " + e.Condition
	}
	if e.Text != "" {
		message += ": " + e.Text
	}
	return message
}

// IQResponseError wraps an IQ of type error.
type IQResponseError struct{ IQ IQ }

func (e *IQResponseError) Error() string {
	if e == nil {
		return "xmpp: IQ error"
	}
	if e.IQ.Error != nil {
		return fmt.Sprintf("xmpp: IQ %s failed: %v", e.IQ.ID, e.IQ.Error)
	}
	return fmt.Sprintf("xmpp: IQ %s failed", e.IQ.ID)
}
