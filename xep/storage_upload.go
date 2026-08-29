package xep

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/1kamma/slixmpp-go/xmpp"
)

const (
	// PrivateXMLNS is the XEP-0049 private-storage namespace.
	PrivateXMLNS = "jabber:iq:private"
	// LegacyBookmarksNS is the XEP-0048 legacy bookmark namespace.
	LegacyBookmarksNS = "storage:bookmarks"
	// BookmarksNS is the XEP-0402 PEP native bookmark namespace.
	BookmarksNS = "urn:xmpp:bookmarks:1"
	// HTTPUploadNS is the XEP-0363 namespace.
	HTTPUploadNS = "urn:xmpp:http:upload:0"
)

// PrivateXML implements XEP-0049 private XML storage.
type PrivateXML struct {
	client *xmpp.Client
}

// NewPrivateXML creates a private-storage plugin.
func NewPrivateXML() *PrivateXML { return &PrivateXML{} }

func (p *PrivateXML) Name() string           { return "xep_0049" }
func (p *PrivateXML) Description() string    { return "XEP-0049 Private XML Storage" }
func (p *PrivateXML) Dependencies() []string { return nil }
func (p *PrivateXML) Features() []string     { return nil }
func (p *PrivateXML) Init(client *xmpp.Client) error {
	p.client = client
	return nil
}
func (p *PrivateXML) Shutdown(context.Context) error { return nil }

// Get retrieves one namespaced element from private storage.
func (p *PrivateXML) Get(ctx context.Context, namespace, local string) (xmpp.Node, error) {
	if p.client == nil {
		return xmpp.Node{}, fmt.Errorf("xep: private XML plugin is not initialized")
	}
	query := xmpp.NewNode(PrivateXMLNS, "query")
	query.AddChild(xmpp.NewNode(namespace, local))
	response, err := p.client.RequestIQ(ctx, xmpp.IQ{
		Type:     xmpp.IQGet,
		Payloads: []xmpp.Node{query},
	})
	if err != nil {
		return xmpp.Node{}, err
	}
	payload := response.Payload()
	if payload == nil {
		return xmpp.Node{}, fmt.Errorf("xep: missing private XML response")
	}
	node := payload.Child(namespace, local)
	if node == nil {
		return xmpp.Node{}, fmt.Errorf("xep: private XML payload {%s}%s missing", namespace, local)
	}
	return *node, nil
}

// Set replaces one namespaced element in private storage.
func (p *PrivateXML) Set(ctx context.Context, node xmpp.Node) error {
	if p.client == nil {
		return fmt.Errorf("xep: private XML plugin is not initialized")
	}
	query := xmpp.NewNode(PrivateXMLNS, "query")
	query.AddChild(node)
	_, err := p.client.RequestIQ(ctx, xmpp.IQ{
		Type:     xmpp.IQSet,
		Payloads: []xmpp.Node{query},
	})
	return err
}

// Bookmark is a conference bookmark shared by XEP-0048 and XEP-0402.
type Bookmark struct {
	JID        string
	Name       string
	Nick       string
	Password   string
	AutoJoin   bool
	Extensions []xmpp.Node
}

func (b Bookmark) legacyNode() xmpp.Node {
	node := xmpp.NewNode(LegacyBookmarksNS, "conference")
	node.SetAttr("jid", b.JID)
	if b.Name != "" {
		node.SetAttr("name", b.Name)
	}
	if b.AutoJoin {
		node.SetAttr("autojoin", "true")
	}
	if b.Nick != "" {
		node.AddChild(xmpp.NewTextNode(LegacyBookmarksNS, "nick", b.Nick))
	}
	if b.Password != "" {
		node.AddChild(xmpp.NewTextNode(LegacyBookmarksNS, "password", b.Password))
	}
	return node
}

func (b Bookmark) nativeNode() xmpp.Node {
	node := xmpp.NewNode(BookmarksNS, "conference")
	if b.Name != "" {
		node.SetAttr("name", b.Name)
	}
	if b.AutoJoin {
		node.SetAttr("autojoin", "true")
	}
	if b.Nick != "" {
		node.AddChild(xmpp.NewTextNode(BookmarksNS, "nick", b.Nick))
	}
	if b.Password != "" {
		node.AddChild(xmpp.NewTextNode(BookmarksNS, "password", b.Password))
	}
	for _, extension := range b.Extensions {
		node.AddChild(extension)
	}
	return node
}

