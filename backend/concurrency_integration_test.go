//go:build integration

package main

import (
	"encoding/json"
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
	"github.com/google/uuid"
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

// inParallel sends n requests at once and counts the answers by status.
func inParallel(n int, send func(i int) int) map[int]int {
	var mu sync.Mutex
	codes := map[int]int{}
	var done sync.WaitGroup
	start := make(chan struct{})
	for i := range n {
		done.Add(1)
		go func() {
			defer done.Done()
			<-start
			code := send(i)
			mu.Lock()
			codes[code]++
			mu.Unlock()
		}()
	}
	close(start)
	done.Wait()
	return codes
}

// CONC-04: a free account asking for many boards at once ends up with no
// more than its plan allows; the rest are refused with 402.
func TestEndToEnd_ParallelCreationsStayWithinThePlan(t *testing.T) {
	s := newStack(t)
	ana := s.register("conc04-" + uuid.NewString()[:8] + "@it.test")
	max := models.LimitsFor(models.PlanFree).BoardsPerSpace

	codes := inParallel(4*max, func(i int) int {
		w := ana.do(http.MethodPost, "/api/v1/boards", map[string]string{"name": fmt.Sprintf("b%d", i)})
		if w.Code == http.StatusCreated {
			var board models.Board
			_ = json.Unmarshal(w.Body.Bytes(), &board)
			t.Cleanup(func() { ana.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil) })
		}
		return w.Code
	})

	if codes[http.StatusCreated] != max || codes[http.StatusPaymentRequired] != 3*max {
		t.Errorf("answers = %v, want %d created and %d refused", codes, max, 3*max)
	}
	count, err := s.db.CountPersonalBoards(ana.id)
	if err != nil {
		t.Fatal(err)
	}
	if int(count) != max {
		t.Errorf("%d boards stored, the plan allows %d", count, max)
	}
}

// CONC-04 for the cards of one board, counted on the board document.
func TestEndToEnd_ParallelCardsStayWithinThePlan(t *testing.T) {
	s := newStack(t)
	ana := s.register("conc04-cards-" + uuid.NewString()[:8] + "@it.test")
	board := ana.newBoard("cards")
	defer ana.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	max := models.LimitsFor(models.PlanFree).NodesPerBoard

	codes := inParallel(max+15, func(i int) int {
		return ana.do(http.MethodPost, "/api/v1/post-its", map[string]any{
			"board": board.Id, "well_known": "static_card", "params": map[string]string{"text": fmt.Sprint(i)},
		}).Code
	})

	stored, err := s.db.FindBoard(board.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.PostIts) != max || codes[http.StatusCreated] != max {
		t.Errorf("%d cards on the board, %v answers; the plan allows %d", len(stored.PostIts), codes, max)
	}
}
