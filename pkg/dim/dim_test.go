package dim

import (
	"reflect"
	"testing"
)

func TestStructToKVList(t *testing.T) {
	type dimRR struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Zone      string `json:"zone,omitempty"`
		View      string `json:"views,omitempty"`
		Ttl       int    `json:"ttl,omitempty"`
		Comment   string `json:"comment,omitempty"`
		Overwrite bool   `json:"overwrite,omitempty"`
		Ip        string `json:"ip,omitempty"`
	}

	tests := []struct {
		input dimRR
		wants []map[string]interface{}
	}{
		{
			input: dimRR{Name: "test01", Type: "TXT", Zone: "example.com", Ip: "10.10.10.10", Overwrite: true},
			wants: []map[string]interface{}{
				{"name": "test01"},
				{"type": "TXT"},
				{"zone": "example.com"},
				{"ip": "10.10.10.10"},
				{"overwrite": true},
			},
		},
	}

	for i := 0; i < len(tests); i++ {
		if got := structToKVList(reflect.ValueOf(tests[i].input)); !mySliceEqu(got, tests[i].wants) {
			t.Errorf("structToKVList(%#v) = %q ; wants = %q", tests[i].input, got, tests[i].wants)
		}
	}

}

func mySliceEqu(x, y []map[string]interface{}) bool {
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if !myMapEqu(x[i], y[i]) {
			return false
		}
	}
	return true
}

func myMapEqu(x, y map[string]interface{}) bool {
	if len(x) != len(y) {
		return false
	}
	for k, xv := range x {
		if yv, ok := y[k]; !ok || yv != xv {
			return false
		}
	}
	return true
}
