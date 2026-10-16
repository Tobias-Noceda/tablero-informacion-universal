package mail

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

const testFrom = "Tablero <no-reply@tiu.example>"

type resendCall struct {
	method, path, authorization string
	body                        map[string]any
}

// fakeResend answers like the Resend API and records what it was sent.
func fakeResend(t *testing.T, status int, response string) (*ResendSender, *[]resendCall) {
	t.Helper()
	calls := &[]resendCall{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := resendCall{method: r.Method, path: r.URL.Path, authorization: r.Header.Get("Authorization")}
		_ = json.NewDecoder(r.Body).Decode(&call.body)
		*calls = append(*calls, call)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)

	sender := NewResendSender("re_test_key", testFrom)
	sender.client.BaseURL, _ = url.Parse(server.URL + "/")
	return sender, calls
}

func message() Message {
	return Message{To: "ana@example.com", Subject: "Verify your email", Text: "Hi Ana", HTML: "<p>Hi Ana</p>"}
}

func TestResendSender_PostsTheRenderedMail(t *testing.T) {
	sender, calls := fakeResend(t, http.StatusOK, `{"id":"email-1"}`)

	if err := sender.Deliver(context.Background(), message()); err != nil {
		t.Fatal(err)
	}

	if len(*calls) != 1 {
		t.Fatalf("calls = %d", len(*calls))
	}
	call := (*calls)[0]
	if call.method != http.MethodPost || call.path != "/emails" || call.authorization != "Bearer re_test_key" {
		t.Errorf("request = %s %s (%s)", call.method, call.path, call.authorization)
	}
	want := map[string]any{
		"from":    testFrom,
		"to":      []any{"ana@example.com"},
		"subject": "Verify your email",
		"text":    "Hi Ana",
		"html":    "<p>Hi Ana</p>",
	}
	for field, value := range want {
		if got, _ := json.Marshal(call.body[field]); string(got) != mustJSON(value) {
			t.Errorf("%s = %s, want %s", field, got, mustJSON(value))
		}
	}
}

func TestResendSender_ReportsARefusalWithoutTheKey(t *testing.T) {
	sender, _ := fakeResend(t, http.StatusUnprocessableEntity, `{"statusCode":422,"name":"validation_error","message":"The from domain is not verified"}`)

	err := sender.Deliver(context.Background(), message())
	if err == nil {
		t.Fatal("expected the refusal")
	}
	if !strings.Contains(err.Error(), "not verified") {
		t.Errorf("err = %v", err)
	}
	if strings.Contains(err.Error(), "re_test_key") {
		t.Errorf("the API key leaked into the error: %v", err)
	}
}

func TestResendSender_GivesUpAtTheDeadline(t *testing.T) {
	blocked := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-blocked:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() { close(blocked); server.Close() })
	sender := NewResendSender("re_test_key", testFrom)
	sender.client.BaseURL, _ = url.Parse(server.URL + "/")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := sender.Deliver(ctx, message()); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want the deadline", err)
	}
}

func mustJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
