package groups

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/groups"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	owner  = "owner-id"
	member = "member-id"
)

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
	NewController(srv.New(store, purger)).RegisterRoutes(r)
	return r, store, purger
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

func create(t *testing.T, r http.Handler, caller, name string) models.Group {
	t.Helper()
	w := do(r, http.MethodPost, "/groups", `{"cognito_id":"`+caller+`","name":"`+name+`"}`)
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

	for _, body := range []string{`{"name":"ops"}`, `{"cognito_id":"x"}`, `{not json`} {
		if w := do(r, http.MethodPost, "/groups", body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", body, w.Code)
		}
	}
}

func TestMembers_OwnerAddsAndRemoves(t *testing.T) {
	r, store, _ := setupRouter()
	group := create(t, r, owner, "ops")
	path := "/groups/" + group.Id.String() + "/members"

	w := do(r, http.MethodPost, path, `{"cognito_id":"`+owner+`","member":"`+member+`"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("add: status = %d (body: %s)", w.Code, w.Body.String())
	}
	if got, _ := store.FindGroup(group.Id); !got.IsMember(member) {
		t.Errorf("member not added: %+v", got)
	}

	w = do(r, http.MethodPost, path, `{"cognito_id":"`+member+`","member":"someone"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("a member added someone: status = %d, want 404", w.Code)
	}

	w = do(r, http.MethodDelete, path, `{"cognito_id":"`+owner+`","member":"`+member+`"}`)
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
	do(r, http.MethodPost, "/groups/"+ops.Id.String()+"/members", `{"cognito_id":"`+owner+`","member":"`+member+`"}`)

	w := do(r, http.MethodGet, "/groups/"+ops.Id.String()+"?cognito_id="+member, "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ops"`) {
		t.Errorf("member GET: status = %d, body = %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodGet, "/groups/"+ops.Id.String()+"?cognito_id=stranger", ""); w.Code != http.StatusNotFound {
		t.Errorf("stranger GET: status = %d, want 404", w.Code)
	}
	if w := do(r, http.MethodGet, "/groups/"+ops.Id.String(), ""); w.Code != http.StatusBadRequest {
		t.Errorf("GET without caller: status = %d, want 400", w.Code)
	}

	w = do(r, http.MethodGet, "/groups?cognito_id="+member, "")
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

	if w := do(r, http.MethodDelete, "/groups/"+group.Id.String()+"?cognito_id="+member, ""); w.Code != http.StatusNotFound {
		t.Errorf("non-owner delete: status = %d, want 404", w.Code)
	}

	w := do(r, http.MethodDelete, "/groups/"+group.Id.String()+"?cognito_id="+owner, "")
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
		{http.MethodGet, "/groups/nope?cognito_id=" + owner},
		{http.MethodDelete, "/groups/nope?cognito_id=" + owner},
		{http.MethodPost, "/groups/nope/members"},
	} {
		if w := do(r, c.method, c.path, `{"cognito_id":"`+owner+`","member":"x"}`); w.Code != http.StatusBadRequest {
			t.Errorf("%s %s: status = %d, want 400", c.method, c.path, w.Code)
		}
	}
}
