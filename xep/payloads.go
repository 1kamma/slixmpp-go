package xep

import (
	"bytes"
	"crypto/md5"  // #nosec G501 -- included only for legacy XMPP hash verification.
	"crypto/sha1" // #nosec G505 -- XEP-0231 CID generation mandates SHA-1.
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"

	"github.com/saret/slixmpp-go/xmpp"
)

const (
	HashesNS       = "urn:xmpp:hashes:2"
	BoBNS          = "urn:xmpp:bob"
	ForwardNS      = "urn:xmpp:forward:0"
	CommandsNS     = "http://jabber.org/protocol/commands"
	FileMetadataNS = "urn:xmpp:file:metadata:0"
	SFSNS          = "urn:xmpp:sfs:0"
	SIMSNS         = "urn:xmpp:sims:1"
	URLDataNS      = "http://jabber.org/protocol/url-data"
	LinkMetadataNS = "urn:xmpp:link-metadata:0"
	OMEMOMediaNS   = "urn:xmpp:omemo:2"
)

// HashValue is an XEP-0300 hash element.
type HashValue struct {
	Algorithm string
	Sum       []byte
}

func (h HashValue) ToNode() xmpp.Node {
	n := xmpp.NewTextNode(HashesNS, "hash", base64.StdEncoding.EncodeToString(h.Sum))
	n.SetAttr("algo", h.Algorithm)
	return n
}
func ParseHash(n xmpp.Node) (HashValue, error) {
	if n.Name.Space != HashesNS || n.Name.Local != "hash" {
		return HashValue{}, fmt.Errorf("xep: invalid hash element")
	}
	algo, _ := n.AttrValue("algo")
	sum, err := base64.StdEncoding.DecodeString(strings.TrimSpace(n.Text()))
	if err != nil {
		return HashValue{}, err
	}
	return HashValue{Algorithm: algo, Sum: sum}, nil
}
func hashFactory(algorithm string) (func() hash.Hash, error) {
	switch strings.ToLower(algorithm) {
	case "md5":
		return md5.New, nil
	case "sha-1":
		return sha1.New, nil
	case "sha-256":
		return sha256.New, nil
	case "sha-512":
		return sha512.New, nil
	case "sha3-256", "sha3-512", "blake2b-256", "blake2b-512":
		return nil, fmt.Errorf("xep: algorithm %q requires an external implementation", algorithm)
	default:
		return nil, fmt.Errorf("xep: unsupported hash algorithm %q", algorithm)
	}
}
func ComputeHash(algorithm string, data []byte) (HashValue, error) {
	factory, err := hashFactory(algorithm)
	if err != nil {
		return HashValue{}, err
	}
	h := factory()
	_, _ = h.Write(data)
	return HashValue{Algorithm: algorithm, Sum: h.Sum(nil)}, nil
}
func (h HashValue) Verify(data []byte) bool {
	got, err := ComputeHash(h.Algorithm, data)
	return err == nil && bytes.Equal(got.Sum, h.Sum)
}

// BoB is an XEP-0231 Bits of Binary payload.
type BoB struct {
	CID, MediaType string
	MaxAge         *int
	Data           []byte
}

func NewBoB(mediaType string, data []byte) BoB {
	sum := sha1.Sum(data)
	return BoB{CID: "sha1+" + hex.EncodeToString(sum[:]) + "@bob.xmpp.org", MediaType: mediaType, Data: append([]byte(nil), data...)}
}
func (b BoB) ToNode() xmpp.Node {
	n := xmpp.NewTextNode(BoBNS, "data", base64.StdEncoding.EncodeToString(b.Data))
	n.SetAttr("cid", b.CID)
	if b.MediaType != "" {
		n.SetAttr("type", b.MediaType)
	}
	if b.MaxAge != nil {
		n.SetAttr("max-age", strconv.Itoa(*b.MaxAge))
	}
	return n
}
func ParseBoB(n xmpp.Node) (BoB, error) {
	if n.Name.Space != BoBNS || n.Name.Local != "data" {
		return BoB{}, fmt.Errorf("xep: invalid Bits of Binary payload")
	}
	b := BoB{}
	b.CID, _ = n.AttrValue("cid")
	b.MediaType, _ = n.AttrValue("type")
	if v, ok := n.AttrValue("max-age"); ok {
		i, err := strconv.Atoi(v)
		if err != nil {
			return b, err
		}
		b.MaxAge = &i
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(n.Text()))
	if err != nil {
		return b, err
	}
	b.Data = data
	return b, nil
}

