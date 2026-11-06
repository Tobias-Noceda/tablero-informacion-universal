package models

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestPostItsMarshalJSON_ResourceIsAString(t *testing.T) {
	card := PostIts{Resource: &url.URL{Scheme: "https", Host: "example.com", Path: "/fact"}, Rate: 60}
	raw, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["resource"] != "https://example.com/fact" {
		t.Errorf("resource = %#v, want the URL", got["resource"])
	}
	if got["rate"] != float64(60) {
		t.Errorf("rate = %#v, other fields must stay", got["rate"])
	}

	raw, _ = json.Marshal(PostIts{})
	json.Unmarshal(raw, &got)
	if got["resource"] != nil {
		t.Errorf("resource = %#v, want null", got["resource"])
	}
}
