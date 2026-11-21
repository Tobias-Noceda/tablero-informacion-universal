//go:build integration

package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
)

// CONC-05: everyone opening a board at once must not multiply the calls to
// a card's API. Requests that miss the cache while the card is running wait
// for that run.
func TestEndToEnd_ConcurrentReadsOfACardAskItsAPIOnce(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	var calls atomic.Int32
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		// Slow enough that every reader arrives while the first run is under way.
		time.Sleep(300 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"compra":1400,"venta":1450}`)
	}))
	defer stub.Close()

	s := newStack(t)
	owner := s.signUp("owner")
	board := owner.newBoard("conc-05")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)

	var postit models.PostIts
	s.mustJSON(owner.do(http.MethodPost, "/api/v1/post-its",
		map[string]any{"board": board.Id, "well_known": "dolar_oficial"}), http.StatusCreated, &postit)
	resource, _ := url.Parse(stub.URL)
	if err := s.db.UpdatePostIt(postit.Id, map[string]any{"resource": resource}); err != nil {
		t.Fatalf("retarget: %v", err)
	}
	if err := s.cache.DropPostItResult(postit.Id); err != nil {
		t.Fatalf("drop cache: %v", err)
	}

	const readers = 25
	codes := make([]int, readers)
	var done sync.WaitGroup
	for i := range readers {
		done.Add(1)
		go func() {
			defer done.Done()
			codes[i] = owner.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil).Code
		}()
	}
	done.Wait()

	for i, code := range codes {
		if code != http.StatusOK {
			t.Errorf("reader %d got %d", i, code)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("%d concurrent readers made %d calls to the card's API, want 1", readers, n)
	}
}
