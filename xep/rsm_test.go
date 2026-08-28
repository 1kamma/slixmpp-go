package xep

import "testing"

func TestRSMRoundTrip(t *testing.T) {
	r := RSM{BeforeSet: true, Max: Int(20), First: "a", FirstIndex: Int(2), Count: Int(9)}
	got, err := ParseRSM(r.ToNode())
	if err != nil {
		t.Fatal(err)
	}
	if !got.BeforeSet || got.Max == nil || *got.Max != 20 || got.FirstIndex == nil || *got.FirstIndex != 2 {
		t.Fatalf("%#v", got)
	}
}