// LegacyBookmarks implements XEP-0048 over private XML storage.
type LegacyBookmarks struct {
	private *PrivateXML
}

// NewLegacyBookmarks creates a legacy bookmark plugin.
func NewLegacyBookmarks() *LegacyBookmarks { return &LegacyBookmarks{} }

func (b *LegacyBookmarks) Name() string        { return "xep_0048" }
func (b *LegacyBookmarks) Description() string { return "XEP-0048 Bookmarks" }
func (b *LegacyBookmarks) Dependencies() []string {
	return []string{"xep_0049"}
}
func (b *LegacyBookmarks) Features() []string { return nil }
func (b *LegacyBookmarks) Init(client *xmpp.Client) error {
	plugin, ok := client.Plugins.Get("xep_0049")
	if !ok {
		return fmt.Errorf("xep: xep_0049 not loaded")
	}
	var cast bool
	b.private, cast = plugin.(*PrivateXML)
	if !cast {
		return fmt.Errorf("xep: unexpected xep_0049 plugin type")
	}
	return nil
}
func (b *LegacyBookmarks) Shutdown(context.Context) error { return nil }

// Get retrieves all legacy conference bookmarks.
func (b *LegacyBookmarks) Get(ctx context.Context) ([]Bookmark, error) {
	storage, err := b.private.Get(ctx, LegacyBookmarksNS, "storage")
	if err != nil {
		return nil, err
	}
	var result []Bookmark
	for _, node := range storage.Children() {
		if node.Name.Local != "conference" {
			continue
		}
		bookmark := Bookmark{}
		bookmark.JID, _ = node.AttrValue("jid")
		bookmark.Name, _ = node.AttrValue("name")
		if raw, ok := node.AttrValue("autojoin"); ok {
			bookmark.AutoJoin = raw == "true" || raw == "1"
		}
		bookmark.Nick = node.ChildText(LegacyBookmarksNS, "nick")
		bookmark.Password = node.ChildText(LegacyBookmarksNS, "password")
		result = append(result, bookmark)
	}
	return result, nil
}

// Set replaces all legacy conference bookmarks.
func (b *LegacyBookmarks) Set(ctx context.Context, bookmarks []Bookmark) error {
	storage := xmpp.NewNode(LegacyBookmarksNS, "storage")
	for _, bookmark := range bookmarks {
		if bookmark.JID == "" {
			return fmt.Errorf("xep: bookmark JID is required")
		}
		storage.AddChild(bookmark.legacyNode())
	}
	return b.private.Set(ctx, storage)
}

// NativeBookmarks implements XEP-0402 over PEP.
type NativeBookmarks struct {
	pep *PEP
}

// NewNativeBookmarks creates a native bookmark plugin.
func NewNativeBookmarks() *NativeBookmarks { return &NativeBookmarks{} }

func (b *NativeBookmarks) Name() string        { return "xep_0402" }
func (b *NativeBookmarks) Description() string { return "XEP-0402 PEP Native Bookmarks" }
func (b *NativeBookmarks) Dependencies() []string {
	return []string{"xep_0163"}
}
func (b *NativeBookmarks) Features() []string { return []string{BookmarksNS} }
func (b *NativeBookmarks) Init(client *xmpp.Client) error {
	plugin, ok := client.Plugins.Get("xep_0163")
	if !ok {
		return fmt.Errorf("xep: xep_0163 not loaded")
	}
	var cast bool
	b.pep, cast = plugin.(*PEP)
	if !cast {
		return fmt.Errorf("xep: unexpected xep_0163 plugin type")
	}
	return nil
}
func (b *NativeBookmarks) Shutdown(context.Context) error { return nil }

// Publish creates or updates one native bookmark.
func (b *NativeBookmarks) Publish(ctx context.Context, bookmark Bookmark) error {
	if bookmark.JID == "" {
		return fmt.Errorf("xep: bookmark JID is required")
	}
	options := Form{Fields: []Field{
		{Var: "pubsub#persist_items", Type: FieldBoolean, Values: []string{"true"}},
		{Var: "pubsub#access_model", Type: FieldListSingle, Values: []string{"whitelist"}},
		{Var: "pubsub#send_last_published_item", Type: FieldListSingle, Values: []string{"never"}},
	}}
	_, err := b.pep.Publish(ctx, BookmarksNS, bookmark.JID, bookmark.nativeNode(), &options)
	return err
}

