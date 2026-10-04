package boards

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	b_srv "github.com/Secreto31126/tesis/common/services/boards"
	r_srv "github.com/Secreto31126/tesis/common/services/realtime"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Every request below is made as one of these: the board is owned by "owner"
// and shared with "ana"; "eve" is a stranger.
const (
	owner    = "owner"
	ana      = "ana"
	outsider = "eve"
)

var boardID = uuid.New()

// withBoard serves the shared board from FindBoard on top of db.
func withBoard(db *mocks.MockDB) *mocks.MockDB {
	if db == nil {
		db = &mocks.MockDB{}
	}
	if db.FindBoardFn == nil {
		db.FindBoardFn = func(id uuid.UUID) (*models.Board, error) {
			if id != boardID {
				return nil, errors.New("mongo: no documents in result")
			}
			return &models.Board{Id: id, Name: "B", Owner: owner, Collaborators: []string{ana}}, nil
		}
	}
	return db
}

func setupRouter(db *mocks.MockDB, cache *mocks.MockCache) *gin.Engine {
	if cache == nil {
		cache = &mocks.MockCache{
			ConnectClientToBoardFn:      func(*models.Board, uuid.UUID) ([]string, error) { return []string{}, nil },
			DisconnectClientFromBoardFn: func(*models.Board, uuid.UUID) error { return nil },
		}
	}

	gin.SetMode(gin.TestMode)

	r := gin.New()
	bs := b_srv.New(withBoard(db), &mocks.MockScopePurger{})
	rs := r_srv.New(bs, cache)

	NewController(bs, rs).RegisterRoutes(r.Group("", middleware.RequireAuth(mocks.SubjectVerifier{})))
	return r
}