// Forwarded is an XEP-0297 forwarded stanza container.
type Forwarded struct {
	Delay  *Delay
	Stanza xmpp.Stanza
}

func (f Forwarded) ToNode() (xmpp.Node, error) {
	n := xmpp.NewNode(ForwardNS, "forwarded")
	if f.Delay != nil {
		n.AddChild(f.Delay.ToNode())
	}
	if f.Stanza != nil {
		data, err := xmpp.StanzaXML(f.Stanza)
		if err != nil {
			return xmpp.Node{}, err
		}
		child, err := xmpp.ParseNode(data)
		if err != nil {
			return xmpp.Node{}, err
		}
		n.AddChild(child)
	}
	return n, nil
}
func ParseForwarded(n xmpp.Node) (Forwarded, error) {
	if n.Name.Space != ForwardNS || n.Name.Local != "forwarded" {
		return Forwarded{}, fmt.Errorf("xep: invalid forwarded payload")
	}
	var f Forwarded
	for _, child := range n.Children() {
		if child.Name.Space == DelayNS && child.Name.Local == "delay" {
			delay, err := ParseDelay(child)
			if err != nil {
				return f, err
			}
			f.Delay = &delay
			continue
		}
		if child.Name.Local == "message" || child.Name.Local == "presence" || child.Name.Local == "iq" {
			raw, err := child.XML()
			if err != nil {
				return f, err
			}
			stanza, err := xmpp.ParseStanza([]byte(raw))
			if err != nil {
				return f, err
			}
			f.Stanza = stanza
		}
	}
	return f, nil
}

// CommandAction is an XEP-0050 action.
type CommandAction string

const (
	CommandExecute  CommandAction = "execute"
	CommandNext     CommandAction = "next"
	CommandPrev     CommandAction = "prev"
	CommandComplete CommandAction = "complete"
	CommandCancel   CommandAction = "cancel"
)

type CommandStatus string

const (
	CommandExecuting CommandStatus = "executing"
	CommandCompleted CommandStatus = "completed"
	CommandCanceled  CommandStatus = "canceled"
)

type CommandNote struct{ Type, Text string }
type Command struct {
	Node, SessionID string
	Action          CommandAction
	Status          CommandStatus
	Allowed         []CommandAction
	Form            *Form
	Notes           []CommandNote
	Payloads        []xmpp.Node
}

func (c Command) ToNode() xmpp.Node {
	n := xmpp.NewNode(CommandsNS, "command")
	n.SetAttr("node", c.Node)
	if c.SessionID != "" {
		n.SetAttr("sessionid", c.SessionID)
	}
	if c.Action != "" {
		n.SetAttr("action", string(c.Action))
	}
	if c.Status != "" {
		n.SetAttr("status", string(c.Status))
	}
	if len(c.Allowed) > 0 {
		actions := xmpp.NewNode(CommandsNS, "actions")
		for _, a := range c.Allowed {
			actions.AddChild(xmpp.NewNode(CommandsNS, string(a)))
		}
		n.AddChild(actions)
	}
	if c.Form != nil {
		n.AddChild(c.Form.ToNode())
	}
	for _, note := range c.Notes {
		x := xmpp.NewTextNode(CommandsNS, "note", note.Text)
		if note.Type != "" {
			x.SetAttr("type", note.Type)
		}
		n.AddChild(x)
	}
	for _, p := range c.Payloads {
		n.AddChild(p)
	}
	return n
}
func ParseCommand(n xmpp.Node) (Command, error) {
	if n.Name.Space != CommandsNS || n.Name.Local != "command" {
		return Command{}, fmt.Errorf("xep: invalid command payload")
	}
	c := Command{}
	c.Node, _ = n.AttrValue("node")
	c.SessionID, _ = n.AttrValue("sessionid")
	if v, ok := n.AttrValue("action"); ok {
		c.Action = CommandAction(v)
	}
	if v, ok := n.AttrValue("status"); ok {
		c.Status = CommandStatus(v)
	}
	for _, child := range n.Children() {
		switch {
		case child.Name.Space == CommandsNS && child.Name.Local == "actions":
			for _, a := range child.Children() {
				c.Allowed = append(c.Allowed, CommandAction(a.Name.Local))
			}
		case child.Name.Space == DataFormsNS && child.Name.Local == "x":
			form, err := ParseForm(child)
			if err != nil {
				return c, err
			}
			c.Form = &form
		case child.Name.Space == CommandsNS && child.Name.Local == "note":
			kind, _ := child.AttrValue("type")
			c.Notes = append(c.Notes, CommandNote{Type: kind, Text: child.Text()})
		default:
			c.Payloads = append(c.Payloads, child)
		}
	}
	return c, nil
}

