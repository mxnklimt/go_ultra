package handler

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestListPlayers_RequiresAuth(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodGet, "/api/players", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", w.Code, w.Body.String())
	}
}

func TestListPlayers_ReturnsArray(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")

	w := ts.do(http.MethodGet, "/api/players", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var arr []struct {
		Username string  `json:"username"`
		Rating   float64 `json:"rating"`
		Dan      int     `json:"dan"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &arr); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if len(arr) != 2 {
		t.Fatalf("len = %d, want 2; body=%s", len(arr), w.Body.String())
	}
}

func TestGetPlayer_ByUsername(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")

	w := ts.do(http.MethodGet, "/api/players/alice", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Username string `json:"username"`
		Stats    struct {
			Wins   int `json:"wins"`
			Losses int `json:"losses"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if body.Username != "alice" {
		t.Fatalf("username = %q, want alice", body.Username)
	}
}

func TestGetPlayer_NotFound_404(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	w := ts.do(http.MethodGet, "/api/players/ghost", "", cookie)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Error.Code != "PLAYER_NOT_FOUND" {
		t.Fatalf("code = %q, want PLAYER_NOT_FOUND", body.Error.Code)
	}
}

func TestPlayerHistory_HasStartingPoint(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	w := ts.do(http.MethodGet, "/api/players/alice/history", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var pts []struct {
		PlayedAt string  `json:"played_at"`
		Rating   float64 `json:"rating"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &pts); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	// 没有对局时也应至少含 1 个起点（created_at, 1500）。
	if len(pts) < 1 {
		t.Fatalf("history len = %d, want >= 1", len(pts))
	}
	if pts[0].Rating != 1500.0 {
		t.Fatalf("first point rating = %v, want 1500.0", pts[0].Rating)
	}
}

func TestPlayerMatches_Empty(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	w := ts.do(http.MethodGet, "/api/players/alice/matches", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var arr []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &arr); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if len(arr) != 0 {
		t.Fatalf("matches len = %d, want 0", len(arr))
	}
}
