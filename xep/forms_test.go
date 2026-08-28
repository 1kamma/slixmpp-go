package xep

import (
	"testing"

	"github.com/saret/slixmpp-go/xmpp"
)

func TestFormRoundTrip(t *testing.T) {
	min, max := 1, 2
	form := Form{
		Type:  FormTypeForm,
		Title: "Config",
		Fields: []Field{
			{
				Var:      "mode",
				Type:     FieldListSingle,
				Required: true,
				Values:   []string{"a"},
				Options:  []Option{{Value: "a"}, {Value: "b"}},
				Validation: &Validation{
					Method:  ValidationBasic,
					ListMin: &min,
					ListMax: &max,
				},
				Media: &Media{
					Width: 10,
					URIs:  []URI{{Type: "image/png", Value: "cid:x"}},
				},
			},
		},
	}
	node := form.ToNode()
	parsed, err := ParseForm(node)
	if err != nil {
		t.Fatal(err)
	}
	if err := parsed.Validate(); err != nil {
		t.Fatal(err)
	}
	raw := node.MustXML()
	if _, err := xmpp.ParseNode([]byte(raw)); err != nil {
		t.Fatal(err)
	}
}
