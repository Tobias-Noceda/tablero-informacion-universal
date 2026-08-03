package executer

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func TestHtml_Simple(t *testing.T) {
	e := New()

	postit := &models.PostIts{
		Id:     uuid.Max,
		Board:  uuid.Max,
		Params: nil,
		Resource: &url.URL{
			Scheme: "https",
			Host:   "example.com",
		},
		Request: models.Request{
			Method: "GET",
		},
		Response: "html",
		Query: map[string]string{
			"title": "h1",
			"link":  "a",
		},
	}

	data, _ := e.Execute(postit)
	bytes, _ := json.MarshalIndent(data, "", "  ")
	println(string(bytes))
}
