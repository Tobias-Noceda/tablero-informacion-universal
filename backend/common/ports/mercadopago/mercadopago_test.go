package mercadopago

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
)

// Vectors computed by hand with HMAC-SHA256, key "secret-key".
const (
	secret    = "secret-key"
	ts        = "1704908010"
	full      = "9ecb9c5f8d4060ef91336ffa599a0c7f456deebffbb4c9125df70286195e49b4" // id:123456;request-id:req-1;ts:...;
	noRequest = "171ec23feae8eb5bbe9c8e673ef230e12d4778345e4776d7536d8bdd1d45e5d0" // id:123456;ts:...;
	noID      = "41ddc76958f8f723d3feae93baf8bfab0869e7a177d0a8691b5c2503451099f2" // request-id:req-1;ts:...;
	lowered   = "d2e5132805dd958fba625eb7feff30d97da619119daf46da27fed7e41ec1283b" // id:abc123;request-id:req-1;ts:...;
)

// vectorNow is the moment the vectors above were signed (ts is milliseconds).
func vectorNow() time.Time { return time.UnixMilli(1704908010) }

func TestValidSignature(t *testing.T) {
	cases := map[string]struct {
		header, request, data, secret string
		want                          bool
	}{
		"valid":                {"ts=" + ts + ",v1=" + full, "req-1", "123456", secret, true},
		"spaces around parts":  {"ts=" + ts + ", v1=" + full, "req-1", "123456", secret, true},
		"tampered ts":          {"ts=1704908011,v1=" + full, "req-1", "123456", secret, false},
		"tampered id":          {"ts=" + ts + ",v1=" + full, "req-1", "123457", secret, false},
		"missing request id":   {"ts=" + ts + ",v1=" + full, "", "123456", secret, false},
		"request id left out":  {"ts=" + ts + ",v1=" + noRequest, "", "123456", secret, true},
		"data id left out":     {"ts=" + ts + ",v1=" + noID, "req-1", "", secret, true},
		"uppercase data id":    {"ts=" + ts + ",v1=" + lowered, "req-1", "ABC123", secret, true},
		"wrong secret":         {"ts=" + ts + ",v1=" + full, "req-1", "123456", "other", false},
		"no secret configured": {"ts=" + ts + ",v1=" + full, "req-1", "123456", "", false},
		"no v1":                {"ts=" + ts, "req-1", "123456", secret, false},
		"no ts":                {"v1=" + full, "req-1", "123456", secret, false},
		"not hex":              {"ts=" + ts + ",v1=zz", "req-1", "123456", secret, false},
		"empty header":         {"", "req-1", "123456", secret, false},
		"truncated hash":       {"ts=" + ts + ",v1=" + full[:20], "req-1", "123456", secret, false},
	}
	for name, c := range cases {
		if got := validSignature(c.header, c.request, c.data, c.secret, vectorNow); got != c.want {
			t.Errorf("%s: %v, want %v", name, got, c.want)
		}
	}
}

// sign is a header the way Mercado Pago builds it, for a call made at ts.
func sign(ts, request, data string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("id:" + data + ";request-id:" + request + ";ts:" + ts + ";"))
	return "ts=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestGateway_ValidSignatureUsesItsSecret(t *testing.T) {
	g := New("token", secret)
	fresh := strconv.FormatInt(time.Now().UnixMilli(), 10)
	header := sign(fresh, "req-1", "123456")
	if !g.ValidSignature(header, "req-1", "123456") || g.ValidSignature(header, "req-1", "999") {
		t.Error("gateway signature check")
	}
	if New("token", "").ValidSignature(header, "req-1", "123456") {
		t.Error("no secret must refuse everything")
	}
}

func TestGateway_ValidSignatureRefusesStaleOrFutureCalls(t *testing.T) {
	g := New("token", secret)
	at := func(d time.Duration) string { return strconv.FormatInt(time.Now().Add(d).UnixMilli(), 10) }

	for name, c := range map[string]struct {
		ts   string
		want bool
	}{
		"now":             {at(0), true},
		"4m ago":          {at(-4 * time.Minute), true},
		"6m ago":          {at(-6 * time.Minute), false},
		"6m ahead":        {at(6 * time.Minute), false},
		"4m ahead":        {at(4 * time.Minute), true},
		"the vector's ts": {ts, false},
		"not a number":    {"soon", false},
	} {
		if got := g.ValidSignature(sign(c.ts, "req-1", "123456"), "req-1", "123456"); got != c.want {
			t.Errorf("%s: %v, want %v", name, got, c.want)
		}
	}
}

