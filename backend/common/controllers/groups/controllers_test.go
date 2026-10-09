package groups

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
	"github.com/Secreto31126/tesis/common/services/access"
	srv "github.com/Secreto31126/tesis/common/services/groups"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	owner  = "owner-id"
	member = "member-id"
)

// acme is the organization owner is in.
var acme = models.Org{Id: uuid.New(), Members: []models.OrgMember{{User: owner, Role: models.OrgRoleMember}}}

type purgeRecorder struct {
	purged []models.SecretScope
}

func (p *purgeRecorder) Purge(scope models.SecretScope) error {
	p.purged = append(p.purged, scope)
	return nil
}

func setupRouter() (*gin.Engine, *mocks.MemoryGroupStore, *purgeRecorder) {
	store := &mocks.MemoryGroupStore{}
	purger := &purgeRecorder{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewController(srv.New(store, purger, access.New(&mocks.MemoryOrgStore{Orgs: []models.Org{acme}}))).RegisterRoutes(r.Group("", middleware.RequireAuth(mocks.SubjectVerifier{})))
	return r, store, purger
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

func create(t *testing.T, r http.Handler, caller, name string) models.Group {
	t.Helper()
	w := do(r, caller, http.MethodPost, "/groups", `{"name":"`+name+`"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d (body: %s)", w.Code, w.Body.String())
	}
	var group models.Group
	if err := json.Unmarshal(w.Body.Bytes(), &group); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return group
}

func TestCreateGroup(t *testing.T) {
	r, _, _ := setupRouter()

	group := create(t, r, owner, "ops")
	if group.Owner != owner || group.Name != "ops" || group.Id == uuid.Nil {
		t.Errorf("group = %+v", group)
	}

	for _, body := range []string{`{}`, `{not json`} {
		if w := do(r, owner, http.MethodPost, "/groups", body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", body, w.Code)
		}
	}
}

func TestEveryRoute_RequiresAToken(t *testing.T) {
	r, _, _ := setupRouter()
	for _, route := range r.Routes() {
		path := strings.ReplaceAll(route.Path, ":id", uuid.NewString())
		if w := do(r, "", route.Method, path, ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401", route.Method, route.Path, w.Code)
		}
	}
}

func TestMembers_OwnerAddsAndRemoves(t *testing.T) {
	r, store, _ := setupRouter()
	group := create(t, r, owner, "ops")
	path := "/groups/" + group.Id.String() + "/members"

	w := do(r, owner, http.MethodPost, path, `{"member":"`+member+`"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("add: status = %d (body: %s)", w.Code, w.Body.String())
	}
	if got, _ := store.FindGroup(group.Id); !got.IsMember(member) {
		t.Errorf("member not added: %+v", got)
	}

	w = do(r, member, http.MethodPost, path, `{"member":"someone"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("a member added someone: status = %d, want 404", w.Code)
	}

	w = do(r, owner, http.MethodDelete, path, `{"member":"`+member+`"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("remove: status = %d (body: %s)", w.Code, w.Body.String())
	}
	if got, _ := store.FindGroup(group.Id); got.IsMember(member) {
		t.Errorf("member not removed: %+v", got)
	}
}

func TestGetAndList(t *testing.T) {
	r, _, _ := setupRouter()
	ops := create(t, r, owner, "ops")
	create(t, r, owner, "private")
	do(r, owner, http.MethodPost, "/groups/"+ops.Id.String()+"/members", `{"member":"`+member+`"}`)

	w := do(r, member, http.MethodGet, "/groups/"+ops.Id.String(), "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ops"`) {
		t.Errorf("member GET: status = %d, body = %s", w.Code, w.Body.String())
	}
	if w := do(r, "stranger", http.MethodGet, "/groups/"+ops.Id.String(), ""); w.Code != http.StatusNotFound {
		t.Errorf("stranger GET: status = %d, want 404", w.Code)
	}

	w = do(r, member, http.MethodGet, "/groups", "")
	var mine []models.Group
	if err := json.Unmarshal(w.Body.Bytes(), &mine); err != nil || w.Code != http.StatusOK {
		t.Fatalf("list: status = %d, body = %s", w.Code, w.Body.String())
	}
	if len(mine) != 1 || mine[0].Id != ops.Id {
		t.Errorf("member's groups = %+v, want only ops", mine)
	}
}

func TestDeleteGroup(t *testing.T) {
	r, _, purger := setupRouter()
	group := create(t, r, owner, "ops")

	if w := do(r, member, http.MethodDelete, "/groups/"+group.Id.String(), ""); w.Code != http.StatusNotFound {
		t.Errorf("non-owner delete: status = %d, want 404", w.Code)
	}

	w := do(r, owner, http.MethodDelete, "/groups/"+group.Id.String(), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d (body: %s)", w.Code, w.Body.String())
	}
	if len(purger.purged) != 1 || purger.purged[0] != models.GroupScope(group.Id) {
		t.Errorf("purged %v, want the group's scope", purger.purged)
	}
}

func TestInvalidUUID(t *testing.T) {
	r, _, _ := setupRouter()

	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/groups/nope"},
		{http.MethodDelete, "/groups/nope"},
		{http.MethodPost, "/groups/nope/members"},
	} {
		if w := do(r, owner, c.method, c.path, `{"member":"x"}`); w.Code != http.StatusBadRequest {
			t.Errorf("%s %s: status = %d, want 400", c.method, c.path, w.Code)
		}
	}
}

func TestCreateGroup_InAnOrg(t *testing.T) {
	r, _, _ := setupRouter()

	w := do(r, owner, http.MethodPost, "/groups", `{"name":"ops","org":"`+acme.Id.String()+`"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d (body: %s)", w.Code, w.Body.String())
	}
	var group models.Group
	_ = json.Unmarshal(w.Body.Bytes(), &group)
	if group.Org == nil || *group.Org != acme.Id || group.Role != models.GroupOwner {
		t.Errorf("group = %+v", group)
	}

	w = do(r, member, http.MethodPost, "/groups", `{"name":"ops","org":"`+acme.Id.String()+`"}`)
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "org_not_found") {
		t.Errorf("outside the org: %d %s, want 404 org_not_found", w.Code, w.Body.String())
	}
}
