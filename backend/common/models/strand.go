package models

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// MaxStrandLabel is how many characters a strand's label may hold.
const MaxStrandLabel = 80

// ErrInvalidStrand reports a strand style outside what a board can draw.
var ErrInvalidStrand = errors.New("invalid strand")

// StrandArrow says which ends of a strand point at their card. Empty draws
// no arrow, like StrandArrowNone.
type StrandArrow string

const (
	StrandArrowNone  StrandArrow = "none"
	StrandArrowStart StrandArrow = "start"
	StrandArrowEnd   StrandArrow = "end"
	StrandArrowBoth  StrandArrow = "both"
)

// StrandWidth is how thick a strand is drawn. Empty is StrandWidthNormal.
type StrandWidth string

const (
	StrandWidthThin   StrandWidth = "thin"
	StrandWidthNormal StrandWidth = "normal"
	StrandWidthThick  StrandWidth = "thick"
)

// StrandColor names a colour of the board's palette, so each one can be
// drawn legibly in both the light and the dark theme. Empty is
// StrandColorGray.
type StrandColor string

const (
	StrandColorGray   StrandColor = "gray"
	StrandColorRed    StrandColor = "red"
	StrandColorOrange StrandColor = "orange"
	StrandColorYellow StrandColor = "yellow"
	StrandColorGreen  StrandColor = "green"
	StrandColorBlue   StrandColor = "blue"
	StrandColorPurple StrandColor = "purple"
	StrandColorPink   StrandColor = "pink"
)

// Valid reports whether a is empty or one of the arrows.
func (a StrandArrow) Valid() bool {
	switch a {
	case "", StrandArrowNone, StrandArrowStart, StrandArrowEnd, StrandArrowBoth:
		return true
	}
	return false
}

// Valid reports whether w is empty or one of the widths.
func (w StrandWidth) Valid() bool {
	switch w {
	case "", StrandWidthThin, StrandWidthNormal, StrandWidthThick:
		return true
	}
	return false
}

// Valid reports whether c is empty or a colour of the palette.
func (c StrandColor) Valid() bool {
	switch c {
	case "", StrandColorGray, StrandColorRed, StrandColorOrange, StrandColorYellow,
		StrandColorGreen, StrandColorBlue, StrandColorPurple, StrandColorPink:
		return true
	}
	return false
}

// StrandStyle is how a strand is drawn. Every field is optional: the zero
// value is an unlabelled, normal, gray line without arrows, which is how
// every strand looked before styles existed.
type StrandStyle struct {
	Label string      `bson:"label,omitempty" json:"label,omitempty"`
	Arrow StrandArrow `bson:"arrow,omitempty" json:"arrow,omitempty"`
	Width StrandWidth `bson:"width,omitempty" json:"width,omitempty"`
	Color StrandColor `bson:"color,omitempty" json:"color,omitempty"`
}

// Normalized returns s with its label trimmed, or ErrInvalidStrand.
func (s StrandStyle) Normalized() (StrandStyle, error) {
	label, err := normalizeStrandLabel(s.Label)
	if err != nil {
		return StrandStyle{}, err
	}
	if err := validateStrandEnums(s.Arrow, s.Width, s.Color); err != nil {
		return StrandStyle{}, err
	}
	s.Label = label
	return s, nil
}

// Strand links two cards of a board.
type Strand struct {
	Id          uuid.UUID `bson:"id" json:"id"`
	Source      uuid.UUID `bson:"source" json:"source"`
	Target      uuid.UUID `bson:"target" json:"target"`
	StrandStyle `bson:",inline"`
}

// StrandPatch changes part of a strand's style. A nil field stays as it is;
// an empty one goes back to its default.
type StrandPatch struct {
	Label *string      `json:"label,omitempty"`
	Arrow *StrandArrow `json:"arrow,omitempty"`
	Width *StrandWidth `json:"width,omitempty"`
	Color *StrandColor `json:"color,omitempty"`
}

// Empty reports whether p changes nothing.
func (p StrandPatch) Empty() bool {
	return p.Label == nil && p.Arrow == nil && p.Width == nil && p.Color == nil
}

// Normalized returns p with its label trimmed, or ErrInvalidStrand.
func (p StrandPatch) Normalized() (StrandPatch, error) {
	if p.Label != nil {
		label, err := normalizeStrandLabel(*p.Label)
		if err != nil {
			return StrandPatch{}, err
		}
		p.Label = &label
	}
	if err := validateStrandEnums(deref(p.Arrow), deref(p.Width), deref(p.Color)); err != nil {
		return StrandPatch{}, err
	}
	return p, nil
}

// Apply returns s with p's changes.
func (p StrandPatch) Apply(s StrandStyle) StrandStyle {
	if p.Label != nil {
		s.Label = *p.Label
	}
	if p.Arrow != nil {
		s.Arrow = *p.Arrow
	}
	if p.Width != nil {
		s.Width = *p.Width
	}
	if p.Color != nil {
		s.Color = *p.Color
	}
	return s
}

func normalizeStrandLabel(label string) (string, error) {
	label = strings.TrimSpace(label)
	if utf8.RuneCountInString(label) > MaxStrandLabel {
		return "", fmt.Errorf("%w: label longer than %d characters", ErrInvalidStrand, MaxStrandLabel)
	}
	if strings.ContainsFunc(label, unicode.IsControl) {
		return "", fmt.Errorf("%w: label is a single line of text", ErrInvalidStrand)
	}
	return label, nil
}

func validateStrandEnums(arrow StrandArrow, width StrandWidth, color StrandColor) error {
	switch {
	case !arrow.Valid():
		return fmt.Errorf("%w: unknown arrow %q", ErrInvalidStrand, arrow)
	case !width.Valid():
		return fmt.Errorf("%w: unknown width %q", ErrInvalidStrand, width)
	case !color.Valid():
		return fmt.Errorf("%w: unknown color %q", ErrInvalidStrand, color)
	}
	return nil
}

func deref[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}
	return *v
}