func do(r http.Handler, caller, method, path, body string) *httptest.ResponseRecorder {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if caller != "" {
		req.Header.Set("Authorization", "Bearer "+caller)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

var board = "/boards/" + boardID.String()

func TestCreateBoard_TheCallerOwnsIt(t *testing.T) {
	id := uuid.New()
	var gotOwner string
	db := &mocks.MockDB{
		CreateBoardFn: func(name, owner string) (*models.Board, error) {
			gotOwner = owner
			return &models.Board{Id: id, Name: name, Owner: owner}, nil
		},
	}
	r := setupRouter(db, nil)

	w := do(r, ana, http.MethodPost, "/boards", `{"name":"B","owner":"someone-else"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	if gotOwner != ana {
		t.Errorf("owner = %q, want the caller", gotOwner)
	}
	var got struct{ Id uuid.UUID }
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Id != id {
		t.Errorf("id = %v, want %v", got.Id, id)
	}
}

func TestCreateBoard_MissingName(t *testing.T) {
	w := do(setupRouter(nil, nil), ana, http.MethodPost, "/boards", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetUserBoards_ListsTheCallersBoards(t *testing.T) {
	db := &mocks.MockDB{
		FindUserBoardsFn: func(user string) ([]models.Board, error) {
			if user != ana {
				t.Errorf("user = %q, want the caller", user)
			}
			return []models.Board{{Id: uuid.New()}}, nil
		},
	}

	w := do(setupRouter(db, nil), ana, http.MethodGet, "/boards", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestEveryRoute_RequiresAToken(t *testing.T) {
	r := setupRouter(nil, nil)
	for _, route := range r.Routes() {
		path := strings.NewReplacer(":id", boardID.String(), ":strand", uuid.NewString()).Replace(route.Path)
		if w := do(r, "", route.Method, path, ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401", route.Method, route.Path, w.Code)
		}
	}
}

func okDB() *mocks.MockDB {
	return &mocks.MockDB{
		FindBoardPostItsFn:            func(uuid.UUID) ([]models.PostIts, error) { return []models.PostIts{}, nil },
		ConnectPostItsFn:              func(_, s, t uuid.UUID) (*models.Strand, error) { return &models.Strand{Source: s, Target: t}, nil },
		DisconnectPostItsFn:           func(_, _ uuid.UUID) error { return nil },
		DeleteBoardFn:                 func(uuid.UUID) error { return nil },
		UpdateBoardNameFn:             func(uuid.UUID, string) error { return nil },
		AddCollaboratorToBoardFn:      func(uuid.UUID, string) error { return nil },
		RemoveCollaboratorFromBoardFn: func(uuid.UUID, string) error { return nil },
	}
}

type call struct {
	name, method, path, body string
	want                     int
}

var strand = `{"source":"` + uuid.NewString() + `","target":"` + uuid.NewString() + `"}`

var memberCalls = []call{
	{"GetBoard", http.MethodGet, board, "", http.StatusOK},
	{"GetBoardPostIts", http.MethodGet, board + "/post-its", "", http.StatusOK},
	{"ConnectPostIts", http.MethodPost, board + "/strands", strand, http.StatusCreated},
	{"DisconnectPostIts", http.MethodDelete, board + "/strands/" + uuid.NewString(), "", http.StatusNoContent},
	{"ConnectClient", http.MethodPut, board + "/online?peer=" + uuid.NewString(), "", http.StatusOK},
	{"DisconnectClient", http.MethodDelete, board + "/online?peer=" + uuid.NewString(), "", http.StatusNoContent},
}

var ownerCalls = []call{
	{"UpdateBoardName", http.MethodPatch, board + "/name", `{"name":"N"}`, http.StatusNoContent},
	{"AddCollaborator", http.MethodPost, board + "/collaborators", `{"user":"bob"}`, http.StatusNoContent},
	{"RemoveCollaborator", http.MethodDelete, board + "/collaborators", `{"user":"bob"}`, http.StatusNoContent},
	{"DeleteBoard", http.MethodDelete, board, "", http.StatusNoContent},
}

func TestMemberRoutes_AStrangerGetsNotFound(t *testing.T) {
	for _, tc := range memberCalls {
		t.Run(tc.name, func(t *testing.T) {
			r := setupRouter(okDB(), nil)
			for _, member := range []string{owner, ana} {
				if w := do(r, member, tc.method, tc.path, tc.body); w.Code != tc.want {
					t.Errorf("as %s: status = %d, want %d (body: %s)", member, w.Code, tc.want, w.Body.String())
				}
			}
			if w := do(r, outsider, tc.method, tc.path, tc.body); w.Code != http.StatusNotFound {
				t.Errorf("as a stranger: status = %d, want 404", w.Code)
			}
		})
	}
}

func TestOwnerRoutes_ACollaboratorGetsNotFound(t *testing.T) {
	for _, tc := range ownerCalls {
		t.Run(tc.name, func(t *testing.T) {
			r := setupRouter(okDB(), nil)
			for _, intruder := range []string{ana, outsider} {
				if w := do(r, intruder, tc.method, tc.path, tc.body); w.Code != http.StatusNotFound {
					t.Errorf("as %s: status = %d, want 404", intruder, w.Code)
				}
			}
			if w := do(r, owner, tc.method, tc.path, tc.body); w.Code != tc.want {
				t.Errorf("as the owner: status = %d, want %d (body: %s)", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

func TestMissingBoard_IsNotFound(t *testing.T) {
	w := do(setupRouter(nil, nil), owner, http.MethodGet, "/boards/"+uuid.NewString(), "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestAddCollaborator_NamesTheUser(t *testing.T) {
	var added string
	db := okDB()
	db.AddCollaboratorToBoardFn = func(_ uuid.UUID, user string) error {
		added = user
		return nil
	}

	w := do(setupRouter(db, nil), owner, http.MethodPost, board+"/collaborators", `{"user":"bob"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if added != "bob" {
		t.Errorf("added %q, want bob", added)
	}
}

func TestRemoveCollaborator_ACollaboratorMayLeave(t *testing.T) {
	w := do(setupRouter(okDB(), nil), ana, http.MethodDelete, board+"/collaborators", `{"user":"ana"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
}

func TestRemoveCollaborator_TheOwnerIsABadRequest(t *testing.T) {
	w := do(setupRouter(okDB(), nil), owner, http.MethodDelete, board+"/collaborators", `{"user":"owner"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestBoardHandlers_ServiceErrors(t *testing.T) {
	boom := errors.New("mongo: connection lost")
	db := &mocks.MockDB{
		FindUserBoardsFn:              func(string) ([]models.Board, error) { return nil, boom },
		CreateBoardFn:                 func(string, string) (*models.Board, error) { return nil, boom },
		DeleteBoardFn:                 func(uuid.UUID) error { return boom },
		AddCollaboratorToBoardFn:      func(uuid.UUID, string) error { return boom },
		RemoveCollaboratorFromBoardFn: func(uuid.UUID, string) error { return boom },
		UpdateBoardNameFn:             func(uuid.UUID, string) error { return boom },
		ConnectPostItsFn:              func(_, _, _ uuid.UUID) (*models.Strand, error) { return nil, boom },
		DisconnectPostItsFn:           func(_, _ uuid.UUID) error { return boom },
	}
	cases := []call{
		{"GetUserBoards", http.MethodGet, "/boards", "", 0},
		{"CreateBoard", http.MethodPost, "/boards", `{"name":"B"}`, 0},
		{"ConnectPostIts", http.MethodPost, board + "/strands", strand, 0},
		{"DisconnectPostIts", http.MethodDelete, board + "/strands/" + uuid.NewString(), "", 0},
	}
	cases = append(cases, ownerCalls...)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if w := do(setupRouter(db, nil), owner, tc.method, tc.path, tc.body); w.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500 (body: %s)", w.Code, w.Body.String())
			}
		})
	}
}

func TestBoardHandlers_InvalidUUID(t *testing.T) {
	id := uuid.NewString()
	cases := []call{
		{"GetBoard", http.MethodGet, "/boards/nope", "", 0},
		{"GetBoardPostIts", http.MethodGet, "/boards/nope/post-its", "", 0},
		{"DeleteBoard", http.MethodDelete, "/boards/nope", "", 0},
		{"AddCollaborator", http.MethodPost, "/boards/nope/collaborators", `{"user":"c"}`, 0},
		{"RemoveCollaborator", http.MethodDelete, "/boards/nope/collaborators", `{"user":"c"}`, 0},
		{"UpdateBoardName", http.MethodPatch, "/boards/nope/name", `{"name":"N"}`, 0},
		{"ConnectPostIts", http.MethodPost, "/boards/nope/strands", strand, 0},
		{"DisconnectPostIts board", http.MethodDelete, "/boards/nope/strands/" + id, "", 0},
		{"DisconnectPostIts strand", http.MethodDelete, board + "/strands/jajant", "", 0},
		{"ConnectClient peer", http.MethodPut, board + "/online?peer=nope", "", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if w := do(setupRouter(nil, nil), owner, tc.method, tc.path, tc.body); w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
		})
	}
}

func TestBoardHandlers_BadBody(t *testing.T) {
	for _, path := range []string{board + "/collaborators", board + "/name", board + "/strands"} {
		method := http.MethodPost
		if strings.HasSuffix(path, "/name") {
			method = http.MethodPatch
		}
		if w := do(setupRouter(nil, nil), owner, method, path, `{bad json`); w.Code != http.StatusBadRequest {
			t.Errorf("%s %s: status = %d, want 400", method, path, w.Code)
		}
	}
}
