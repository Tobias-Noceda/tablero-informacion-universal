package models

import (
	"encoding/json"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestCardStyle_Normalized(t *testing.T) {
	tests := []struct {
		name    string
		give    CardStyle
		want    CardStyle
		wantErr bool
	}{
		{name: "the zero style is the usual card", give: CardStyle{}, want: CardStyle{}},
		{
			name: "every field set",
			give: CardStyle{Color: StrandColorBlue, Border: CardBorderDashed, Width: 320, Height: 180},
			want: CardStyle{Color: StrandColorBlue, Border: CardBorderDashed, Width: 320, Height: 180},
		},
		{name: "a side is rounded to whole pixels", give: CardStyle{Width: 240.6, Height: 99.4}, want: CardStyle{Width: 241, Height: 99}},
		{name: "one side alone", give: CardStyle{Height: MinCardHeight}, want: CardStyle{Height: MinCardHeight}},
		{name: "the limits are included", give: CardStyle{Width: MaxCardWidth, Height: MaxCardHeight}, want: CardStyle{Width: MaxCardWidth, Height: MaxCardHeight}},
		{name: "too narrow", give: CardStyle{Width: MinCardWidth - 1}, wantErr: true},
		{name: "too wide", give: CardStyle{Width: MaxCardWidth + 1}, wantErr: true},
		{name: "too short", give: CardStyle{Height: MinCardHeight - 1}, wantErr: true},
		{name: "too tall", give: CardStyle{Height: MaxCardHeight + 1}, wantErr: true},
		{name: "a negative side", give: CardStyle{Width: -200}, wantErr: true},
		{name: "an unknown border", give: CardStyle{Border: "double"}, wantErr: true},
		{name: "a colour outside the palette", give: CardStyle{Color: "#ff0000"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.give.Normalized()
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidCardStyle) {
					t.Fatalf("err = %v, want ErrInvalidCardStyle", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCardStylePatch_Normalized(t *testing.T) {
	width, height := float32(300.4), float32(0)
	patch, err := CardStylePatch{Width: &width, Height: &height}.Normalized()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if *patch.Width != 300 || *patch.Height != 0 {
		t.Errorf("sides = %v x %v, want 300 x 0", *patch.Width, *patch.Height)
	}
	if patch.Color != nil || patch.Border != nil {
		t.Errorf("fields not sent appeared: %+v", patch)
	}

	invalid := []CardStylePatch{
		{Color: ptr(CardColor("teal"))},
		{Border: ptr(CardBorder("groove"))},
		{Width: ptr(float32(10))},
		{Height: ptr(float32(5000))},
	}
	for _, p := range invalid {
		if _, err := p.Normalized(); !errors.Is(err, ErrInvalidCardStyle) {
			t.Errorf("%+v: err = %v, want ErrInvalidCardStyle", p, err)
		}
	}
}

func TestCardStylePatch_Apply(t *testing.T) {
	style := CardStyle{Color: StrandColorRed, Border: CardBorderSolid, Width: 300, Height: 200}

	got := CardStylePatch{Border: ptr(CardBorder("")), Height: ptr(float32(0)), Color: ptr(StrandColorGreen)}.Apply(style)

	want := CardStyle{Color: StrandColorGreen, Width: 300}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if !(CardStylePatch{}).Empty() || (CardStylePatch{Width: ptr(float32(0))}).Empty() {
		t.Error("Empty: only a patch without fields changes nothing")
	}
}

// A card without a style keeps the shape it had before styles existed.
func TestBoardPostIt_DefaultStyleIsNotStored(t *testing.T) {
	plain := BoardPostIt{Title: BoardPostItTitle{Text: "x"}}

	raw, err := bson.Marshal(plain)
	if err != nil {
		t.Fatal(err)
	}
	var doc bson.M
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"color", "border", "width", "height"} {
		if _, ok := doc[field]; ok {
			t.Errorf("bson has %q for a card without a style", field)
		}
	}

	styled := BoardPostIt{CardStyle: CardStyle{Color: StrandColorPink, Width: 250}}
	js, err := json.Marshal(styled)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(js, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["color"] != "pink" || fields["width"] != float64(250) {
		t.Errorf("json = %s, want color and width beside the card's fields", js)
	}
	if _, ok := fields["height"]; ok {
		t.Errorf("json = %s, an unset side is left out", js)
	}
}

func ptr[T any](v T) *T { return &v }
