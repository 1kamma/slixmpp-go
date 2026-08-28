package xmpp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Part is one ordered item in an XML element. Exactly one of Text or Child is
// normally populated.
type Part struct {
	Text  string
	Child *Node
}

// Node is a namespace-aware XML element retaining ordered mixed content.
type Node struct {
	Name  xml.Name
	Attr  []xml.Attr
	Parts []Part
}

// NewNode constructs an XML node using a namespace URI and local name.
func NewNode(namespace, local string) Node {
	return Node{Name: xml.Name{Space: namespace, Local: local}}
}

// NewTextNode constructs a node containing value.
func NewTextNode(namespace, local, value string) Node {
	n := NewNode(namespace, local)
	n.AddText(value)
	return n
}

// SetAttr sets or replaces an unqualified attribute.
func (n *Node) SetAttr(name, value string) { n.SetAttrNS("", name, value) }

// SetAttrNS sets or replaces a namespaced attribute.
func (n *Node) SetAttrNS(namespace, name, value string) {
	for i := range n.Attr {
		if n.Attr[i].Name.Space == namespace && n.Attr[i].Name.Local == name {
			n.Attr[i].Value = value
			return
		}
	}
	n.Attr = append(n.Attr, xml.Attr{Name: xml.Name{Space: namespace, Local: name}, Value: value})
}

// AttrValue returns an unqualified attribute and whether it exists.
func (n Node) AttrValue(name string) (string, bool) { return n.AttrValueNS("", name) }

// AttrValueNS returns a namespaced attribute and whether it exists.
func (n Node) AttrValueNS(namespace, name string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Name.Space == namespace && attr.Name.Local == name {
			return attr.Value, true
		}
	}
	return "", false
}

// AddText appends character data.
func (n *Node) AddText(value string) {
	if value == "" {
		return
	}
	if len(n.Parts) > 0 && n.Parts[len(n.Parts)-1].Child == nil {
		n.Parts[len(n.Parts)-1].Text += value
		return
	}
	n.Parts = append(n.Parts, Part{Text: value})
}

// AddChild appends a child element.
func (n *Node) AddChild(child Node) { copy := child; n.Parts = append(n.Parts, Part{Child: &copy}) }

// Text returns concatenated direct character data.
func (n Node) Text() string {
	var b strings.Builder
	for _, part := range n.Parts {
		if part.Child == nil {
			b.WriteString(part.Text)
		}
	}
	return b.String()
}

// Children returns deep copies of direct child elements.
func (n Node) Children() []Node {
	out := make([]Node, 0, len(n.Parts))
	for _, part := range n.Parts {
		if part.Child != nil {
			out = append(out, part.Child.Clone())
		}
	}
	return out
}

// Child returns the first child matching local name and, when non-empty,
// namespace.
func (n Node) Child(namespace, local string) *Node {
	for _, part := range n.Parts {
		if part.Child == nil || part.Child.Name.Local != local {
			continue
		}
		if namespace == "" || part.Child.Name.Space == namespace {
			copy := part.Child.Clone()
			return &copy
		}
	}
	return nil
}

// ChildText returns the direct child text or an empty string.
func (n Node) ChildText(namespace, local string) string {
	child := n.Child(namespace, local)
	if child == nil {
		return ""
	}
	return child.Text()
}

// FindPath follows local-name-only child steps.
func (n Node) FindPath(path ...string) *Node {
	current := n.Clone()
	for _, local := range path {
		next := current.Child("", local)
		if next == nil {
			return nil
		}
		current = next.Clone()
	}
	return &current
}

// Clone returns a deep copy.
func (n Node) Clone() Node {
	out := Node{Name: n.Name, Attr: append([]xml.Attr(nil), n.Attr...)}
	for _, part := range n.Parts {
		if part.Child != nil {
			child := part.Child.Clone()
			out.Parts = append(out.Parts, Part{Child: &child})
		} else {
			out.Parts = append(out.Parts, Part{Text: part.Text})
		}
	}
	return out
}

// XML serializes n without an XML declaration.
func (n Node) XML() (string, error) {
	var b bytes.Buffer
	enc := xml.NewEncoder(&b)
	if err := enc.Encode(n); err != nil {
		return "", err
	}
	if err := enc.Flush(); err != nil {
		return "", err
	}
	return b.String(), nil
}

// MustXML serializes n and panics on error.
func (n Node) MustXML() string {
	value, err := n.XML()
	if err != nil {
		panic(err)
	}
	return value
}

// ParseNode parses exactly one XML element.
func ParseNode(data []byte) (Node, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := dec.Token()
		if err != nil {
			return Node{}, err
		}
		if start, ok := token.(xml.StartElement); ok {
			var node Node
			if err := dec.DecodeElement(&node, &start); err != nil {
				return Node{}, err
			}
			return node, nil
		}
	}
}

// MarshalXML implements xml.Marshaler.
func (n Node) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	if n.Name.Local == "" {
		return fmt.Errorf("xmpp: cannot marshal node without a local name")
	}
	start := xml.StartElement{Name: n.Name, Attr: append([]xml.Attr(nil), n.Attr...)}
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	for _, part := range n.Parts {
		if part.Child != nil {
			if err := enc.Encode(part.Child); err != nil {
				return err
			}
		} else if part.Text != "" {
			if err := enc.EncodeToken(xml.CharData(part.Text)); err != nil {
				return err
			}
		}
	}
	return enc.EncodeToken(start.End())
}

// UnmarshalXML implements xml.Unmarshaler while preserving mixed content.
func (n *Node) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	n.Name, n.Attr, n.Parts = start.Name, append(n.Attr[:0], start.Attr...), n.Parts[:0]
	for {
		token, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		switch value := token.(type) {
		case xml.StartElement:
			var child Node
			if err := dec.DecodeElement(&child, &value); err != nil {
				return err
			}
			n.AddChild(child)
		case xml.CharData:
			n.AddText(string(value))
		case xml.EndElement:
			if value.Name == start.Name {
				return nil
			}
		}
	}
}
