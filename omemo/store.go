package omemo

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Store persists OMEMO integration metadata. Cryptographic session state is
// owned by SessionBackend.
type Store interface {
	OwnDeviceID(context.Context) (DeviceID, error)
	SetOwnDeviceID(context.Context, DeviceID) error
	DeviceList(context.Context, string) (DeviceList, bool, error)
	PutDeviceList(context.Context, DeviceList) error
	Bundle(context.Context, Address) (Bundle, bool, error)
	PutBundle(context.Context, Address, Bundle) error
	Trust(context.Context, Address) (TrustLevel, error)
	SetTrust(context.Context, Address, TrustLevel) error
	KnownAddresses(context.Context) ([]Address, error)
}

type memoryData struct {
	Own     DeviceID              `json:"own_device_id"`
	Lists   map[string]DeviceList `json:"device_lists"`
	Bundles map[string]Bundle     `json:"bundles"`
	Trust   map[string]TrustLevel `json:"trust"`
}

// MemoryStore is a concurrency-safe in-memory Store.
type MemoryStore struct {
	mu   sync.RWMutex
	data memoryData
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: memoryData{Lists: map[string]DeviceList{}, Bundles: map[string]Bundle{}, Trust: map[string]TrustLevel{}}}
}
func (s *MemoryStore) ensure() {
	if s.data.Lists == nil {
		s.data.Lists = map[string]DeviceList{}
	}
	if s.data.Bundles == nil {
		s.data.Bundles = map[string]Bundle{}
	}
	if s.data.Trust == nil {
		s.data.Trust = map[string]TrustLevel{}
	}
}
func (s *MemoryStore) OwnDeviceID(context.Context) (DeviceID, error) {
	s.mu.RLock()
	id := s.data.Own
	s.mu.RUnlock()
	if id != 0 {
		return id, nil
	}
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return 0, err
	}
	id = DeviceID(binary.BigEndian.Uint32(raw[:]) & 0x7fffffff)
	if id == 0 {
		id = 1
	}
	s.mu.Lock()
	if s.data.Own == 0 {
		s.data.Own = id
	} else {
		id = s.data.Own
	}
	s.mu.Unlock()
	return id, nil
}
func (s *MemoryStore) SetOwnDeviceID(_ context.Context, id DeviceID) error {
	if id == 0 {
		return fmt.Errorf("omemo: device ID must be nonzero")
	}
	s.mu.Lock()
	s.ensure()
	s.data.Own = id
	s.mu.Unlock()
	return nil
}
func (s *MemoryStore) DeviceList(_ context.Context, jid string) (DeviceList, bool, error) {
	s.mu.RLock()
	v, ok := s.data.Lists[jid]
	s.mu.RUnlock()
	v.Devices = append([]DeviceID(nil), v.Devices...)
	return v, ok, nil
}
func (s *MemoryStore) PutDeviceList(_ context.Context, list DeviceList) error {
	list.Devices = uniqueDevices(list.Devices)
	if list.FetchedAt.IsZero() {
		list.FetchedAt = time.Now()
	}
	s.mu.Lock()
	s.ensure()
	s.data.Lists[list.JID] = list
	s.mu.Unlock()
	return nil
}
func (s *MemoryStore) Bundle(_ context.Context, address Address) (Bundle, bool, error) {
	s.mu.RLock()
	v, ok := s.data.Bundles[address.String()]
	s.mu.RUnlock()
	return cloneBundle(v), ok, nil
}
func (s *MemoryStore) PutBundle(_ context.Context, address Address, bundle Bundle) error {
	s.mu.Lock()
	s.ensure()
	s.data.Bundles[address.String()] = cloneBundle(bundle)
	s.mu.Unlock()
	return nil
}
func (s *MemoryStore) Trust(_ context.Context, address Address) (TrustLevel, error) {
	s.mu.RLock()
	v, ok := s.data.Trust[address.String()]
	s.mu.RUnlock()
	if !ok {
		return TrustUndecided, nil
	}
	return v, nil
}
func (s *MemoryStore) SetTrust(_ context.Context, address Address, level TrustLevel) error {
	if !validTrust(level) {
		return fmt.Errorf("omemo: invalid trust level %q", level)
	}
	s.mu.Lock()
	s.ensure()
	s.data.Trust[address.String()] = level
	s.mu.Unlock()
	return nil
}
func (s *MemoryStore) KnownAddresses(context.Context) ([]Address, error) {
	s.mu.RLock()
	keys := map[string]bool{}
	for key := range s.data.Bundles {
		keys[key] = true
	}
	for key := range s.data.Trust {
		keys[key] = true
	}
	s.mu.RUnlock()

	out := make([]Address, 0, len(keys))
	for key := range keys {
		separator := strings.LastIndexByte(key, ':')
		if separator <= 0 || separator == len(key)-1 {
			continue
		}
		value, err := strconv.ParseUint(key[separator+1:], 10, 32)
		if err != nil || value == 0 {
			continue
		}
		out = append(out, Address{JID: key[:separator], DeviceID: DeviceID(value)})
	}
	return out, nil
}