// FileMetadata is the common XEP-0446 file description.
type FileMetadata struct {
	Name, Description, MediaType string
	Size                         *uint64
	Date                         *time.Time
	Duration                     *time.Duration
	Height, Width                *int
	Hashes                       []HashValue
	Thumbnails                   []string
}

func (f FileMetadata) ToNode() xmpp.Node {
	n := xmpp.NewNode(FileMetadataNS, "file")
	if f.Date != nil {
		n.AddChild(xmpp.NewTextNode(FileMetadataNS, "date", FormatDateTime(*f.Date)))
	}
	if f.Description != "" {
		n.AddChild(xmpp.NewTextNode(FileMetadataNS, "desc", f.Description))
	}
	if f.MediaType != "" {
		n.AddChild(xmpp.NewTextNode(FileMetadataNS, "media-type", f.MediaType))
	}
	if f.Name != "" {
		n.AddChild(xmpp.NewTextNode(FileMetadataNS, "name", f.Name))
	}
	if f.Size != nil {
		n.AddChild(xmpp.NewTextNode(FileMetadataNS, "size", strconv.FormatUint(*f.Size, 10)))
	}
	if f.Duration != nil {
		n.AddChild(xmpp.NewTextNode(FileMetadataNS, "duration", strconv.FormatInt(f.Duration.Milliseconds(), 10)))
	}
	if f.Height != nil {
		n.AddChild(xmpp.NewTextNode(FileMetadataNS, "height", strconv.Itoa(*f.Height)))
	}
	if f.Width != nil {
		n.AddChild(xmpp.NewTextNode(FileMetadataNS, "width", strconv.Itoa(*f.Width)))
	}
	for _, h := range f.Hashes {
		n.AddChild(h.ToNode())
	}
	for _, uri := range f.Thumbnails {
		x := xmpp.NewNode(FileMetadataNS, "thumbnail")
		x.SetAttr("uri", uri)
		n.AddChild(x)
	}
	return n
}
func ParseFileMetadata(n xmpp.Node) (FileMetadata, error) {
	if n.Name.Space != FileMetadataNS || n.Name.Local != "file" {
		return FileMetadata{}, fmt.Errorf("xep: invalid file metadata")
	}
	var f FileMetadata
	for _, c := range n.Children() {
		switch c.Name.Local {
		case "date":
			v, e := ParseDateTime(c.Text())
			if e != nil {
				return f, e
			}
			f.Date = &v
		case "desc":
			f.Description = c.Text()
		case "media-type":
			f.MediaType = c.Text()
		case "name":
			f.Name = c.Text()
		case "size":
			v, e := strconv.ParseUint(c.Text(), 10, 64)
			if e != nil {
				return f, e
			}
			f.Size = &v
		case "duration":
			v, e := strconv.ParseInt(c.Text(), 10, 64)
			if e != nil {
				return f, e
			}
			d := time.Duration(v) * time.Millisecond
			f.Duration = &d
		case "height":
			v, e := strconv.Atoi(c.Text())
			if e != nil {
				return f, e
			}
			f.Height = &v
		case "width":
			v, e := strconv.Atoi(c.Text())
			if e != nil {
				return f, e
			}
			f.Width = &v
		case "hash":
			if c.Name.Space == HashesNS {
				h, e := ParseHash(c)
				if e != nil {
					return f, e
				}
				f.Hashes = append(f.Hashes, h)
			}
		case "thumbnail":
			uri, _ := c.AttrValue("uri")
			f.Thumbnails = append(f.Thumbnails, uri)
		}
	}
	return f, nil
}

type URLSource struct {
	URL     string
	Headers map[string]string
}

func (s URLSource) ToNode() xmpp.Node {
	n := xmpp.NewNode(URLDataNS, "url-data")
	n.SetAttr("target", s.URL)
	keys := make([]string, 0, len(s.Headers))
	for k := range s.Headers {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		h := xmpp.NewTextNode(URLDataNS, "header", s.Headers[k])
		h.SetAttr("name", k)
		n.AddChild(h)
	}
	return n
}

