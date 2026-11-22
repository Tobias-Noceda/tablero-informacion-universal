package postits

import (
	"net/url"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
)

// Run longer with: go test -run '^$' -fuzz FuzzValidateOneOff -fuzztime 30s ./common/services/postits/

// Whatever a card of one's own asks for, what is accepted is a GET or a POST
// to an http(s) URL, a GET carries no body, a POST at most MaxRequestBody
// bytes of one JSON document, and checking it again changes nothing.
func FuzzValidateOneOff(f *testing.F) {
	f.Add("https://api.example.com/v1?x=1", "post", `{"token":"$KEY"}`, "json", "out", ".value", "X-Key")
	f.Add("http://example.com", "GET", "ignored", "html", "title", "h1", "")
	f.Add("ftp://example.com", "DELETE", "", "xml", "", "", " ")
	f.Add("https://", "POST", "{", "json", "a", ".", "A")
	f.Add("javascript:alert(1)", "", "", "json", "a", ".", "")
	f.Fuzz(func(t *testing.T, rawURL, method, body, response, outKey, outQuery, header string) {
		resource, err := url.Parse(rawURL)
		if err != nil {
			return
		}
		card := &models.PostIts{
			Resource: resource,
			Request:  models.Request{Method: method, Body: body, Headers: map[string]string{header: "v"}},
			Response: response,
			Query:    map[string]string{outKey: outQuery},
		}
		if validateOneOff(card) != nil {
			return
		}

		if card.Resource.Scheme != "http" && card.Resource.Scheme != "https" || card.Resource.Host == "" {
			t.Fatalf("accepted %q", rawURL)
		}
		switch card.Request.Method {
		case "GET":
			if card.Request.Body != "" {
				t.Fatalf("a GET kept a body: %q", card.Request.Body)
			}
		case "POST":
			if len(card.Request.Body) > models.MaxRequestBody {
				t.Fatalf("accepted a body of %d bytes", len(card.Request.Body))
			}
			if card.Request.Body != "" {
				if _, err := models.BodyStrings(card.Request.Body); err != nil {
					t.Fatalf("accepted a body that is not JSON: %q", card.Request.Body)
				}
			}
		default:
			t.Fatalf("accepted method %q", card.Request.Method)
		}
		if card.Response != "json" && card.Response != "html" {
			t.Fatalf("accepted response %q", card.Response)
		}

		again := *card
		if err := validateOneOff(&again); err != nil ||
			again.Request.Method != card.Request.Method || again.Request.Body != card.Request.Body {
			t.Fatalf("checking it again gave %+v, %v", again.Request, err)
		}
	})
}
