package models

import (
	"strings"
	"testing"
)

func catFact() CustomWellKnown {
	return CustomWellKnown{
		Key:      NewCustomWellKnownKey(),
		Scope:    CustomWellKnownScope{Kind: CustomScopeUser, Owner: "ana"},
		Name:     "Cat fact",
		Title:    BoardPostItTitle{Text: "{{fact}}", Vars: true},
		Resource: "https://catfact.ninja/fact",
		Request:  Request{Method: "GET", Queries: map[string]string{"max_length": "$max"}},
		Response: "json",
		Query:    map[string]string{"fact": ".fact"},
		Rate:     60,
		Parameters: []CustomWellKnownParameter{
			{Key: "$max", Label: "Largo máximo", Type: ParameterNumber},
		},
		Outputs: []CustomWellKnownOutput{{Key: "fact", Label: "Dato", Example: "Cats sleep"}},
	}
}

func TestCustomWellKnown_AsWellKnown(t *testing.T) {
	c := catFact()
	wk := c.AsWellKnown()

	if wk.Key != c.Key || wk.NameEn != "Cat fact" || wk.NameEs != "Cat fact" {
		t.Errorf("names: %+v", wk)
	}
	if p := wk.Parameters[0]; p.LabelEn != "Largo máximo" || p.LabelEs != "Largo máximo" || p.Type != ParameterNumber {
		t.Errorf("parameter: %+v", p)
	}
	if o := wk.Outputs[0]; o.LabelEs != "Dato" || o.ExampleEn != "Cats sleep" || o.ExampleEs != "Cats sleep" {
		t.Errorf("output: %+v", o)
	}
	if len(wk.SystemSecrets) != 0 {
		t.Error("a custom type never uses platform credentials")
	}

	if wk.TitleEn != "{{fact}}" || wk.TitleEs != "{{fact}}" {
		t.Errorf("titles: %q / %q", wk.TitleEn, wk.TitleEs)
	}

	postit, title, err := wk.Instantiate(map[string]string{"$max": "80"})
	if err != nil {
		t.Fatal(err)
	}
	if title.Text != "{{fact}}" {
		t.Errorf("default title = %q", title.Text)
	}
	if postit.WellKnown != c.Key || postit.Params["$max"] != "80" || postit.Resource.String() != c.Resource {
		t.Errorf("instantiated: %+v", postit)
	}
}

func TestCustomWellKnown_Validate(t *testing.T) {
	if err := catFact().Validate(); err != nil {
		t.Fatalf("valid: %v", err)
	}

	cases := map[string]struct {
		edit func(*CustomWellKnown)
		want string
	}{
		"client key":       {func(c *CustomWellKnown) { c.Key = "cat_fact" }, "generated"},
		"bad scope":        {func(c *CustomWellKnown) { c.Scope.Kind = "system" }, "scope"},
		"no owner":         {func(c *CustomWellKnown) { c.Scope.Owner = "" }, "scope"},
		"no resource":      {func(c *CustomWellKnown) { c.Resource = "" }, "resource"},
		"not http":         {func(c *CustomWellKnown) { c.Resource = "file:///etc/passwd" }, "http"},
		"no name":          {func(c *CustomWellKnown) { c.Name = " " }, "name is required"},
		"long name":        {func(c *CustomWellKnown) { c.Name = strings.Repeat("a", 65) }, "name"},
		"no label":         {func(c *CustomWellKnown) { c.Parameters[0].Label = "" }, "needs a label"},
		"undeclared param": {func(c *CustomWellKnown) { c.Request.Queries["x"] = "$other" }, "not a parameter"},
		"uppercase param":  {func(c *CustomWellKnown) { c.Parameters[0].Key = "$MAX" }, "lowercase"},
		"output mismatch":  {func(c *CustomWellKnown) { c.Outputs[0].Key = "other" }, "query key"},
		"too many outputs": {func(c *CustomWellKnown) { c.Outputs = make([]CustomWellKnownOutput, 21) }, "at most"},
	}
	for name, tc := range cases {
		c := catFact()
		tc.edit(&c)
		err := c.Validate()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want it to mention %q", name, err, tc.want)
		}
	}
}

func TestWellKnown_RefusesCustomKeys(t *testing.T) {
	wk := catFact().AsWellKnown()
	wk.custom = false
	if err := wk.Validate(); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Errorf("catalogue key %q: err = %v", wk.Key, err)
	}
}