type StatelessFile struct {
	Disposition string
	File        FileMetadata
	Sources     []URLSource
}

func (s StatelessFile) ToNode() xmpp.Node {
	n := xmpp.NewNode(SFSNS, "file-sharing")
	if s.Disposition != "" {
		n.SetAttr("disposition", s.Disposition)
	}
	n.AddChild(s.File.ToNode())
	sources := xmpp.NewNode(SFSNS, "sources")
	for _, source := range s.Sources {
		sources.AddChild(source.ToNode())
	}
	n.AddChild(sources)
	return n
}
func ParseStatelessFile(n xmpp.Node) (StatelessFile, error) {
	if n.Name.Space != SFSNS || n.Name.Local != "file-sharing" {
		return StatelessFile{}, fmt.Errorf("xep: invalid stateless file sharing payload")
	}
	var s StatelessFile
	s.Disposition, _ = n.AttrValue("disposition")
	for _, c := range n.Children() {
		if c.Name.Space == FileMetadataNS && c.Name.Local == "file" {
			v, e := ParseFileMetadata(c)
			if e != nil {
				return s, e
			}
			s.File = v
		}
		if c.Name.Local == "sources" {
			for _, u := range c.Children() {
				if u.Name.Space == URLDataNS && u.Name.Local == "url-data" {
					source := URLSource{Headers: map[string]string{}}
					source.URL, _ = u.AttrValue("target")
					for _, h := range u.Children() {
						name, _ := h.AttrValue("name")
						source.Headers[name] = h.Text()
					}
					s.Sources = append(s.Sources, source)
				}
			}
		}
	}
	return s, nil
}

// SIMSReference wraps file metadata and sources in XEP-0385 media-sharing.
type SIMSReference struct {
	URI     string
	File    FileMetadata
	Sources []URLSource
}

func (s SIMSReference) ToNode() xmpp.Node {
	reference := Reference{Type: "data", URI: s.URI}.ToNode()
	sharing := xmpp.NewNode(SIMSNS, "media-sharing")
	sharing.AddChild(s.File.ToNode())
	sources := xmpp.NewNode(SIMSNS, "sources")
	for _, source := range s.Sources {
		sources.AddChild(source.ToNode())
	}
	sharing.AddChild(sources)
	reference.AddChild(sharing)
	return reference
}

type LinkMetadata struct{ URL, Title, Description, Image, SiteName, MediaType string }

func (l LinkMetadata) ToNode() xmpp.Node {
	n := xmpp.NewNode(LinkMetadataNS, "link-metadata")
	for name, value := range map[string]string{"url": l.URL, "title": l.Title, "description": l.Description, "image": l.Image, "site-name": l.SiteName, "media-type": l.MediaType} {
		if value != "" {
			n.AddChild(xmpp.NewTextNode(LinkMetadataNS, name, value))
		}
	}
	return n
}

// OMEMOMediaFragment encodes key and nonce fragments used by XEP-0454.
type OMEMOMediaFragment struct{ Key, IV []byte }

func (f OMEMOMediaFragment) String() string {
	return base64.RawURLEncoding.EncodeToString(append(append([]byte(nil), f.Key...), f.IV...))
}
func ParseOMEMOMediaFragment(value string, keySize int) (OMEMOMediaFragment, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, "#"))
	if err != nil {
		return OMEMOMediaFragment{}, err
	}
	if keySize <= 0 || len(raw) <= keySize {
		return OMEMOMediaFragment{}, fmt.Errorf("xep: invalid OMEMO media fragment length")
	}
	return OMEMOMediaFragment{Key: append([]byte(nil), raw[:keySize]...), IV: append([]byte(nil), raw[keySize:]...)}, nil
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func init() {
	registerSpecialized(50, staticPlugin(50, CommandsNS))
	registerSpecialized(231, staticPlugin(231, BoBNS))
	registerSpecialized(297, staticPlugin(297, ForwardNS))
	registerSpecialized(300, staticPlugin(300, HashesNS))
	registerSpecialized(385, staticPlugin(385, SIMSNS, ReferenceNS, FileMetadataNS))
	registerSpecialized(446, staticPlugin(446, FileMetadataNS))
	registerSpecialized(447, staticPlugin(447, SFSNS, FileMetadataNS, URLDataNS))
	registerSpecialized(454, staticPlugin(454, OMEMOMediaNS))
	registerSpecialized(511, staticPlugin(511, LinkMetadataNS))
}
