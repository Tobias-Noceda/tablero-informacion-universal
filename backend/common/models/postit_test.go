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

func TestPostItsUnmarshalJSON_RoundTrips(t *testing.T) {
	card := PostIts{
		Resource: &url.URL{Scheme: "https", Host: "example.com", Path: "/fact", RawQuery: "limit=2"},
		Rate:     60,
		Query:    map[string]string{"fact": ".fact"},
	}
	raw, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}

	var got PostIts
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	if got.Resource == nil || got.Resource.String() != "https://example.com/fact?limit=2" {
		t.Errorf("resource = %v, want the URL back", got.Resource)
	}
	if got.Rate != 60 || got.Query["fact"] != ".fact" {
		t.Errorf("other fields lost: %+v", got)
	}
}

func TestPostItsUnmarshalJSON_EmptyOrBadResource(t *testing.T) {
	for _, raw := range []string{`{"resource":null}`, `{"resource":""}`, `{}`} {
		got := PostIts{Resource: &url.URL{Host: "stale"}}
		if err := json.Unmarshal([]byte(raw), &got); err != nil || got.Resource != nil {
			t.Errorf("%s: resource = %v, err = %v, want nil", raw, got.Resource, err)
		}
	}

	var got PostIts
	if err := json.Unmarshal([]byte(`{"resource":"https://example.com/%zz"}`), &got); err == nil {
		t.Error("a malformed resource was accepted")
	}
}
