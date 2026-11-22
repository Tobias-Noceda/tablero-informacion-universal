package postits

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	srv "github.com/Secreto31126/tesis/common/services/postits"
	"github.com/gin-gonic/gin"
)

func failWith(t *testing.T, err error) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	fail(c, err)

	var body map[string]any
	if decodeErr := json.Unmarshal(w.Body.Bytes(), &body); decodeErr != nil {
		t.Fatalf("body %q: %v", w.Body.String(), decodeErr)
	}
	return w.Code, body
}

// A card that cannot run says why with a code the board turns into what to
// check, and keeps the message an agent reads through the MCP server.
func TestFail_SaysWhyACardCouldNotRun(t *testing.T) {
	for name, c := range map[string]struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		"source status":          {&infrastructure.SourceStatusError{Status: 404}, http.StatusBadGateway, "source_status"},
		"source timeout":         {fmt.Errorf("%w: deadline", infrastructure.ErrSourceTimeout), http.StatusGatewayTimeout, "source_timeout"},
		"source unreachable":     {fmt.Errorf("%w: refused", infrastructure.ErrSourceUnreachable), http.StatusBadGateway, "source_unreachable"},
		"source unreadable":      {fmt.Errorf("%w: bad json", infrastructure.ErrSourceUnreadable), http.StatusBadGateway, "source_unreadable"},
		"query failed":           {fmt.Errorf("%w: no results", infrastructure.ErrQueryFailed), http.StatusUnprocessableEntity, "query_failed"},
		"credential unavailable": {srv.ErrCredentialUnavailable, http.StatusForbidden, "credential_unavailable"},
		"platform credential":    {srv.ErrSystemSecretMissing, http.StatusServiceUnavailable, "card_unavailable"},
	} {
		t.Run(name, func(t *testing.T) {
			status, body := failWith(t, c.err)

			if status != c.wantStatus || body["code"] != c.wantCode {
				t.Errorf("got %d %v, want %d code %q", status, body, c.wantStatus, c.wantCode)
			}
			if body["error"] != c.err.Error() {
				t.Errorf("error = %v, want the message %q", body["error"], c.err.Error())
			}
		})
	}
}

func TestFail_NamesTheSourcesStatus(t *testing.T) {
	_, body := failWith(t, &infrastructure.SourceStatusError{Status: 401})

	if body["status"] != float64(401) {
		t.Errorf("status = %v, want the source's 401", body["status"])
	}
}

func TestFail_OtherErrorsHaveNoCode(t *testing.T) {
	status, body := failWith(t, srv.ErrNotAMember)

	if status != http.StatusNotFound {
		t.Errorf("status = %d, want 404", status)
	}
	if _, has := body["code"]; has {
		t.Errorf("body = %v, want no code", body)
	}
}