// Delete retracts a native bookmark by room JID.
func (b *NativeBookmarks) Delete(ctx context.Context, jid string) error {
	return b.pep.PubSub.Retract(ctx, "", BookmarksNS, true, jid)
}

// Get retrieves native bookmarks from jid's PEP service.
func (b *NativeBookmarks) Get(ctx context.Context, jid string) ([]Bookmark, error) {
	items, err := b.pep.Items(ctx, jid, BookmarksNS, 0)
	if err != nil {
		return nil, err
	}
	var result []Bookmark
	for _, item := range items {
		for _, node := range item.Payloads {
			if node.Name.Space != BookmarksNS || node.Name.Local != "conference" {
				continue
			}
			bookmark := Bookmark{JID: item.ID}
			bookmark.Name, _ = node.AttrValue("name")
			if raw, ok := node.AttrValue("autojoin"); ok {
				bookmark.AutoJoin = raw == "true" || raw == "1"
			}
			bookmark.Nick = node.ChildText(BookmarksNS, "nick")
			bookmark.Password = node.ChildText(BookmarksNS, "password")
			for _, extension := range node.Children() {
				if extension.Name.Local != "nick" && extension.Name.Local != "password" {
					bookmark.Extensions = append(bookmark.Extensions, extension)
				}
			}
			result = append(result, bookmark)
		}
	}
	return result, nil
}

// UploadSlot is an XEP-0363 upload slot.
type UploadSlot struct {
	PutURL  string
	GetURL  string
	Headers http.Header
}

// HTTPUpload implements XEP-0363 slot discovery, requests, and HTTP PUT.
type HTTPUpload struct {
	client            *xmpp.Client
	HTTPClient        *http.Client
	AllowInsecureHTTP bool
}

// NewHTTPUpload creates an HTTP upload plugin.
func NewHTTPUpload() *HTTPUpload {
	return &HTTPUpload{HTTPClient: &http.Client{Timeout: 2 * time.Minute}}
}

func (u *HTTPUpload) Name() string        { return "xep_0363" }
func (u *HTTPUpload) Description() string { return "XEP-0363 HTTP File Upload" }
func (u *HTTPUpload) Dependencies() []string {
	return []string{"xep_0030"}
}
func (u *HTTPUpload) Features() []string { return []string{HTTPUploadNS} }
func (u *HTTPUpload) Init(client *xmpp.Client) error {
	u.client = client
	if u.HTTPClient == nil {
		u.HTTPClient = &http.Client{Timeout: 2 * time.Minute}
	}
	return nil
}
func (u *HTTPUpload) Shutdown(context.Context) error { return nil }

// RequestSlot requests a PUT/GET slot from an upload service.
func (u *HTTPUpload) RequestSlot(ctx context.Context, service, filename, mediaType string, size int64) (UploadSlot, error) {
	if u.client == nil {
		return UploadSlot{}, fmt.Errorf("xep: HTTP upload plugin is not initialized")
	}
	if filename == "" {
		return UploadSlot{}, fmt.Errorf("xep: upload filename is required")
	}
	if size < 0 {
		return UploadSlot{}, fmt.Errorf("xep: upload size cannot be negative")
	}
	request := xmpp.NewNode(HTTPUploadNS, "request")
	request.SetAttr("filename", filename)
	request.SetAttr("size", strconv.FormatInt(size, 10))
	if mediaType != "" {
		request.SetAttr("content-type", mediaType)
	}
	response, err := u.client.RequestIQ(ctx, xmpp.IQ{
		To:       service,
		Type:     xmpp.IQGet,
		Payloads: []xmpp.Node{request},
	})
	if err != nil {
		return UploadSlot{}, err
	}
	payload := response.Payload()
	if payload == nil || payload.Name.Space != HTTPUploadNS || payload.Name.Local != "slot" {
		return UploadSlot{}, fmt.Errorf("xep: missing HTTP upload slot")
	}
	slot := UploadSlot{Headers: make(http.Header)}
	if put := payload.Child(HTTPUploadNS, "put"); put != nil {
		slot.PutURL, _ = put.AttrValue("url")
		for _, header := range put.Children() {
			if header.Name.Local != "header" {
				continue
			}
			name, _ := header.AttrValue("name")
			if name != "" {
				slot.Headers.Add(name, header.Text())
			}
		}
	}
	if get := payload.Child(HTTPUploadNS, "get"); get != nil {
		slot.GetURL, _ = get.AttrValue("url")
	}
	if slot.PutURL == "" || slot.GetURL == "" {
		return UploadSlot{}, fmt.Errorf("xep: incomplete HTTP upload slot")
	}
	return slot, nil
}

