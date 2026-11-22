package postits

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func previewBody(board uuid.UUID, extra string) string {
	return `{"board":"` + board.String() + `","resource":"https://example.com/fact"` + extra + `}`
}

func TestPreviewPostIt_AnswersTheWholeResponse(t *testing.T) {
	run := &mocks.MockExecuter{ExecuteFn: func(*models.PostIts) (any, error) {
		return map[string]any{"response": map[string]any{"fact": "cats sleep"}}, nil
	}}
	r := setupRouter(nil, nil, run)

	w := do(r, http.MethodPost, "/post-its/preview", previewBody(uuid.New(), ""))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", w.Header().Get("Cache-Control"))
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if fmt.Sprint(got) != "map[response:map[fact:cats sleep]]" {
		t.Errorf("body = %v", got)
	}
}

func TestPreviewPostIt_AnswersTheOutputs(t *testing.T) {
	run := &mocks.MockExecuter{ExecuteFn: func(*models.PostIts) (any, error) {
		return map[string]any{"fact": "cats sleep"}, nil
	}}
	r := setupRouter(nil, nil, run)

	w := do(r, http.MethodPost, "/post-its/preview", previewBody(uuid.New(), `,"response":"json","query":{"fact":".fact"}`))
	if w.Code != http.StatusOK || w.Body.String() != `{"outputs":{"fact":"cats sleep"}}` {
		t.Errorf("got %d %s", w.Code, w.Body.String())
	}
}

func TestPreviewPostIt_SaysWhatToCheck(t *testing.T) {
	run := &mocks.MockExecuter{ExecuteFn: func(*models.PostIts) (any, error) {
		return nil, &infrastructure.SourceStatusError{Status: http.StatusUnauthorized}
	}}
	r := setupRouter(nil, nil, run)

	w := do(r, http.MethodPost, "/post-its/preview", previewBody(uuid.New(), ""))
	var got struct {
		Code   string `json:"code"`
		Status int    `json:"status"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if w.Code != http.StatusBadGateway || got.Code != "source_status" || got.Status != http.StatusUnauthorized {
		t.Errorf("got %d %s", w.Code, w.Body.String())
	}
}

func TestPreviewPostIt_RefusesBadRequests(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	cases := map[string]string{
		"no resource":  `{"board":"` + uuid.NewString() + `"}`,
		"not http(s)":  `{"board":"` + uuid.NewString() + `","resource":"ftp://x"}`,
		"bad response": previewBody(uuid.New(), `,"response":"xml"`),
		"bad method":   previewBody(uuid.New(), `,"request":{"method":"DELETE"}`),
	}
	for name, body := range cases {
		if w := do(r, http.MethodPost, "/post-its/preview", body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400 (body: %s)", name, w.Code, w.Body.String())
		}
	}
}

func TestPreviewPostIt_StrangersGet404(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	if w := doAs(r, "stranger", http.MethodPost, "/post-its/preview", previewBody(uuid.New(), "")); w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
