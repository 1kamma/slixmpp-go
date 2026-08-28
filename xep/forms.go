package xep

import (
	"fmt"
	"github.com/saret/slixmpp-go/xmpp"
	"regexp"
	"strconv"
	"strings"
)

const (
	DataFormsNS      = "jabber:x:data"
	DataValidationNS = "http://jabber.org/protocol/xdata-validate"
	DataMediaNS      = "urn:xmpp:media-element"
)

type FormType string

const (
	FormTypeForm   FormType = "form"
	FormTypeSubmit FormType = "submit"
	FormTypeCancel FormType = "cancel"
	FormTypeResult FormType = "result"
)

type FieldType string

const (
	FieldBoolean     FieldType = "boolean"
	FieldFixed       FieldType = "fixed"
	FieldHidden      FieldType = "hidden"
	FieldJIDMulti    FieldType = "jid-multi"
	FieldJIDSingle   FieldType = "jid-single"
	FieldListMulti   FieldType = "list-multi"
	FieldListSingle  FieldType = "list-single"
	FieldTextMulti   FieldType = "text-multi"
	FieldTextPrivate FieldType = "text-private"
	FieldTextSingle  FieldType = "text-single"
)

type Option struct{ Label, Value string }
type URI struct{ Type, Value string }
type Media struct {
	Height, Width int
	URIs          []URI
}
type ValidationMethod string

const (
	ValidationBasic ValidationMethod = "basic"
	ValidationOpen  ValidationMethod = "open"
	ValidationRange ValidationMethod = "range"
	ValidationRegex ValidationMethod = "regex"
)

type Validation struct {
	Datatype, Regex, Min, Max string
	Method                    ValidationMethod
	ListMin, ListMax          *int
}
type Field struct {
	Var, Label, Desc string
	Type             FieldType
	Required         bool
	Values           []string
	Options          []Option
	Validation       *Validation
	Media            *Media
}
type Reported struct{ Fields []Field }
type Item struct{ Fields []Field }
type Form struct {
	Type         FormType
	Title        string
	Instructions []string
	Fields       []Field
	Reported     *Reported
	Items        []Item
}

