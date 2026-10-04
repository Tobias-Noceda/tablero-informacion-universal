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
	"github.com/Secreto31126/tesis/common/services/access"
	b_srv "github.com/Secreto31126/tesis/common/services/boards"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Every request below is made as one of these: the board is owned by owner,
// edited by ana and viewed by bob; eve is a stranger. The bearer token is the
// user id (mocks.SubjectVerifier).
var (
	ownerUser = models.User{Id: uuid.New(), Name: "Owner", Email: "owner@example.com"}
	anaUser   = models.User{Id: uuid.New(), Name: "Ana", Email: "ana@example.com"}
	bobUser   = models.User{Id: uuid.New(), Name: "Bob", Email: "bob@example.com"}
	eveUser   = models.User{Id: uuid.New(), Name: "Eve", Email: "eve@example.com"}

	owner    = ownerUser.Id.String()
	ana      = anaUser.Id.String()
	bob      = bobUser.Id.String()
	outsider = eveUser.Id.String()

	boardID = uuid.New()
	board   = "/boards/" + boardID.String()

	// acme is the organization ana is in.
	acme = models.Org{Id: uuid.New(), Members: []models.OrgMember{{User: ana, Role: models.OrgRoleMember}}}
)

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
			return &models.Board{Id: id, Name: "B", Owner: owner, Members: []models.BoardMember{
				{User: ana, Role: models.BoardEditor},
				{User: bob, Role: models.BoardViewer},
			}}, nil
		}
	}
	return db
}

func setupRouter(db *mocks.MockDB) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	users := &mocks.MemoryUserStore{Users: []models.User{ownerUser, anaUser, bobUser, eveUser}}
	bs := b_srv.New(withBoard(db), &mocks.MockScopePurger{}, access.New(&mocks.MemoryOrgStore{Orgs: []models.Org{acme}}), users, &mocks.CountingLimiter{})

	NewController(bs).RegisterRoutes(r.Group("", middleware.RequireAuth(mocks.SubjectVerifier{})))
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