// JSONStore persists MemoryStore metadata atomically in a mode-0600 JSON file.
type JSONStore struct {
	*MemoryStore
	path      string
	persistMu sync.Mutex
}

func OpenJSONStore(path string) (*JSONStore, error) {
	s := &JSONStore{MemoryStore: NewMemoryStore(), path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &s.data); err != nil {
		return nil, fmt.Errorf("omemo: decode store: %w", err)
	}
	s.ensure()
	return s, nil
}
func (s *JSONStore) persist() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.RLock()
	data, err := json.MarshalIndent(s.data, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
func (s *JSONStore) SetOwnDeviceID(ctx context.Context, id DeviceID) error {
	if err := s.MemoryStore.SetOwnDeviceID(ctx, id); err != nil {
		return err
	}
	return s.persist()
}
func (s *JSONStore) PutDeviceList(ctx context.Context, v DeviceList) error {
	if err := s.MemoryStore.PutDeviceList(ctx, v); err != nil {
		return err
	}
	return s.persist()
}
func (s *JSONStore) PutBundle(ctx context.Context, a Address, v Bundle) error {
	if err := s.MemoryStore.PutBundle(ctx, a, v); err != nil {
		return err
	}
	return s.persist()
}
func (s *JSONStore) SetTrust(ctx context.Context, a Address, v TrustLevel) error {
	if err := s.MemoryStore.SetTrust(ctx, a, v); err != nil {
		return err
	}
	return s.persist()
}
func (s *JSONStore) OwnDeviceID(ctx context.Context) (DeviceID, error) {
	s.mu.RLock()
	before := s.data.Own
	s.mu.RUnlock()
	id, err := s.MemoryStore.OwnDeviceID(ctx)
	if err != nil {
		return 0, err
	}
	if before == 0 {
		if err := s.persist(); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func validTrust(v TrustLevel) bool {
	switch v {
	case TrustUndecided, TrustTrusted, TrustVerified, TrustDistrusted, TrustIgnored:
		return true
	}
	return false
}
func uniqueDevices(values []DeviceID) []DeviceID {
	seen := map[DeviceID]bool{}
	out := make([]DeviceID, 0, len(values))
	for _, v := range values {
		if v != 0 && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
func cloneBundle(v Bundle) Bundle {
	v.IdentityKey = append([]byte(nil), v.IdentityKey...)
	v.SignedPreKey = append([]byte(nil), v.SignedPreKey...)
	v.SignedPreKeySignature = append([]byte(nil), v.SignedPreKeySignature...)
	v.PreKeys = append([]PreKey(nil), v.PreKeys...)
	for i := range v.PreKeys {
		v.PreKeys[i].Public = append([]byte(nil), v.PreKeys[i].Public...)
	}
	return v
}
