package orgs

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/orgs"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// The bearer token is the user id (mocks.SubjectVerifier).
var (
	anaUser = models.User{Id: uuid.New(), Name: "Ana", Email: "ana@example.com"}
	bobUser = models.User{Id: uuid.New(), Name: "Bob", Email: "bob@example.com"}
	eveUser = models.User{Id: uuid.New(), Name: "Eve", Email: "eve@example.com"}

	ana = anaUser.Id.String()
	bob = bobUser.Id.String()
	eve = eveUser.Id.String()
)

func setupRouter() (*gin.Engine, *mocks.MemoryOrgStore) {
	store := &mocks.MemoryOrgStore{Boards: map[uuid.UUID]int64{}, Groups: map[uuid.UUID]int64{}}
	users := &mocks.MemoryUserStore{Users: []models.User{anaUser, bobUser, eveUser}}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewController(srv.New(store, users)).RegisterRoutes(r.Group("", middleware.RequireAuth(mocks.SubjectVerifier{})))
	return r, store
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

func decode[T any](t *testing.T, w *httptest.ResponseRecorder, status int) T {
	t.Helper()
	var out T
	if w.Code != status {
		t.Fatalf("status = %d, want %d (body: %s)", w.Code, status, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func errorOf(w *httptest.ResponseRecorder) string {
	var body struct{ Error string }
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return body.Error
}

// acme is ana's, with bob as a member.
func acme(t *testing.T, r http.Handler) string {
	t.Helper()
	org := decode[models.Org](t, do(r, ana, http.MethodPost, "/orgs", `{"name":"acme"}`), http.StatusCreated)
	if org.Role != models.OrgRoleAdmin {
		t.Errorf("creator's role = %q, want admin", org.Role)
	}
	decode[models.OrgMemberSummary](t, do(r, ana, http.MethodPut, "/orgs/"+org.Id.String()+"/members",
		`{"email":"bob@example.com","role":"member"}`), http.StatusOK)
	return "/orgs/" + org.Id.String()
}

func TestCreateAndList(t *testing.T) {
	r, _ := setupRouter()
	path := acme(t, r)

	for caller, want := range map[string]models.OrgRole{ana: models.OrgRoleAdmin, bob: models.OrgRoleMember} {
		mine := decode[[]models.Org](t, do(r, caller, http.MethodGet, "/orgs", ""), http.StatusOK)
		if len(mine) != 1 || "/orgs/"+mine[0].Id.String() != path || mine[0].Role != want {
			t.Errorf("%s lists %+v", caller, mine)
		}
	}
	if mine := decode[[]models.Org](t, do(r, eve, http.MethodGet, "/orgs", ""), http.StatusOK); len(mine) != 0 {
		t.Errorf("eve lists %+v", mine)
	}

	for body, want := range map[string]int{`{}`: http.StatusBadRequest, `{"name":"  "}`: http.StatusBadRequest} {
		if w := do(r, ana, http.MethodPost, "/orgs", body); w.Code != want {
			t.Errorf("%s: status = %d, want %d", body, w.Code, want)
		}
	}
	if w := do(r, "", http.MethodGet, "/orgs", ""); w.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: status = %d, want 401", w.Code)
	}
}

func TestGetOrg(t *testing.T) {
	r, _ := setupRouter()
	path := acme(t, r)

	detail := decode[models.OrgDetail](t, do(r, bob, http.MethodGet, path, ""), http.StatusOK)
	if detail.Role != models.OrgRoleMember || len(detail.Members) != 2 || detail.Members[1].User.Email != bobUser.Email {
		t.Errorf("detail = %+v", detail)
	}

	if w := do(r, eve, http.MethodGet, path, ""); w.Code != http.StatusNotFound {
		t.Errorf("a stranger: status = %d, want 404", w.Code)
	}
	if w := do(r, ana, http.MethodGet, "/orgs/not-a-uuid", ""); w.Code != http.StatusBadRequest {
		t.Errorf("bad id: status = %d, want 400", w.Code)
	}
}

func TestRenameOrg(t *testing.T) {
	r, store := setupRouter()
	path := acme(t, r)

	cases := []struct {
		caller, body string
		status       int
		code         string
	}{
		{bob, `{"name":"mine"}`, http.StatusNotFound, "Organization not found"},
		{ana, `{"name":"   "}`, http.StatusBadRequest, "invalid_name"},
		{ana, `{"name":"acme inc"}`, http.StatusNoContent, ""},
	}
	for _, c := range cases {
		w := do(r, c.caller, http.MethodPatch, path, c.body)
		if w.Code != c.status || errorOf(w) != c.code {
			t.Errorf("%s %s: %d %q, want %d %q", c.caller, c.body, w.Code, errorOf(w), c.status, c.code)
		}
	}
	if store.Orgs[0].Name != "acme inc" {
		t.Errorf("name = %q", store.Orgs[0].Name)
	}
}

func TestMembers(t *testing.T) {
	r, store := setupRouter()
	path := acme(t, r)

	cases := []struct {
		caller, method, path, body string
		status                     int
		code                       string
	}{
		{bob, http.MethodPut, path + "/members", `{"email":"eve@example.com","role":"member"}`, http.StatusNotFound, "Organization not found"},
		{ana, http.MethodPut, path + "/members", `{"email":"nobody@example.com","role":"member"}`, http.StatusNotFound, "user_not_found"},
		{ana, http.MethodPut, path + "/members", `{"email":"eve@example.com","role":"owner"}`, http.StatusBadRequest, "invalid_role"},
		{ana, http.MethodPut, path + "/members", `{"email":"ana@example.com","role":"member"}`, http.StatusConflict, "last_admin"},
		{ana, http.MethodDelete, path + "/members/" + ana, "", http.StatusConflict, "last_admin"},
		{bob, http.MethodDelete, path + "/members/" + ana, "", http.StatusNotFound, "Organization not found"},
		{eve, http.MethodDelete, path + "/members/" + eve, "", http.StatusNotFound, "Organization not found"},
		{ana, http.MethodPut, path + "/members", `{"email":"EVE@example.com","role":"admin"}`, http.StatusOK, ""},
		{bob, http.MethodDelete, path + "/members/" + bob, "", http.StatusNoContent, ""},
	}
	for _, c := range cases {
		w := do(r, c.caller, c.method, c.path, c.body)
		if w.Code != c.status || errorOf(w) != c.code {
			t.Errorf("%s %s %s %s: %d %q, want %d %q", c.caller, c.method, c.path, c.body, w.Code, errorOf(w), c.status, c.code)
		}
	}

	members := store.Orgs[0].Members
	if len(members) != 2 || store.Orgs[0].RoleOf(eve) != models.OrgRoleAdmin || store.Orgs[0].RoleOf(bob) != "" {
		t.Errorf("members = %+v, want ana and eve as admins", members)
	}
}

func TestDeleteOrg(t *testing.T) {
	r, store := setupRouter()
	path := acme(t, r)
	id := store.Orgs[0].Id

	store.Boards[id] = 1
	if w := do(r, ana, http.MethodDelete, path, ""); w.Code != http.StatusConflict || errorOf(w) != "org_not_empty" {
		t.Errorf("with a board: %d %s, want 409 org_not_empty", w.Code, w.Body.String())
	}
	store.Boards[id] = 0

	if w := do(r, bob, http.MethodDelete, path, ""); w.Code != http.StatusNotFound {
		t.Errorf("a member: status = %d, want 404", w.Code)
	}
	if w := do(r, ana, http.MethodDelete, path, ""); w.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d (body: %s)", w.Code, w.Body.String())
	}
	if w := do(r, ana, http.MethodGet, path, ""); w.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want 404", w.Code)
	}
}
