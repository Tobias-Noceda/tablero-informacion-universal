package customwellknowns

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	srv "github.com/Secreto31126/tesis/common/services/customwellknowns"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// tokenVerifier lets a test call as a personal access token: "pat:<id>".
type tokenVerifier struct{ mocks.SubjectVerifier }

func (v tokenVerifier) Verify(token string) (infrastructure.Claims, error) {
	if id, ok := strings.CutPrefix(token, "pat:"); ok {
		return infrastructure.Claims{Subject: id, AccessToken: true}, nil
	}
	return v.SubjectVerifier.Verify(token)
}

const catFact = `{
	"name": "Cat fact",
	"title": {"text": "{{fact}}", "vars": true},
	"resource": "https://catfact.ninja/fact",
	"response": "json",
	"query": {"fact": ".fact"},
	"rate": 60,
	"outputs": [{"key": "fact", "label": "Dato", "example": "Cats sleep"}]
}`

// setup: a board owned by "ana" with "bob" as a viewer.
func setup(t *testing.T) (http.Handler, uuid.UUID) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	board := &models.Board{Id: uuid.New(), Owner: "ana", Members: []models.BoardMember{{User: "bob", Role: models.BoardViewer}}}
	db := &mocks.MockDB{FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
		if id != board.Id {
			return nil, errors.New("no board")
		}
		b := *board
		return &b, nil
	}}

	r := gin.New()
	api := r.Group("/api/v1", middleware.RequireAuth(tokenVerifier{}), middleware.RestrictAccessTokens())
	NewController(srv.New(mocks.NewMemoryCustomWellKnownStore(), db, access.New(nil))).RegisterRoutes(api)
	return r, board.Id
}

func do(r http.Handler, token, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLifecycle(t *testing.T) {
	r, board := setup(t)

	w := do(r, "ana", http.MethodPost, "/users/ana/well-knowns", catFact)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var created models.CustomWellKnown
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if !models.IsCustomWellKnownKey(created.Key) || created.Scope.Kind != models.CustomScopeUser || created.Outputs[0].Label != "Dato" {
		t.Errorf("created: %+v", created)
	}

	if w := do(r, "ana", http.MethodGet, "/users/ana/well-knowns", ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), created.Key) {
		t.Errorf("list: %d %s", w.Code, w.Body)
	}
	if w := do(r, "bob", http.MethodGet, "/users/ana/well-knowns", ""); w.Code != http.StatusNotFound {
		t.Errorf("someone else's list: %d", w.Code)
	}
	if w := do(r, "bob", http.MethodGet, "/custom-well-knowns/"+created.Key, ""); w.Code != http.StatusNotFound {
		t.Errorf("someone else's get: %d", w.Code)
	}

	edited := strings.Replace(catFact, "Cat fact", "Gato", 1)
	if w := do(r, "ana", http.MethodPut, "/custom-well-knowns/"+created.Key, edited); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Gato") {
		t.Errorf("update: %d %s", w.Code, w.Body)
	}

	w = do(r, "ana", http.MethodGet, "/boards/"+board.String()+"/well-knowns/usable", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"usable":true`) {
		t.Errorf("usable: %d %s", w.Code, w.Body)
	}

	if w := do(r, "ana", http.MethodDelete, "/custom-well-knowns/"+created.Key, ""); w.Code != http.StatusNoContent {
		t.Errorf("delete: %d %s", w.Code, w.Body)
	}
	if w := do(r, "ana", http.MethodGet, "/custom-well-knowns/"+created.Key, ""); w.Code != http.StatusNotFound {
		t.Errorf("after delete: %d", w.Code)
	}
}

func TestBoardScopeAndErrors(t *testing.T) {
	r, board := setup(t)
	path := "/boards/" + board.String() + "/well-knowns"

	if w := do(r, "bob", http.MethodPost, path, catFact); w.Code != http.StatusNotFound {
		t.Errorf("viewer create: %d", w.Code)
	}
	if w := do(r, "ana", http.MethodPost, path, catFact); w.Code != http.StatusCreated {
		t.Errorf("owner create: %d %s", w.Code, w.Body)
	}
	if w := do(r, "bob", http.MethodGet, path, ""); w.Code != http.StatusOK {
		t.Errorf("viewer list: %d", w.Code)
	}

	w := do(r, "ana", http.MethodPost, path, strings.Replace(catFact, `"https://catfact.ninja/fact"`, `""`, 1))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "resource is required") {
		t.Errorf("invalid: %d %s", w.Code, w.Body)
	}
	if w := do(r, "ana", http.MethodPost, path, "{"); w.Code != http.StatusBadRequest {
		t.Errorf("malformed: %d", w.Code)
	}
	if w := do(r, "ana", http.MethodGet, "/boards/nope/well-knowns", ""); w.Code != http.StatusBadRequest {
		t.Errorf("bad uuid: %d", w.Code)
	}
}

func TestAccessTokens(t *testing.T) {
	r, board := setup(t)

	if w := do(r, "pat:ana", http.MethodGet, "/boards/"+board.String()+"/well-knowns/usable", ""); w.Code != http.StatusOK {
		t.Errorf("token reads the board's types: %d %s", w.Code, w.Body)
	}
	for _, path := range []string{"/users/ana/well-knowns", "/boards/" + board.String() + "/well-knowns"} {
		if w := do(r, "pat:ana", http.MethodPost, path, catFact); w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "token_not_allowed") {
			t.Errorf("token create %s: %d %s", path, w.Code, w.Body)
		}
	}
}
