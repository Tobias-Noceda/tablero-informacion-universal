package models

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// Run longer with: go test -run '^$' -fuzz FuzzReplaceBodyStrings -fuzztime 30s ./common/models/

// A card's JSON body takes credentials string by string: whatever the body
// and the value, the result is one JSON document with as many strings as
// before, so a value holding quotes never splits or ends a string.
func FuzzReplaceBodyStrings(f *testing.F) {
	f.Add(`{"token":"$KEY","n":1}`, "$KEY", `abc"def\`)
	f.Add(`["$KEY",{"a":["$KEY"]},null,true,1e400]`, "$KEY", "nul:"+string(rune(0))+" line:"+string(rune(0x2028)))
	f.Add(`"$KEY"`, "$KEY", "")
	f.Add(`{"a":`, "$KEY", "x")
	f.Fuzz(func(t *testing.T, body, name, value string) {
		before, err := BodyStrings(body)
		replaced, replaceErr := ReplaceBodyStrings(body, map[string]string{name: value})
		if (err == nil) != (replaceErr == nil) {
			t.Fatalf("BodyStrings err = %v, ReplaceBodyStrings err = %v", err, replaceErr)
		}
		if err != nil {
			return
		}
		if !json.Valid([]byte(replaced)) {
			t.Fatalf("the result is not JSON: %q", replaced)
		}
		after, err := BodyStrings(replaced)
		if err != nil {
			t.Fatalf("the result does not decode again: %v", err)
		}
		if len(after) != len(before) {
			t.Fatalf("%d strings became %d", len(before), len(after))
		}
	})
}

// A strand's label is one line of at most MaxStrandLabel characters of text.
func FuzzStrandPatch(f *testing.F) {
	f.Add("depends on", "end", "thick", "red")
	f.Add(" two\nlines ", "", "", "")
	f.Add(strings.Repeat("ñ", MaxStrandLabel+1), "both", "thin", "blue")
	f.Add("a"+string(rune(0x2028))+"b", "", "", "")
	f.Add("\xff\xfe", "", "", "")
	f.Fuzz(func(t *testing.T, label, arrow, width, color string) {
		patch := StrandPatch{Label: &label, Arrow: (*StrandArrow)(&arrow), Width: (*StrandWidth)(&width), Color: (*StrandColor)(&color)}
		normalized, err := patch.Normalized()
		if err != nil {
			return
		}
		got := *normalized.Label
		if !utf8.ValidString(got) {
			t.Fatalf("accepted a label that is not UTF-8: %q", got)
		}
		if utf8.RuneCountInString(got) > MaxStrandLabel {
			t.Fatalf("accepted a label of %d characters", utf8.RuneCountInString(got))
		}
		if strings.ContainsFunc(got, func(r rune) bool { return unicode.IsControl(r) || r == 0x2028 || r == 0x2029 }) {
			t.Fatalf("accepted a label that breaks the line: %q", got)
		}
		again, err := normalized.Normalized()
		if err != nil || *again.Label != got {
			t.Fatalf("normalizing twice gave %q, %v", *again.Label, err)
		}
	})
}

// A card's size is a whole number of pixels within the limits, or the
// default (zero); anything else is refused.
func FuzzCardStylePatch(f *testing.F) {
	f.Add(float32(320.4), float32(180), "white", "dotted")
	f.Add(float32(0), float32(-1), "", "")
	f.Add(float32(math.Inf(1)), float32(math.NaN()), "red", "solid")
	f.Add(float32(119.5), float32(1200.49), "", "")
	f.Fuzz(func(t *testing.T, width, height float32, color, border string) {
		patch := CardStylePatch{Width: &width, Height: &height, Color: (*CardColor)(&color), Border: (*CardBorder)(&border)}
		normalized, err := patch.Normalized()
		if err != nil {
			return
		}
		for name, side := range map[string]struct{ value, min, max float32 }{
			"width":  {*normalized.Width, MinCardWidth, MaxCardWidth},
			"height": {*normalized.Height, MinCardHeight, MaxCardHeight},
		} {
			v := side.value
			if v == 0 {
				continue
			}
			if v != float32(math.Round(float64(v))) || v < side.min || v > side.max {
				t.Fatalf("accepted %s %v", name, v)
			}
		}
		again, err := normalized.Normalized()
		if err != nil || *again.Width != *normalized.Width || *again.Height != *normalized.Height {
			t.Fatalf("normalizing twice changed the size: %+v, %v", again, err)
		}
	})
}
