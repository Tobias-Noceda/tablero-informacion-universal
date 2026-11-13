package models

import (
	"errors"
	"fmt"
	"math"
)

// The sizes a card may be resized to, in canvas pixels.
const (
	MinCardWidth  = 120
	MaxCardWidth  = 1600
	MinCardHeight = 60
	MaxCardHeight = 1200
)

// ErrInvalidCardStyle reports a card style outside what a board can draw.
var ErrInvalidCardStyle = errors.New("invalid card style")

// CardColor is a colour of the board's palette, the same one strands are
// drawn with. Empty is the card's usual colour.
type CardColor = StrandColor

// CardBorder is the line drawn around a card. Empty is CardBorderNone.
type CardBorder string

const (
	CardBorderNone   CardBorder = "none"
	CardBorderSolid  CardBorder = "solid"
	CardBorderDashed CardBorder = "dashed"
	CardBorderDotted CardBorder = "dotted"
)

// Valid reports whether b is empty or one of the borders.
func (b CardBorder) Valid() bool {
	switch b {
	case "", CardBorderNone, CardBorderSolid, CardBorderDashed, CardBorderDotted:
		return true
	}
	return false
}

// CardStyle is how a card is drawn on its board. Every field is optional:
// the zero value is a card of the usual colour, without a border, as large
// as its content, which is how every card looked before styles existed.
// Width and Height are set independently; zero leaves that side to the
// content.
type CardStyle struct {
	Color  CardColor  `bson:"color,omitempty" json:"color,omitempty"`
	Border CardBorder `bson:"border,omitempty" json:"border,omitempty"`
	Width  float32    `bson:"width,omitempty" json:"width,omitempty"`
	Height float32    `bson:"height,omitempty" json:"height,omitempty"`
}

// Normalized returns s with its size rounded to whole pixels, or
// ErrInvalidCardStyle.
func (s CardStyle) Normalized() (CardStyle, error) {
	if err := validateCardEnums(s.Color, s.Border); err != nil {
		return CardStyle{}, err
	}
	width, err := normalizeCardSide("width", s.Width, MinCardWidth, MaxCardWidth)
	if err != nil {
		return CardStyle{}, err
	}
	height, err := normalizeCardSide("height", s.Height, MinCardHeight, MaxCardHeight)
	if err != nil {
		return CardStyle{}, err
	}
	s.Width, s.Height = width, height
	return s, nil
}

// CardStylePatch changes part of a card's style. A nil field stays as it
// is; an empty one (zero for a size) goes back to its default.
type CardStylePatch struct {
	Color  *CardColor  `json:"color,omitempty"`
	Border *CardBorder `json:"border,omitempty"`
	Width  *float32    `json:"width,omitempty"`
	Height *float32    `json:"height,omitempty"`
}

// Empty reports whether p changes nothing.
func (p CardStylePatch) Empty() bool {
	return p.Color == nil && p.Border == nil && p.Width == nil && p.Height == nil
}

// Normalized returns p with its size rounded to whole pixels, or
// ErrInvalidCardStyle.
func (p CardStylePatch) Normalized() (CardStylePatch, error) {
	if err := validateCardEnums(deref(p.Color), deref(p.Border)); err != nil {
		return CardStylePatch{}, err
	}
	if p.Width != nil {
		width, err := normalizeCardSide("width", *p.Width, MinCardWidth, MaxCardWidth)
		if err != nil {
			return CardStylePatch{}, err
		}
		p.Width = &width
	}
	if p.Height != nil {
		height, err := normalizeCardSide("height", *p.Height, MinCardHeight, MaxCardHeight)
		if err != nil {
			return CardStylePatch{}, err
		}
		p.Height = &height
	}
	return p, nil
}

// Apply returns s with p's changes.
func (p CardStylePatch) Apply(s CardStyle) CardStyle {
	if p.Color != nil {
		s.Color = *p.Color
	}
	if p.Border != nil {
		s.Border = *p.Border
	}
	if p.Width != nil {
		s.Width = *p.Width
	}
	if p.Height != nil {
		s.Height = *p.Height
	}
	return s
}

func validateCardEnums(color CardColor, border CardBorder) error {
	switch {
	case !color.Valid():
		return fmt.Errorf("%w: unknown color %q", ErrInvalidCardStyle, color)
	case !border.Valid():
		return fmt.Errorf("%w: unknown border %q", ErrInvalidCardStyle, border)
	}
	return nil
}

// normalizeCardSide rounds a side to whole pixels; zero means "as large as
// the content" and is always accepted.
func normalizeCardSide(name string, value float32, min, max float32) (float32, error) {
	if value == 0 {
		return 0, nil
	}
	rounded := float32(math.Round(float64(value)))
	if math.IsNaN(float64(value)) || rounded < min || rounded > max {
		return 0, fmt.Errorf("%w: %s must be between %g and %g", ErrInvalidCardStyle, name, min, max)
	}
	return rounded, nil
}
