package models

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestBodyStrings(t *testing.T) {
	got, err := BodyStrings(`{"origin":"EZE","legs":[{"key":"$IGNAV","n":2}],"nested":{"deep":["x",true,null]}}`)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	want := []string{"$IGNAV", "EZE", "x"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBodyStrings_RefusesWhatIsNotOneJSONDocument(t *testing.T) {
	for _, body := range []string{"", "{not json", `{"a":1} {"b":2}`, "origin=EZE"} {
		if _, err := BodyStrings(body); !errors.Is(err, ErrInvalidRequestBody) {
			t.Errorf("%q: err = %v, want ErrInvalidRequestBody", body, err)
		}
	}
}

func TestReplaceBodyStrings(t *testing.T) {
	body := `{"key":"$TOKEN","list":["$TOKEN","keep"],"count":12345678901234567890,"label":"$OTHER"}`

	got, err := ReplaceBodyStrings(body, map[string]string{"$TOKEN": `se"cr\et`})
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]any
	decoder := json.NewDecoder(strings.NewReader(got))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatalf("the result is not JSON: %v (%s)", err, got)
	}
	if decoded["key"] != `se"cr\et` {
		t.Errorf("key = %v, want the secret as one string", decoded["key"])
	}
	if list := decoded["list"].([]any); list[0] != `se"cr\et` || list[1] != "keep" {
		t.Errorf("list = %v", list)
	}
	if decoded["label"] != "$OTHER" {
		t.Errorf("a name without a value changed: %v", decoded["label"])
	}
	if decoded["count"].(json.Number).String() != "12345678901234567890" {
		t.Errorf("a large number was rounded: %v", decoded["count"])
	}
}

func TestReplaceBodyStrings_OnlyWholeValues(t *testing.T) {
	got, err := ReplaceBodyStrings(`{"note":"pay $TOKEN now"}`, map[string]string{"$TOKEN": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"note":"pay $TOKEN now"}` {
		t.Errorf("got %s, want a value that only contains the name left alone", got)
	}
}