func TestCreateBoard_TheCallerOwnsIt(t *testing.T) {
	id := uuid.New()
	var gotOwner string
	db := &mocks.MockDB{
		CreateBoardFn: func(name, owner string, org *uuid.UUID) (*models.Board, error) {
			gotOwner = owner
			return &models.Board{Id: id, Name: name, Owner: owner}, nil
		},
	}

	w := do(setupRouter(db), ana, http.MethodPost, "/boards", `{"name":"B","owner":"someone-else"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	if gotOwner != ana {
		t.Errorf("owner = %q, want the caller", gotOwner)
	}
	var got struct {
		Id   uuid.UUID
		Role models.BoardRole
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Id != id || got.Role != models.BoardOwner {
		t.Errorf("got %+v, want the board as its owner", got)
	}
}

func TestCreateBoard_MissingName(t *testing.T) {
	w := do(setupRouter(nil), ana, http.MethodPost, "/boards", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetUserBoards_ListsTheCallersBoards(t *testing.T) {
	db := &mocks.MockDB{
		FindUserBoardsFn: func(user string, orgs []uuid.UUID) ([]models.Board, error) {
			if user != ana {
				t.Errorf("user = %q, want the caller", user)
			}
			return []models.Board{{Id: uuid.New()}}, nil
		},
	}

	w := do(setupRouter(db), ana, http.MethodGet, "/boards", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestGetBoard_CarriesTheCallersRole(t *testing.T) {
	w := do(setupRouter(nil), bob, http.MethodGet, board, "")
	var got struct{ Role models.BoardRole }
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if w.Code != http.StatusOK || got.Role != models.BoardViewer {
		t.Fatalf("status = %d, role = %q, want 200 as viewer", w.Code, got.Role)
	}
}

func TestEveryRoute_RequiresAToken(t *testing.T) {
	r := setupRouter(nil)
	for _, route := range r.Routes() {
		path := strings.NewReplacer(":id", boardID.String(), ":strand", uuid.NewString(), ":user", ana).Replace(route.Path)
		if w := do(r, "", route.Method, path, ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401", route.Method, route.Path, w.Code)
		}
	}
}

func okDB() *mocks.MockDB {
	return &mocks.MockDB{
		FindBoardPostItsFn:  func(uuid.UUID) ([]models.PostIts, error) { return []models.PostIts{}, nil },
		ConnectPostItsFn:    func(_, s, t uuid.UUID) (*models.Strand, error) { return &models.Strand{Source: s, Target: t}, nil },
		DisconnectPostItsFn: func(_, _ uuid.UUID) error { return nil },
		DeleteBoardFn:       func(uuid.UUID) error { return nil },
		UpdateBoardNameFn:   func(uuid.UUID, string) error { return nil },
		SetBoardMemberFn:    func(uuid.UUID, string, models.BoardRole) error { return nil },
		RemoveBoardMemberFn: func(uuid.UUID, string) error { return nil },
	}
}

type call struct {
	name, method, path, body string
	want                     int
}

var strand = `{"source":"` + uuid.NewString() + `","target":"` + uuid.NewString() + `"}`

// Every route with the least role it needs and what it answers then.
var calls = []struct {
	call
	min models.BoardRole
}{
	{call{"GetBoard", http.MethodGet, board, "", http.StatusOK}, models.BoardViewer},
	{call{"GetBoardPostIts", http.MethodGet, board + "/post-its", "", http.StatusOK}, models.BoardViewer},
	{call{"GetMembers", http.MethodGet, board + "/members", "", http.StatusOK}, models.BoardViewer},
	{call{"ConnectPostIts", http.MethodPost, board + "/strands", strand, http.StatusCreated}, models.BoardEditor},
	{call{"DisconnectPostIts", http.MethodDelete, board + "/strands/" + uuid.NewString(), "", http.StatusNoContent}, models.BoardEditor},
	{call{"UpdateBoardName", http.MethodPatch, board + "/name", `{"name":"N"}`, http.StatusNoContent}, models.BoardOwner},
	{call{"SetMember", http.MethodPut, board + "/members", `{"email":"eve@example.com","role":"viewer"}`, http.StatusOK}, models.BoardOwner},
	{call{"RemoveMember", http.MethodDelete, board + "/members/" + uuid.NewString(), "", http.StatusNoContent}, models.BoardOwner},
	{call{"DeleteBoard", http.MethodDelete, board, "", http.StatusNoContent}, models.BoardOwner},
}

// Below the role a route needs, the board answers as if it did not exist.
func TestRoutes_ByRole(t *testing.T) {
	roles := map[string]models.BoardRole{owner: models.BoardOwner, ana: models.BoardEditor, bob: models.BoardViewer, outsider: ""}

	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			r := setupRouter(okDB())
			for caller, role := range roles {
				want := http.StatusNotFound
				if role.AtLeast(tc.min) {
					want = tc.want
				}
				if w := do(r, caller, tc.method, tc.path, tc.body); w.Code != want {
					t.Errorf("as %q: status = %d, want %d (body: %s)", role, w.Code, want, w.Body.String())
				}
			}
		})
	}
}

func TestMissingBoard_IsNotFound(t *testing.T) {
	w := do(setupRouter(nil), owner, http.MethodGet, "/boards/"+uuid.NewString(), "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestGetMembers_ListsWhoIsOnTheBoard(t *testing.T) {
	w := do(setupRouter(nil), bob, http.MethodGet, board+"/members", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var members []models.BoardMemberSummary
	_ = json.Unmarshal(w.Body.Bytes(), &members)
	if len(members) != 3 || members[0].User.Email != ownerUser.Email || members[0].Role != models.BoardOwner ||
		members[1].Role != models.BoardEditor || members[2].User.Name != "Bob" {
		t.Errorf("members = %+v", members)
	}
}

func TestSetMember_NamesTheUserByEmail(t *testing.T) {
	var added string
	var role models.BoardRole
	db := okDB()
	db.SetBoardMemberFn = func(_ uuid.UUID, user string, r models.BoardRole) error {
		added, role = user, r
		return nil
	}

	w := do(setupRouter(db), owner, http.MethodPut, board+"/members", `{"email":"Eve@Example.com","role":"editor"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if added != outsider || role != models.BoardEditor {
		t.Errorf("stored (%q, %q), want (eve, editor)", added, role)
	}
	var member models.BoardMemberSummary
	_ = json.Unmarshal(w.Body.Bytes(), &member)
	if member.User.Id != eveUser.Id || member.Role != models.BoardEditor {
		t.Errorf("member = %+v", member)
	}
}

func TestSetMember_ErrorCodes(t *testing.T) {
	cases := []struct {
		body string
		code int
		err  string
	}{
		{`{"email":"nobody@example.com","role":"viewer"}`, http.StatusNotFound, "user_not_found"},
		{`{"email":"eve@example.com","role":"owner"}`, http.StatusBadRequest, "invalid_role"},
		{`{"email":"eve@example.com","role":"admin"}`, http.StatusBadRequest, "invalid_role"},
		{`{"email":"owner@example.com","role":"viewer"}`, http.StatusBadRequest, "owner_role"},
		{`{"role":"viewer"}`, http.StatusBadRequest, ""},
	}

	for _, c := range cases {
		w := do(setupRouter(okDB()), owner, http.MethodPut, board+"/members", c.body)
		if w.Code != c.code || (c.err != "" && !strings.Contains(w.Body.String(), `"`+c.err+`"`)) {
			t.Errorf("%s: %d %s, want %d %s", c.body, w.Code, w.Body.String(), c.code, c.err)
		}
	}
}

func TestRemoveMember_AMemberMayLeave(t *testing.T) {
	w := do(setupRouter(okDB()), bob, http.MethodDelete, board+"/members/"+bob, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
}

func TestRemoveMember_TheOwnerIsABadRequest(t *testing.T) {
	w := do(setupRouter(okDB()), owner, http.MethodDelete, board+"/members/"+owner, "")
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "owner_role") {
		t.Fatalf("status = %d %s, want 400 owner_role", w.Code, w.Body.String())
	}
}

func TestBoardHandlers_ServiceErrors(t *testing.T) {
	boom := errors.New("mongo: connection lost")
	db := &mocks.MockDB{
		FindUserBoardsFn:    func(string, []uuid.UUID) ([]models.Board, error) { return nil, boom },
		CreateBoardFn:       func(string, string, *uuid.UUID) (*models.Board, error) { return nil, boom },
		DeleteBoardFn:       func(uuid.UUID) error { return boom },
		SetBoardMemberFn:    func(uuid.UUID, string, models.BoardRole) error { return boom },
		RemoveBoardMemberFn: func(uuid.UUID, string) error { return boom },
		UpdateBoardNameFn:   func(uuid.UUID, string) error { return boom },
		ConnectPostItsFn:    func(_, _, _ uuid.UUID) (*models.Strand, error) { return nil, boom },
		DisconnectPostItsFn: func(_, _ uuid.UUID) error { return boom },
	}
	cases := []call{
		{"GetUserBoards", http.MethodGet, "/boards", "", 0},
		{"CreateBoard", http.MethodPost, "/boards", `{"name":"B"}`, 0},
	}
	for _, c := range calls {
		if c.min != models.BoardViewer {
			cases = append(cases, c.call)
		}
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if w := do(setupRouter(db), owner, tc.method, tc.path, tc.body); w.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500 (body: %s)", w.Code, w.Body.String())
			}
		})
	}
}

