package models

import (
	"strings"
	"testing"
)

func validWellKnown() WellKnown {
	def := "bitcoin"
	return WellKnown{
		Key:      "coin",
		NameEn:   "Coin",
		NameEs:   "Moneda",
		Title:    BoardPostItTitle{Text: "{{price}}", Vars: true},
		Resource: "https://api.example.com/price",
		Request: Request{
			Method:  "GET",
			Queries: map[string]string{"ids": "$coin", "key": "$api_key"},
			Headers: map[string]string{"Accept": "application/json"},
		},
		Response: "json",
		Query:    map[string]string{"price": ".price"},
		Rate:     60,
		Parameters: []WellKnownParameter{
			{Key: "$coin", LabelEn: "Coin", LabelEs: "Moneda", Type: ParameterString, Default: &def},
			{Key: "$day", LabelEn: "Day", LabelEs: "Día", Type: ParameterString},
		},
		Outputs: []WellKnownOutput{
			{Key: "price", LabelEn: "Price", LabelEs: "Precio", ExampleEn: 1, ExampleEs: 1},
		},
		SystemSecrets: map[string]SystemSecretName{"$api_key": SystemNasaApiKey},
	}
}

func TestWellKnownValidate(t *testing.T) {
	if err := validWellKnown().Validate(); err != nil {
		t.Fatalf("valid definition refused: %v", err)
	}

	cases := map[string]func(*WellKnown){
		"bad key":               func(wk *WellKnown) { wk.Key = "Bad-Key" },
		"missing name":          func(wk *WellKnown) { wk.NameEs = " " },
		"missing title":         func(wk *WellKnown) { wk.Title.Text = "" },
		"negative rate":         func(wk *WellKnown) { wk.Rate = -1 },
		"non-http resource":     func(wk *WellKnown) { wk.Resource = "file:///etc/passwd" },
		"bad response":          func(wk *WellKnown) { wk.Response = "xml" },
		"non-GET":               func(wk *WellKnown) { wk.Request.Method = "POST" },
		"no query":              func(wk *WellKnown) { wk.Query = nil; wk.Outputs = nil },
		"output/query mismatch": func(wk *WellKnown) { wk.Query = map[string]string{"other": ".x"} },
		"extra output": func(wk *WellKnown) {
			wk.Outputs = append(wk.Outputs, WellKnownOutput{Key: "x", LabelEn: "x", LabelEs: "x"})
		},
		"uppercase placeholder": func(wk *WellKnown) { wk.Request.Queries["ids"] = "$MY_KEY" },
		"undeclared placeholder": func(wk *WellKnown) {
			wk.Request.Headers["Authorization"] = "$token"
		},
		"uppercase parameter": func(wk *WellKnown) { wk.Parameters[0].Key = "$COIN" },
		"repeated parameter":  func(wk *WellKnown) { wk.Parameters[1].Key = "$coin" },
		"bad parameter type":  func(wk *WellKnown) { wk.Parameters[0].Type = "date" },
		"unlabelled parameter": func(wk *WellKnown) {
			wk.Parameters[0].LabelEs = ""
		},
		"unknown system secret": func(wk *WellKnown) {
			wk.SystemSecrets["$api_key"] = "NOT_A_SECRET"
		},
		"system secret shadows a parameter": func(wk *WellKnown) {
			wk.SystemSecrets["$coin"] = SystemNasaApiKey
		},
		"static card with query": func(wk *WellKnown) { wk.Resource = "" },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			wk := validWellKnown()
			mutate(&wk)
			if err := wk.Validate(); err == nil {
				t.Error("accepted")
			}
		})
	}
}

func TestWellKnownValidate_StaticCardOutputsAreParams(t *testing.T) {
	empty := ""
	wk := WellKnown{
		Key: "note", NameEn: "Note", NameEs: "Nota",
		Title:      BoardPostItTitle{Text: "{{text}}", Vars: true},
		Parameters: []WellKnownParameter{{Key: "text", LabelEn: "Text", LabelEs: "Texto", Type: ParameterString, Default: &empty}},
		Outputs:    []WellKnownOutput{{Key: "text", LabelEn: "Text", LabelEs: "Texto"}},
	}
	if err := wk.Validate(); err != nil {
		t.Fatalf("static card refused: %v", err)
	}

	wk.Outputs[0].Key = "other"
	if err := wk.Validate(); err == nil || !strings.Contains(err.Error(), "not a parameter") {
		t.Errorf("err = %v, want an output that is not a parameter refused", err)
	}
}

func TestWellKnownInstantiate(t *testing.T) {
	wk := validWellKnown()

	if _, _, err := wk.Instantiate(nil); err != ErrWellKnownMissingVariable {
		t.Fatalf("err = %v, want the parameter without default required", err)
	}

	postit, title, err := wk.Instantiate(map[string]string{"$day": "today"})
	if err != nil {
		t.Fatalf("instantiate: %v", err)
	}
	if postit.WellKnown != "coin" || postit.Resource.String() != wk.Resource || postit.Rate != 60 {
		t.Errorf("postit = %+v", postit)
	}
	if postit.Params["$coin"] != "bitcoin" || postit.Params["$day"] != "today" {
		t.Errorf("params = %v, want the default and the supplied value", postit.Params)
	}
	if title.Text != "{{price}}" {
		t.Errorf("title = %+v", title)
	}

	// The instance never aliases the definition's maps.
	postit.Request.Queries["ids"] = "changed"
	postit.Query["price"] = "changed"
	if wk.Request.Queries["ids"] != "$coin" || wk.Query["price"] != ".price" {
		t.Error("instantiating shares maps with the definition")
	}
	// Platform credentials never reach the stored card.
	if _, leaked := postit.Params["$api_key"]; leaked {
		t.Error("a system secret placeholder became a param")
	}
}