// Upload performs the HTTP PUT for a previously issued slot.
func (u *HTTPUpload) Upload(ctx context.Context, slot UploadSlot, body io.Reader, size int64, mediaType string) error {
	parsed, err := url.Parse(slot.PutURL)
	if err != nil {
		return fmt.Errorf("xep: invalid upload URL: %w", err)
	}
	if !u.AllowInsecureHTTP && parsed.Scheme != "https" {
		return fmt.Errorf("xep: refusing non-HTTPS upload URL")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, slot.PutURL, body)
	if err != nil {
		return err
	}
	request.ContentLength = size
	if mediaType != "" {
		request.Header.Set("Content-Type", mediaType)
	}
	for name, values := range slot.Headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	response, err := u.HTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("xep: upload failed with HTTP %s", response.Status)
	}
	return nil
}

// Discover locates a compatible upload service and advertised maximum size.
func (u *HTTPUpload) Discover(ctx context.Context, domain string) (service string, maxSize int64, err error) {
	plugin, ok := u.client.Plugins.Get("xep_0030")
	if !ok {
		return "", 0, fmt.Errorf("xep: discovery plugin not loaded")
	}
	disco, ok := plugin.(*Disco)
	if !ok {
		return "", 0, fmt.Errorf("xep: unexpected discovery plugin type")
	}
	items, err := disco.GetItems(ctx, domain, "", nil)
	if err != nil {
		return "", 0, err
	}
	for _, item := range items.Items {
		info, infoErr := disco.GetInfo(ctx, item.JID, "")
		if infoErr != nil {
			continue
		}
		supported := false
		for _, feature := range info.Features {
			if feature == HTTPUploadNS {
				supported = true
				break
			}
		}
		if !supported {
			continue
		}
		for _, form := range info.Forms {
			if field := form.Field("max-file-size"); field != nil && len(field.Values) > 0 {
				maxSize, _ = strconv.ParseInt(field.Values[0], 10, 64)
			}
		}
		return item.JID, maxSize, nil
	}
	return "", 0, fmt.Errorf("xep: no HTTP upload service discovered")
}

// MUCSelfPing implements XEP-0410 using the loaded MUC and ping plugins.
type MUCSelfPing struct {
	muc  *MUC
	ping *Ping
}

// NewMUCSelfPing creates a MUC self-ping plugin.
func NewMUCSelfPing() *MUCSelfPing { return &MUCSelfPing{} }

func (m *MUCSelfPing) Name() string        { return "xep_0410" }
func (m *MUCSelfPing) Description() string { return "XEP-0410 MUC Self-Ping" }
func (m *MUCSelfPing) Dependencies() []string {
	return []string{"xep_0045", "xep_0199"}
}
func (m *MUCSelfPing) Features() []string { return nil }
func (m *MUCSelfPing) Init(client *xmpp.Client) error {
	mucPlugin, _ := client.Plugins.Get("xep_0045")
	pingPlugin, _ := client.Plugins.Get("xep_0199")
	var ok bool
	m.muc, ok = mucPlugin.(*MUC)
	if !ok {
		return fmt.Errorf("xep: unexpected MUC plugin type")
	}
	m.ping, ok = pingPlugin.(*Ping)
	if !ok {
		return fmt.Errorf("xep: unexpected ping plugin type")
	}
	return nil
}
func (m *MUCSelfPing) Shutdown(context.Context) error { return nil }

// Ping checks whether the current room occupant is still present.
func (m *MUCSelfPing) Ping(ctx context.Context, room string) (time.Duration, error) {
	state, ok := m.muc.Room(room)
	if !ok || !state.Joined {
		return 0, fmt.Errorf("xep: not joined to %s", room)
	}
	return m.ping.Ping(ctx, xmpp.BareJIDString(room)+"/"+state.Nick)
}

func init() {
	registerSpecialized(48, func() xmpp.Plugin { return NewLegacyBookmarks() })
	registerSpecialized(49, func() xmpp.Plugin { return NewPrivateXML() })
	registerSpecialized(363, func() xmpp.Plugin { return NewHTTPUpload() })
	registerSpecialized(402, func() xmpp.Plugin { return NewNativeBookmarks() })
	registerSpecialized(410, func() xmpp.Plugin { return NewMUCSelfPing() })
}