func TestBoardHandlers_InvalidUUID(t *testing.T) {
	cases := []call{
		{"GetBoard", http.MethodGet, "/boards/nope", "", 0},
		{"GetBoardPostIts", http.MethodGet, "/boards/nope/post-its", "", 0},
		{"GetMembers", http.MethodGet, "/boards/nope/members", "", 0},
		{"DeleteBoard", http.MethodDelete, "/boards/nope", "", 0},
		{"SetMember", http.MethodPut, "/boards/nope/members", `{"email":"eve@example.com","role":"viewer"}`, 0},
		{"RemoveMember", http.MethodDelete, "/boards/nope/members/" + ana, "", 0},
		{"UpdateBoardName", http.MethodPatch, "/boards/nope/name", `{"name":"N"}`, 0},
		{"ConnectPostIts", http.MethodPost, "/boards/nope/strands", strand, 0},
		{"DisconnectPostIts board", http.MethodDelete, "/boards/nope/strands/" + uuid.NewString(), "", 0},
		{"DisconnectPostIts strand", http.MethodDelete, board + "/strands/jajant", "", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if w := do(setupRouter(nil), owner, tc.method, tc.path, tc.body); w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
		})
	}
}

func TestBoardHandlers_BadBody(t *testing.T) {
	for _, c := range []struct{ method, path string }{
		{http.MethodPut, board + "/members"},
		{http.MethodPatch, board + "/name"},
		{http.MethodPost, board + "/strands"},
	} {
		if w := do(setupRouter(nil), owner, c.method, c.path, `{bad json`); w.Code != http.StatusBadRequest {
			t.Errorf("%s %s: status = %d, want 400", c.method, c.path, w.Code)
		}
	}
}

func TestCreateBoard_InAnOrg(t *testing.T) {
	var gotOrg *uuid.UUID
	db := &mocks.MockDB{
		CreateBoardFn: func(name, owner string, org *uuid.UUID) (*models.Board, error) {
			gotOrg = org
			return &models.Board{Id: uuid.New(), Name: name, Owner: owner, Org: org}, nil
		},
	}
	r := setupRouter(db)

	w := do(r, ana, http.MethodPost, "/boards", `{"name":"B","org":"`+acme.Id.String()+`"}`)
	if w.Code != http.StatusCreated || gotOrg == nil || *gotOrg != acme.Id {
		t.Fatalf("status = %d, org = %v (body: %s)", w.Code, gotOrg, w.Body.String())
	}
	var got models.Board
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Org == nil || *got.Org != acme.Id {
		t.Errorf("response org = %v", got.Org)
	}

	gotOrg = nil
	for _, c := range []struct{ caller, org string }{{bob, acme.Id.String()}, {ana, uuid.NewString()}} {
		w := do(r, c.caller, http.MethodPost, "/boards", `{"name":"B","org":"`+c.org+`"}`)
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "org_not_found") || gotOrg != nil {
			t.Errorf("%s in %s: %d %s, want 404 org_not_found", c.caller, c.org, w.Code, w.Body.String())
		}
	}
}
