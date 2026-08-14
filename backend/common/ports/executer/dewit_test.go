package executer

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
)

func nopCloser(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

func TestParse_DetectsTypes(t *testing.T) {
	e := New()

	obj, err := e.parse(nopCloser(`{"a":1}`))
	if err != nil {
		t.Fatalf("object parse: %v", err)
	}
	if _, ok := obj.(map[string]any); !ok {
		t.Errorf("object parsed as %T, want map", obj)
	}

	arr, err := e.parse(nopCloser(`[1,2,3]`))
	if err != nil {
		t.Fatalf("array parse: %v", err)
	}
	if _, ok := arr.([]any); !ok {
		t.Errorf("array parsed as %T, want slice", arr)
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	e := New()
	if _, err := e.parse(nopCloser(`{not json`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestQuery_ObjectProjection(t *testing.T) {
	e := New()
	data := map[string]any{"compra": 1000.0, "venta": 1050.0, "extra": "x"}

	res, err := e.query(data, "{ compra: .compra, venta: .venta }")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map", res)
	}
	if m["compra"] != 1000.0 || m["venta"] != 1050.0 {
		t.Errorf("got %v, want compra=1000 venta=1050", m)
	}
}

// Reproduces the dolar_oficial bug: an object projection over an array payload
// must surface a gojq error rather than silently succeeding.
func TestQuery_ObjectQueryOverArrayErrors(t *testing.T) {
	e := New()
	data := []any{
		map[string]any{"casa": "oficial", "compra": 1000.0},
	}

	if _, err := e.query(data, "{ compra: .compra }"); err == nil {
		t.Fatal("expected error when projecting an object query over an array")
	}
}

// The array-aware alternative query should work against the same array payload.
func TestQuery_ArrayFilterSelectsElement(t *testing.T) {
	e := New()
	data := []any{
		map[string]any{"casa": "oficial", "compra": 1000.0, "venta": 1050.0},
		map[string]any{"casa": "blue", "compra": 1200.0, "venta": 1250.0},
	}

	res, err := e.query(data, `.[] | select(.casa == "oficial") | { compra, venta }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map", res)
	}
	if m["compra"] != 1000.0 {
		t.Errorf("got %v, want oficial compra=1000", m)
	}
}

func TestQuery_NoResults(t *testing.T) {
	e := New()
	data := []any{}

	if _, err := e.query(data, ".[]"); err == nil {
		t.Fatal("expected 'no results' error for empty iteration")
	}
}

func TestQuery_InvalidSyntax(t *testing.T) {
	e := New()
	if _, err := e.query(map[string]any{}, "{ broken"); err == nil {
		t.Fatal("expected parse error for invalid query syntax")
	}
}

func TestPopulate_ReplacesMatchingParams(t *testing.T) {
	e := New()
	params := map[string]string{"$kw": "concert", "$size": "1"}
	out := map[string]string{"keyword": "$kw", "size": "$size", "fixed": "stays"}

	if err := e.populate(params, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["keyword"] != "concert" {
		t.Errorf("keyword = %q, want concert", out["keyword"])
	}
	if out["size"] != "1" {
		t.Errorf("size = %q, want 1", out["size"])
	}
	// "fixed" maps to "stays", which is not a known param name, so it is left as-is.
	if out["fixed"] != "stays" {
		t.Errorf("fixed = %q, want stays (unmatched values are untouched)", out["fixed"])
	}
}

func TestExecute_NoResourceReturnsParams(t *testing.T) {
	e := New()
	params := map[string]string{"text": "hello"}

	res, err := e.Execute(&models.PostIts{Params: params})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := res.(map[string]string)
	if !ok || got["text"] != "hello" {
		t.Errorf("got %#v, want the raw params back when Resource is nil", res)
	}
}

func TestExecute_EndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"compra":1000,"venta":1050,"casa":"oficial"}`)
	}))
	defer srv.Close()

	resource, _ := url.Parse(srv.URL)
	e := New()
	postit := &models.PostIts{
		Resource: resource,
		Request:  models.Request{Method: http.MethodGet},
		Query:    "{ compra: .compra, venta: .venta }",
	}

	res, err := e.Execute(postit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map", res)
	}
	if m["compra"] != 1000.0 || m["venta"] != 1050.0 {
		t.Errorf("got %v, want compra=1000 venta=1050", m)
	}
}

func TestExecute_PopulatesQueryParamsIntoRequest(t *testing.T) {
	var gotKeyword string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKeyword = r.URL.Query().Get("keyword")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	resource, _ := url.Parse(srv.URL)
	e := New()
	postit := &models.PostIts{
		Resource: resource,
		Params:   map[string]string{"$kw": "jazz"},
		Request: models.Request{
			Method:  http.MethodGet,
			Queries: map[string]string{"keyword": "$kw"},
		},
		Query: ".ok",
	}

	if _, err := e.Execute(postit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKeyword != "jazz" {
		t.Errorf("server saw keyword=%q, want jazz (param substitution failed)", gotKeyword)
	}
}

func TestExecute_Non200Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	resource, _ := url.Parse(srv.URL)
	e := New()
	postit := &models.PostIts{
		Resource: resource,
		Request:  models.Request{Method: http.MethodGet},
		Query:    ".",
	}

	if _, err := e.Execute(postit); err == nil {
		t.Fatal("expected error when the resource returns a non-200 status")
	}
}

func TestExecute_DoesNotMutateSharedResource(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	// Well-knowns hand the same *url.URL to every post-it built from them.
	resource, _ := url.Parse(srv.URL)
	e := New()

	for _, coin := range []string{"bitcoin", "ethereum"} {
		postit := &models.PostIts{
			Resource: resource,
			Request: models.Request{
				Method:  http.MethodGet,
				Queries: map[string]string{"ids": "$coin"},
			},
			Params: map[string]string{"$coin": coin},
			Query:  ".",
		}
		if _, err := e.Execute(postit); err != nil {
			t.Fatalf("%s: unexpected error: %v", coin, err)
		}
	}

	if resource.RawQuery != "" {
		t.Errorf("shared resource was mutated: RawQuery = %q, want empty", resource.RawQuery)
	}
	want := []string{"ids=bitcoin", "ids=ethereum"}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("request %d: server saw %q, want %q", i+1, seen[i], want[i])
		}
	}
}
