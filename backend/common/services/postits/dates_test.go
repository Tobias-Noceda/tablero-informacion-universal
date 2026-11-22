package postits

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func temperatureParams(start, end string) map[string]string {
	return map[string]string{"$latitude": "-34.6", "$longitude": "-58.4", "$start_date": start, "$end_date": end}
}

func TestCreatePostIt_RefusesADateThatIsNotOne(t *testing.T) {
	svc := newService(boardOf(boardOwner), nil, nil)

	card := &models.PostIts{Board: uuid.New(), WellKnown: "temperature", Params: temperatureParams("22/11/2026", "2026-11-23")}
	if _, err := svc.CreatePostIt(boardOwner, nil, card); !errors.Is(err, models.ErrInvalidParameter) {
		t.Errorf("got %v, want ErrInvalidParameter", err)
	}

	card = &models.PostIts{Board: uuid.New(), WellKnown: "temperature", Params: temperatureParams("2026-11-22", "2026-11-23")}
	if _, err := svc.CreatePostIt(boardOwner, nil, card); err != nil {
		t.Errorf("valid dates were refused: %v", err)
	}
}

func TestUpdatePostIt_RefusesADateThatIsNotOne(t *testing.T) {
	id, board := uuid.New(), uuid.New()
	db := boardOf(boardOwner)
	db.FindPostItFn = func(uuid.UUID) (*models.PostIts, error) {
		return &models.PostIts{Id: id, Board: board, WellKnown: "temperature", RunAs: boardOwner.ID}, nil
	}
	saved := false
	db.UpdatePostItFn = func(uuid.UUID, map[string]any) error {
		saved = true
		return nil
	}
	svc := newService(db, nil, nil)

	err := svc.UpdatePostIt(boardOwner, id, map[string]any{"params": temperatureParams("2026-11-22", "mañana")})
	if !errors.Is(err, models.ErrInvalidParameter) {
		t.Errorf("got %v, want ErrInvalidParameter", err)
	}
	if saved {
		t.Error("the invalid date was saved")
	}

	if err := svc.UpdatePostIt(boardOwner, id, map[string]any{"params": temperatureParams("2026-11-22", "2026-11-30")}); err != nil {
		t.Errorf("valid dates were refused: %v", err)
	}
}

func TestBuiltinDatesHaveTheirTypes(t *testing.T) {
	want := map[string]models.WellKnownParameterType{
		"temperature/$start_date":   models.ParameterDate,
		"temperature/$end_date":     models.ParameterDate,
		"google_calendar/$time_min": models.ParameterDateTime,
	}
	defs, _ := builtins().ListWellKnowns()
	for _, def := range defs {
		for _, p := range def.Parameters {
			if kind, ok := want[def.Key+"/"+p.Key]; ok && p.Type != kind {
				t.Errorf("%s %s: type %s, want %s", def.Key, p.Key, p.Type, kind)
			}
		}
	}
}
