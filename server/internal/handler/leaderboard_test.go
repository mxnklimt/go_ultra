package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestLeaderboard_RequiresAuth(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodGet, "/api/leaderboard", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", w.Code, w.Body.String())
	}
}

func TestLeaderboard_ReturnsRanks(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")
	_ = ts.do(http.MethodPost, "/api/matches", `{"opponent_username":"bob","result":"win"}`, cookie)

	w := ts.do(http.MethodGet, "/api/leaderboard?min_games=0", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var rows []struct {
		Rank     int    `json:"rank"`
		Username string `json:"username"`
		Rating   int    `json:"rating"`
		Dan      int    `json:"dan"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if len(rows) < 1 {
		t.Fatalf("rows len = %d, want >= 1", len(rows))
	}
	if rows[0].Rank != 1 {
		t.Fatalf("first row rank = %d, want 1", rows[0].Rank)
	}
}

func TestCompare_TooMany_400(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	names := make([]string, 11)
	for i := range names {
		names[i] = "u" + string(rune('a'+i))
	}
	w := ts.do(http.MethodGet, "/api/compare?usernames="+strings.Join(names, ","), "", cookie)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	var b struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &b)
	if b.Error.Code != "INVALID_PARAM" {
		t.Fatalf("code = %q, want INVALID_PARAM", b.Error.Code)
	}
}

func TestCompare_Empty_400(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	w := ts.do(http.MethodGet, "/api/compare", "", cookie)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestCompare_Valid(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")
	w := ts.do(http.MethodGet, "/api/compare?usernames=alice,bob", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Series []struct {
			Username string `json:"username"`
		} `json:"series"`
		HeadToHead []map[string]any `json:"head_to_head"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if len(body.Series) != 2 {
		t.Fatalf("series len = %d, want 2", len(body.Series))
	}
}