func (f Form) Field(name string) *Field {
	for i := range f.Fields {
		if f.Fields[i].Var == name {
			return &f.Fields[i]
		}
	}
	return nil
}
func (f *Form) Set(name string, values ...string) {
	for i := range f.Fields {
		if f.Fields[i].Var == name {
			f.Fields[i].Values = append([]string(nil), values...)
			return
		}
	}
	f.Fields = append(f.Fields, Field{Var: name, Type: FieldTextSingle, Values: append([]string(nil), values...)})
}
func (f Form) ToNode() xmpp.Node {
	n := xmpp.NewNode(DataFormsNS, "x")
	if f.Type != "" {
		n.SetAttr("type", string(f.Type))
	}
	if f.Title != "" {
		n.AddChild(xmpp.NewTextNode(DataFormsNS, "title", f.Title))
	}
	for _, v := range f.Instructions {
		n.AddChild(xmpp.NewTextNode(DataFormsNS, "instructions", v))
	}
	for _, field := range f.Fields {
		n.AddChild(fieldToNode(field))
	}
	if f.Reported != nil {
		r := xmpp.NewNode(DataFormsNS, "reported")
		for _, field := range f.Reported.Fields {
			r.AddChild(fieldToNode(field))
		}
		n.AddChild(r)
	}
	for _, item := range f.Items {
		i := xmpp.NewNode(DataFormsNS, "item")
		for _, field := range item.Fields {
			i.AddChild(fieldToNode(field))
		}
		n.AddChild(i)
	}
	return n
}
func fieldToNode(f Field) xmpp.Node {
	n := xmpp.NewNode(DataFormsNS, "field")
	if f.Var != "" {
		n.SetAttr("var", f.Var)
	}
	if f.Label != "" {
		n.SetAttr("label", f.Label)
	}
	if f.Type != "" {
		n.SetAttr("type", string(f.Type))
	}
	if f.Desc != "" {
		n.AddChild(xmpp.NewTextNode(DataFormsNS, "desc", f.Desc))
	}
	if f.Required {
		n.AddChild(xmpp.NewNode(DataFormsNS, "required"))
	}
	for _, v := range f.Values {
		n.AddChild(xmpp.NewTextNode(DataFormsNS, "value", v))
	}
	for _, o := range f.Options {
		node := xmpp.NewNode(DataFormsNS, "option")
		if o.Label != "" {
			node.SetAttr("label", o.Label)
		}
		node.AddChild(xmpp.NewTextNode(DataFormsNS, "value", o.Value))
		n.AddChild(node)
	}
	if f.Validation != nil {
		n.AddChild(validationToNode(*f.Validation))
	}
	if f.Media != nil {
		n.AddChild(mediaToNode(*f.Media))
	}
	return n
}
func validationToNode(v Validation) xmpp.Node {
	n := xmpp.NewNode(DataValidationNS, "validate")
	if v.Datatype != "" {
		n.SetAttr("datatype", v.Datatype)
	}
	method := v.Method
	if method == "" {
		method = ValidationBasic
	}
	switch method {
	case ValidationBasic:
		n.AddChild(xmpp.NewNode(DataValidationNS, "basic"))
	case ValidationOpen:
		n.AddChild(xmpp.NewNode(DataValidationNS, "open"))
	case ValidationRange:
		r := xmpp.NewNode(DataValidationNS, "range")
		if v.Min != "" {
			r.SetAttr("min", v.Min)
		}
		if v.Max != "" {
			r.SetAttr("max", v.Max)
		}
		n.AddChild(r)
	case ValidationRegex:
		n.AddChild(xmpp.NewTextNode(DataValidationNS, "regex", v.Regex))
	}
	if v.ListMin != nil || v.ListMax != nil {
		l := xmpp.NewNode(DataValidationNS, "list-range")
		if v.ListMin != nil {
			l.SetAttr("min", strconv.Itoa(*v.ListMin))
		}
		if v.ListMax != nil {
			l.SetAttr("max", strconv.Itoa(*v.ListMax))
		}
		n.AddChild(l)
	}
	return n
}
func mediaToNode(m Media) xmpp.Node {
	n := xmpp.NewNode(DataMediaNS, "media")
	if m.Height > 0 {
		n.SetAttr("height", strconv.Itoa(m.Height))
	}
	if m.Width > 0 {
		n.SetAttr("width", strconv.Itoa(m.Width))
	}
	for _, u := range m.URIs {
		v := xmpp.NewTextNode(DataMediaNS, "uri", u.Value)
		if u.Type != "" {
			v.SetAttr("type", u.Type)
		}
		n.AddChild(v)
	}
	return n
}
func ParseForm(n xmpp.Node) (Form, error) {
	if n.Name.Local != "x" || n.Name.Space != DataFormsNS {
		return Form{}, fmt.Errorf("xep: expected {%s}x", DataFormsNS)
	}
	var f Form
	if v, ok := n.AttrValue("type"); ok {
		f.Type = FormType(v)
	}
	for _, c := range n.Children() {
		switch c.Name.Local {
		case "title":
			f.Title = c.Text()
		case "instructions":
			f.Instructions = append(f.Instructions, c.Text())
		case "field":
			field, err := parseField(c)
			if err != nil {
				return Form{}, err
			}
			f.Fields = append(f.Fields, field)
		case "reported":
			r := &Reported{}
			for _, fc := range c.Children() {
				if fc.Name.Local == "field" {
					field, err := parseField(fc)
					if err != nil {
						return Form{}, err
					}
					r.Fields = append(r.Fields, field)
				}
			}
			f.Reported = r
		case "item":
			item := Item{}
			for _, fc := range c.Children() {
				if fc.Name.Local == "field" {
					field, err := parseField(fc)
					if err != nil {
						return Form{}, err
					}
					item.Fields = append(item.Fields, field)
				}
			}
			f.Items = append(f.Items, item)
		}
	}
	return f, nil
}
func parseField(n xmpp.Node) (Field, error) {
	var f Field
	f.Var, _ = n.AttrValue("var")
	f.Label, _ = n.AttrValue("label")
	if v, ok := n.AttrValue("type"); ok {
		f.Type = FieldType(v)
	}
	for _, c := range n.Children() {
		switch c.Name.Local {
		case "desc":
			f.Desc = c.Text()
		case "required":
			f.Required = true
		case "value":
			f.Values = append(f.Values, c.Text())
		case "option":
			label, _ := c.AttrValue("label")
			f.Options = append(f.Options, Option{Label: label, Value: c.ChildText(DataFormsNS, "value")})
		case "validate":
			v := parseValidation(c)
			f.Validation = &v
		case "media":
			m := parseMedia(c)
			f.Media = &m
		}
	}
	return f, nil
}
func parseValidation(n xmpp.Node) Validation {
	v := Validation{}
	v.Datatype, _ = n.AttrValue("datatype")
	for _, c := range n.Children() {
		switch c.Name.Local {
		case "basic":
			v.Method = ValidationBasic
		case "open":
			v.Method = ValidationOpen
		case "range":
			v.Method = ValidationRange
			v.Min, _ = c.AttrValue("min")
			v.Max, _ = c.AttrValue("max")
		case "regex":
			v.Method = ValidationRegex
			v.Regex = c.Text()
		case "list-range":
			if x, ok := c.AttrValue("min"); ok {
				if i, e := strconv.Atoi(x); e == nil {
					v.ListMin = &i
				}
			}
			if x, ok := c.AttrValue("max"); ok {
				if i, e := strconv.Atoi(x); e == nil {
					v.ListMax = &i
				}
			}
		}
	}
	return v
}
func parseMedia(n xmpp.Node) Media {
	m := Media{}
	if v, ok := n.AttrValue("height"); ok {
		m.Height, _ = strconv.Atoi(v)
	}
	if v, ok := n.AttrValue("width"); ok {
		m.Width, _ = strconv.Atoi(v)
	}
	for _, c := range n.Children() {
		if c.Name.Local == "uri" {
			kind, _ := c.AttrValue("type")
			m.URIs = append(m.URIs, URI{Type: kind, Value: c.Text()})
		}
	}
	return m
}

