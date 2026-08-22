package boards

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/boards"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func setupRouter(db *mocks.MockDB) *gin.Engine {
	if db == nil {
		db = &mocks.MockDB{}
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewController(srv.New(db)).RegisterRoutes(r)
	return r
}

func do(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateBoard_OK(t *testing.T) {
	id := uuid.New()
	db := &mocks.MockDB{
		CreateBoardFn: func(name, owner string) (*models.Board, error) {
			return &models.Board{Id: id, Name: name, Owner: owner}, nil
		},
	}
	r := setupRouter(db)

	w := do(r, http.MethodPost, "/boards", `{"name":"B","owner":"o1"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	var got struct{ Id uuid.UUID }
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Id != id {
		t.Errorf("id = %v, want %v", got.Id, id)
	}
}

func TestCreateBoard_MissingFields(t *testing.T) {
	r := setupRouter(nil)
	w := do(r, http.MethodPost, "/boards", `{"name":"B"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetUserBoards_MissingCognitoID(t *testing.T) {
	r := setupRouter(nil)
	w := do(r, http.MethodGet, "/boards", "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (missing cognito_id)", w.Code)
	}
}

func TestGetUserBoards_OK(t *testing.T) {
	db := &mocks.MockDB{
		FindUserBoardsFn: func(cognitoID string) ([]models.Board, error) {
			if cognitoID != "user-1" {
				t.Errorf("cognitoID = %q, want user-1", cognitoID)
			}
			return []models.Board{{Id: uuid.New()}}, nil
		},
	}
	r := setupRouter(db)

	w := do(r, http.MethodGet, "/boards?cognito_id=user-1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestConnectPostIts_OK(t *testing.T) {
	board, src, tgt := uuid.New(), uuid.New(), uuid.New()
	var gb, gs, gt uuid.UUID
	db := &mocks.MockDB{
		ConnectPostItsFn: func(b, s, t uuid.UUID) (*models.Strand, error) {
			gb, gs, gt = b, s, t
			return &models.Strand{Id: uuid.New(), Source: s, Target: t}, nil
		},
	}
	r := setupRouter(db)

	body := `{"source":"` + src.String() + `","target":"` + tgt.String() + `"}`
	w := do(r, http.MethodPost, "/boards/"+board.String()+"/strands", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	if gb != board || gs != src || gt != tgt {
		t.Errorf("connected (%v,%v,%v), want (%v,%v,%v)", gb, gs, gt, board, src, tgt)
	}
}

func TestConnectPostIts_InvalidBoardUUID(t *testing.T) {
	r := setupRouter(nil)
	w := do(r, http.MethodPost, "/boards/not-a-uuid/strands", `{"source":"`+uuid.New().String()+`","target":"`+uuid.New().String()+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestDisconnectPostIts_OK(t *testing.T) {
	board, strand := uuid.New(), uuid.New()
	called := false
	db := &mocks.MockDB{
		DisconnectPostItsFn: func(b, s uuid.UUID) error {
			called = true
			return nil
		},
	}
	r := setupRouter(db)

	w := do(r, http.MethodDelete, "/boards/"+board.String()+"/strands/"+strand.String(), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
	if !called {
		t.Error("DisconnectPostIts was not called")
	}
}

func TestUpdateBoardName_OK(t *testing.T) {
	var gotName string
	db := &mocks.MockDB{
		UpdateBoardNameFn: func(_ uuid.UUID, name string) error {
			gotName = name
			return nil
		},
	}
	r := setupRouter(db)

	w := do(r, http.MethodPatch, "/boards/"+uuid.New().String()+"/name", `{"name":"Renamed"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if gotName != "Renamed" {
		t.Errorf("name = %q, want Renamed", gotName)
	}
}

func TestAddCollaborator_OK(t *testing.T) {
	var gotCog string
	db := &mocks.MockDB{
		AddCollaboratorToBoardFn: func(_ uuid.UUID, cognitoID string) error {
			gotCog = cognitoID
			return nil
		},
	}
	r := setupRouter(db)

	w := do(r, http.MethodPost, "/boards/"+uuid.New().String()+"/collaborators", `{"cognito_id":"collab-1"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if gotCog != "collab-1" {
		t.Errorf("cognito_id = %q, want collab-1", gotCog)
	}
}

func TestDeleteBoard_OK(t *testing.T) {
	called := false
	db := &mocks.MockDB{
		DeleteBoardFn: func(_ uuid.UUID) error {
			called = true
			return nil
		},
	}
	r := setupRouter(db)

	w := do(r, http.MethodDelete, "/boards/"+uuid.New().String(), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if !called {
		t.Error("DeleteBoard was not called")
	}
}

func TestGetBoard_OK(t *testing.T) {
	id := uuid.New()
	db := &mocks.MockDB{
		FindBoardFn: func(_ uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Name: "B"}, nil
		},
	}
	r := setupRouter(db)

	w := do(r, http.MethodGet, "/boards/"+id.String(), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got struct{ Id uuid.UUID }
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Id != id {
		t.Errorf("id = %v, want %v", got.Id, id)
	}
}

func TestGetBoard_InvalidUUID(t *testing.T) {
	r := setupRouter(nil)
	w := do(r, http.MethodGet, "/boards/not-a-uuid", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetBoard_ServiceError(t *testing.T) {
	db := &mocks.MockDB{
		FindBoardFn: func(_ uuid.UUID) (*models.Board, error) {
			return nil, errors.New("mongo: no documents in result")
		},
	}
	r := setupRouter(db)

	w := do(r, http.MethodGet, "/boards/"+uuid.New().String(), "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestGetBoardPostIts_OK(t *testing.T) {
	db := &mocks.MockDB{
		FindBoardPostItsFn: func(_ uuid.UUID) ([]models.PostIts, error) {
			return []models.PostIts{{Id: uuid.New()}, {Id: uuid.New()}}, nil
		},
	}
	r := setupRouter(db)

	w := do(r, http.MethodGet, "/boards/"+uuid.New().String()+"/post-its", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got []map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 2 {
		t.Errorf("got %d post-its, want 2", len(got))
	}
}

func TestGetBoardPostIts_InvalidUUID(t *testing.T) {
	r := setupRouter(nil)
	w := do(r, http.MethodGet, "/boards/nope/post-its", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestRemoveCollaborator_OK(t *testing.T) {
	var gotCog string
	db := &mocks.MockDB{
		RemoveCollaboratorFromBoardFn: func(_ uuid.UUID, cognitoID string) error {
			gotCog = cognitoID
			return nil
		},
	}
	r := setupRouter(db)

	w := do(r, http.MethodDelete, "/boards/"+uuid.New().String()+"/collaborators", `{"cognito_id":"collab-2"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if gotCog != "collab-2" {
		t.Errorf("cognito_id = %q, want collab-2", gotCog)
	}
}

func TestRemoveCollaborator_MissingField(t *testing.T) {
	r := setupRouter(nil)
	w := do(r, http.MethodDelete, "/boards/"+uuid.New().String()+"/collaborators", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (cognito_id required)", w.Code)
	}
}

func TestConnectPostIts_ServiceError(t *testing.T) {
	db := &mocks.MockDB{
		ConnectPostItsFn: func(_, _, _ uuid.UUID) (*models.Strand, error) {
			return nil, errors.New("mongo: no documents in result")
		},
	}
	r := setupRouter(db)

	body := `{"source":"` + uuid.New().String() + `","target":"` + uuid.New().String() + `"}`
	w := do(r, http.MethodPost, "/boards/"+uuid.New().String()+"/strands", body)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

// errDB returns a MockDB whose every relevant method fails, to drive the
// 500 branches of the handlers below.
func errDB() *mocks.MockDB {
	boom := errors.New("mongo: no documents in result")
	return &mocks.MockDB{
		FindUserBoardsFn:              func(string) ([]models.Board, error) { return nil, boom },
		CreateBoardFn:                 func(string, string) (*models.Board, error) { return nil, boom },
		DeleteBoardFn:                 func(uuid.UUID) error { return boom },
		AddCollaboratorToBoardFn:      func(uuid.UUID, string) error { return boom },
		RemoveCollaboratorFromBoardFn: func(uuid.UUID, string) error { return boom },
		UpdateBoardNameFn:             func(uuid.UUID, string) error { return boom },
		DisconnectPostItsFn:           func(_, _ uuid.UUID) error { return boom },
	}
}

func TestBoardHandlers_ServiceErrors(t *testing.T) {
	id := uuid.New().String()
	cases := []struct {
		name, method, path, body string
	}{
		{"GetUserBoards", http.MethodGet, "/boards?cognito_id=u", ""},
		{"CreateBoard", http.MethodPost, "/boards", `{"name":"B","owner":"o"}`},
		{"DeleteBoard", http.MethodDelete, "/boards/" + id, ""},
		{"AddCollaborator", http.MethodPost, "/boards/" + id + "/collaborators", `{"cognito_id":"c"}`},
		{"RemoveCollaborator", http.MethodDelete, "/boards/" + id + "/collaborators", `{"cognito_id":"c"}`},
		{"UpdateBoardName", http.MethodPatch, "/boards/" + id + "/name", `{"name":"N"}`},
		{"DisconnectPostIts", http.MethodDelete, "/boards/" + id + "/strands/" + id, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := setupRouter(errDB())
			w := do(r, tc.method, tc.path, tc.body)
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500 (body: %s)", w.Code, w.Body.String())
			}
		})
	}
}

func TestBoardHandlers_InvalidUUID(t *testing.T) {
	id := uuid.New().String()
	cases := []struct {
		name, method, path, body string
	}{
		{"DeleteBoard", http.MethodDelete, "/boards/nope", ""},
		{"AddCollaborator", http.MethodPost, "/boards/nope/collaborators", `{"cognito_id":"c"}`},
		{"RemoveCollaborator", http.MethodDelete, "/boards/nope/collaborators", `{"cognito_id":"c"}`},
		{"UpdateBoardName", http.MethodPatch, "/boards/nope/name", `{"name":"N"}`},
		{"DisconnectPostIts", http.MethodDelete, "/boards/nope/strands/" + id, ""},
		{"DisconnectPostIts", http.MethodDelete, "/boards/" + id + "/strands/jajant", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := setupRouter(nil)
			w := do(r, tc.method, tc.path, tc.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
		})
	}
}

func TestBoardHandlers_BadBody(t *testing.T) {
	id := uuid.New().String()
	cases := []struct {
		name, method, path string
	}{
		{"AddCollaborator", http.MethodPost, "/boards/" + id + "/collaborators"},
		{"UpdateBoardName", http.MethodPatch, "/boards/" + id + "/name"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := setupRouter(nil)
			w := do(r, tc.method, tc.path, `{bad json`)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
		})
	}
}
