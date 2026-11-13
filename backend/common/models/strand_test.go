package models

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestStrandStyle_Normalized(t *testing.T) {
	tests := []struct {
		name    string
		give    StrandStyle
		want    StrandStyle
		wantErr bool
	}{
		{name: "the zero style is the default line", give: StrandStyle{}, want: StrandStyle{}},
		{
			name: "every field set",
			give: StrandStyle{Label: "depends on", Arrow: StrandArrowEnd, Width: StrandWidthThick, Color: StrandColorBlue},
			want: StrandStyle{Label: "depends on", Arrow: StrandArrowEnd, Width: StrandWidthThick, Color: StrandColorBlue},
		},
		{name: "the label is trimmed", give: StrandStyle{Label: "  feeds  "}, want: StrandStyle{Label: "feeds"}},
		{name: "a label of only spaces is no label", give: StrandStyle{Label: "   "}, want: StrandStyle{}},
		{
			name: "the limit counts characters, not bytes",
			give: StrandStyle{Label: strings.Repeat("ñ", MaxStrandLabel)},
			want: StrandStyle{Label: strings.Repeat("ñ", MaxStrandLabel)},
		},
		{name: "a label over the limit", give: StrandStyle{Label: strings.Repeat("a", MaxStrandLabel+1)}, wantErr: true},
		{name: "a label over two lines", give: StrandStyle{Label: "one\ntwo"}, wantErr: true},
		{name: "an unknown arrow", give: StrandStyle{Arrow: "sideways"}, wantErr: true},
		{name: "an unknown width", give: StrandStyle{Width: "huge"}, wantErr: true},
		{name: "a colour outside the palette", give: StrandStyle{Color: "#ff0000"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.give.Normalized()
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidStrand) {
					t.Fatalf("err = %v, want ErrInvalidStrand", err)
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

func TestStrandPatch_Normalized(t *testing.T) {
	label, arrow, color := "  feeds ", StrandArrowBoth, StrandColor("teal")

	got, err := StrandPatch{Label: &label, Arrow: &arrow}.Normalized()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if *got.Label != "feeds" || *got.Arrow != StrandArrowBoth || got.Width != nil || got.Color != nil {
		t.Errorf("got %+v", got)
	}
	if label != "  feeds " {
		t.Errorf("the caller's label changed to %q", label)
	}

	if _, err := (StrandPatch{Color: &color}).Normalized(); !errors.Is(err, ErrInvalidStrand) {
		t.Errorf("err = %v, want ErrInvalidStrand", err)
	}
}

func TestStrandPatch_Apply(t *testing.T) {
	empty, width := "", StrandWidthThin
	style := StrandStyle{Label: "feeds", Arrow: StrandArrowEnd, Width: StrandWidthThick, Color: StrandColorRed}

	got := StrandPatch{Label: &empty, Width: &width}.Apply(style)

	want := StrandStyle{Arrow: StrandArrowEnd, Width: StrandWidthThin, Color: StrandColorRed}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if !(StrandPatch{}).Empty() || (StrandPatch{Width: &width}).Empty() {
		t.Error("Empty is wrong")
	}
}

// Strands saved before styles existed decode to the default style, and a
// default style adds nothing to what is stored or sent.
func TestStrand_Encoding(t *testing.T) {
	type unstyled struct {
		Id     uuid.UUID `bson:"id"`
		Source uuid.UUID `bson:"source"`
		Target uuid.UUID `bson:"target"`
	}
	want := unstyled{Id: uuid.New(), Source: uuid.New(), Target: uuid.New()}
	raw, err := bson.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var old Strand
	if err := bson.Unmarshal(raw, &old); err != nil {
		t.Fatal(err)
	}
	if old.Id != want.Id || old.Target != want.Target || old.StrandStyle != (StrandStyle{}) {
		t.Errorf("old strand = %+v", old)
	}

	styled := Strand{StrandStyle: StrandStyle{Label: "feeds", Color: StrandColorBlue}}
	doc, err := bson.Marshal(styled)
	if err != nil {
		t.Fatal(err)
	}
	var stored bson.M
	_ = bson.Unmarshal(doc, &stored)
	if stored["label"] != "feeds" || stored["color"] != "blue" || stored["arrow"] != nil {
		t.Errorf("stored = %v", stored)
	}

	body, _ := json.Marshal(Strand{})
	var sent map[string]any
	_ = json.Unmarshal(body, &sent)
	for _, field := range []string{"label", "arrow", "width", "color"} {
		if _, ok := sent[field]; ok {
			t.Errorf("a default strand sends %q", field)
		}
	}
}