// Validate checks required fields, field cardinality, options, regexes, and list ranges.
func (f Form) Validate() error {
	for _, field := range f.Fields {
		if err := field.Validate(); err != nil {
			return fmt.Errorf("field %q: %w", field.Var, err)
		}
	}
	return nil
}
func (f Field) Validate() error {
	if f.Required && len(f.Values) == 0 {
		return fmt.Errorf("value is required")
	}
	single := f.Type == FieldBoolean || f.Type == FieldJIDSingle || f.Type == FieldListSingle || f.Type == FieldTextPrivate || f.Type == FieldTextSingle
	if single && len(f.Values) > 1 {
		return fmt.Errorf("field type %s accepts one value", f.Type)
	}
	if f.Type == FieldBoolean && len(f.Values) == 1 {
		v := strings.ToLower(f.Values[0])
		if v != "1" && v != "0" && v != "true" && v != "false" {
			return fmt.Errorf("invalid boolean %q", f.Values[0])
		}
	}
	if f.Type == FieldListSingle || f.Type == FieldListMulti {
		allowed := map[string]bool{}
		for _, o := range f.Options {
			allowed[o.Value] = true
		}
		if len(allowed) > 0 && f.Validation != nil && f.Validation.Method != ValidationOpen {
			for _, v := range f.Values {
				if !allowed[v] {
					return fmt.Errorf("value %q is not an option", v)
				}
			}
		}
	}
	if v := f.Validation; v != nil {
		if v.ListMin != nil && len(f.Values) < *v.ListMin {
			return fmt.Errorf("requires at least %d values", *v.ListMin)
		}
		if v.ListMax != nil && len(f.Values) > *v.ListMax {
			return fmt.Errorf("allows at most %d values", *v.ListMax)
		}
		if v.Method == ValidationRegex {
			re, err := regexp.Compile(v.Regex)
			if err != nil {
				return fmt.Errorf("invalid validation regex: %w", err)
			}
			for _, value := range f.Values {
				if !re.MatchString(value) {
					return fmt.Errorf("value %q does not match validation regex", value)
				}
			}
		}
	}
	return nil
}

type formsPlugin struct{ xmpp.BasicPlugin }

func newFormsPlugin() xmpp.Plugin {
	return &formsPlugin{BasicPlugin: xmpp.BasicPlugin{PluginName: "xep_0004", PluginDescription: "XEP-0004 Data Forms", PluginFeatures: []string{DataFormsNS, DataValidationNS, DataMediaNS}}}
}
func init() {
	registerSpecialized(4, newFormsPlugin)
	registerSpecialized(122, staticPlugin(122, DataValidationNS))
	registerSpecialized(221, staticPlugin(221, DataMediaNS))
}