// rewrite sends every request to the test server instead of Mercado Pago.
type rewrite struct{ to *url.URL }

func (r rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != "api.mercadopago.com" {
		return nil, fmt.Errorf("unexpected host %s", req.URL.Host)
	}
	req = req.Clone(req.Context())
	req.URL.Scheme, req.URL.Host = r.to.Scheme, r.to.Host
	return http.DefaultTransport.RoundTrip(req)
}

func server(t *testing.T, handler http.HandlerFunc) *Gateway {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return newGateway("tok", secret, &http.Client{Transport: rewrite{to: u}})
}

func TestCreateSubscription(t *testing.T) {
	g := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/preapproval" || r.Header.Get("Authorization") != "Bearer tok" || r.Header.Get("X-Idempotency-Key") != "key-1" {
			t.Errorf("request = %s %s %v", r.Method, r.URL.Path, r.Header)
		}
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		rec := body["auto_recurring"].(map[string]any)
		if body["status"] != "pending" || body["external_reference"] != "u1:pro" || body["payer_email"] != "a@b.co" || body["back_url"] != "https://x/billing/return" ||
			rec["frequency"] != 1.0 || rec["frequency_type"] != "months" || rec["transaction_amount"] != 14100.0 || rec["currency_id"] != "ARS" {
			t.Errorf("body = %s", raw)
		}
		_, _ = w.Write([]byte(`{"id":"pre-1","init_point":"https://mp/checkout","status":"pending","external_reference":"u1:pro","auto_recurring":{"transaction_amount":14100}}`))
	})

	sub, err := g.CreateSubscription(infrastructure.NewSubscription{Key: "key-1", Reason: "Pro", ExternalReference: "u1:pro", PayerEmail: "a@b.co", AmountARS: 14100, BackURL: "https://x/billing/return"})
	if err != nil || sub.Id != "pre-1" || sub.InitPoint != "https://mp/checkout" || sub.AmountARS != 14100 {
		t.Errorf("sub = %+v, %v", sub, err)
	}
}

func TestGetSubscriptionParsesDateAndBothSpellings(t *testing.T) {
	g := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/preapproval/pre-1" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"pre-1","status":"cancelled","external_reference":"u1:pro","next_payment_date":"2026-12-14T10:00:00.000-04:00","auto_recurring":{"transaction_amount":14100}}`))
	})
	sub, err := g.GetSubscription("pre-1")
	if err != nil || sub.Status != "cancelled" || sub.NextPayment == nil || sub.NextPayment.UTC().Hour() != 14 {
		t.Errorf("sub = %+v, %v", sub, err)
	}
}

func TestGetInvoice(t *testing.T) {
	g := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/authorized_payments/77" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":77,"preapproval_id":"pre-1","status":"processed","payment":{"id":1,"status":"approved"}}`))
	})
	inv, err := g.GetInvoice("77")
	if err != nil || inv.SubscriptionId != "pre-1" || !inv.Paid || inv.Id != "77" {
		t.Errorf("invoice = %+v, %v", inv, err)
	}
}

func TestUpdateAndCancel(t *testing.T) {
	var bodies []string
	g := server(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, r.Method+" "+r.URL.Path+" "+string(raw))
		_, _ = w.Write([]byte(`{}`))
	})
	if err := g.UpdateAmount("pre-1", 45500); err != nil {
		t.Fatal(err)
	}
	if err := g.Cancel("pre-1"); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 || bodies[0] != `PUT /preapproval/pre-1 {"auto_recurring":{"transaction_amount":45500}}` || bodies[1] != `PUT /preapproval/pre-1 {"status":"cancelled"}` {
		t.Errorf("bodies = %q", bodies)
	}
}

func TestErrorsNeverEchoTheBody(t *testing.T) {
	g := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"payer a@b.co card 4111"}`))
	})
	_, err := g.GetSubscription("x")
	if err == nil || err.Error() != "Mercado Pago returned 400" {
		t.Errorf("err = %v", err)
	}
}
