// Package testkit provides a deliberately non-interoperable OMEMO backend for
// unit tests and local demonstrations. It is not a Signal protocol
// implementation and must never be used for real communications.
package testkit

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/1kamma/slixmpp-go/omemo"
)

type Backend struct {
	mu       sync.Mutex
	bundle   omemo.Bundle
	sessions map[string][]byte
	fresh    map[string]bool
}

func NewBackend() (*Backend, error) {
	identity := make([]byte, 32)
	signed := make([]byte, 32)
	signature := make([]byte, 64)
	prekey := make([]byte, 32)
	for _, v := range [][]byte{identity, signed, signature, prekey} {
		if _, err := rand.Read(v); err != nil {
			return nil, err
		}
	}
	return &Backend{bundle: omemo.Bundle{IdentityKey: identity, SignedPreKeyID: 1, SignedPreKey: signed, SignedPreKeySignature: signature, PreKeys: []omemo.PreKey{{ID: 1, Public: prekey}}}, sessions: map[string][]byte{}, fresh: map[string]bool{}}, nil
}
func (b *Backend) OwnBundle(context.Context) (omemo.Bundle, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.bundle, nil
}
func (b *Backend) HasSession(_ context.Context, address omemo.Address) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.sessions[address.String()]
	return ok, nil
}
func (b *Backend) BuildSession(_ context.Context, address omemo.Address, remote omemo.Bundle) error {
	if len(remote.IdentityKey) == 0 {
		return fmt.Errorf("testkit: remote identity missing")
	}
	key := derive(b.bundle.IdentityKey, remote.IdentityKey)
	b.mu.Lock()
	b.sessions[address.String()] = key
	b.fresh[address.String()] = true
	b.mu.Unlock()
	return nil
}
func (b *Backend) Encrypt(_ context.Context, address omemo.Address, plain []byte) (omemo.SessionCiphertext, error) {
	b.mu.Lock()
	key, ok := b.sessions[address.String()]
	pre := b.fresh[address.String()]
	delete(b.fresh, address.String())
	identity := append([]byte(nil), b.bundle.IdentityKey...)
	b.mu.Unlock()
	if !ok {
		return omemo.SessionCiphertext{}, omemo.ErrMissingSession
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return omemo.SessionCiphertext{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return omemo.SessionCiphertext{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return omemo.SessionCiphertext{}, err
	}
	data := append(identity, nonce...)
	data = append(data, gcm.Seal(nil, nonce, plain, nil)...)
	return omemo.SessionCiphertext{Data: data, PreKey: pre}, nil
}
func (b *Backend) Decrypt(_ context.Context, address omemo.Address, ciphertext omemo.SessionCiphertext) ([]byte, error) {
	if len(ciphertext.Data) < 32+12 {
		return nil, fmt.Errorf("testkit: ciphertext too short")
	}
	remote := ciphertext.Data[:32]
	key := derive(b.bundle.IdentityKey, remote)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	offset := 32 + gcm.NonceSize()
	if len(ciphertext.Data) < offset {
		return nil, fmt.Errorf("testkit: ciphertext too short")
	}
	nonce := ciphertext.Data[32:offset]
	plain, err := gcm.Open(nil, nonce, ciphertext.Data[offset:], nil)
	if err != nil {
		return nil, err
	}
	b.mu.Lock()
	b.sessions[address.String()] = key
	b.mu.Unlock()
	return plain, nil
}
func derive(a, b []byte) []byte {
	left, right := a, b
	if bytes.Compare(left, right) > 0 {
		left, right = right, left
	}
	sum := sha256.Sum256(append(append([]byte(nil), left...), right...))
	return sum[:]
}
