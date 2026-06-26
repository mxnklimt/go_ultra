package handler

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestRecordMatch_Self_409(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	body := `{"opponent_username":"alice","result":"win"}`
	w := ts.do(http.MethodPost, "/api/matches", body, cookie)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	var b struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &b)
	if b.Error.Code != "SELF_MATCH" {
		t.Fatalf("code = %q, want SELF_MATCH", b.Error.Code)
	}
}

func TestRecordMatch_FuturePlayedAt_400(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")
	future := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	body := `{"opponent_username":"bob","result":"win","played_at":"` + future + `"}`
	w := ts.do(http.MethodPost, "/api/matches", body, cookie)
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

func TestRecordMatch_Success_201(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")
	body := `{"opponent_username":"bob","result":"win"}`
	w := ts.do(http.MethodPost, "/api/matches", body, cookie)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var res struct {
		ID                int64   `json:"id"`
		WinnerDelta       float64 `json:"winner_delta"`
		LoserDelta        float64 `json:"loser_delta"`
		NewSelfRating     float64 `json:"new_self_rating"`
		NewOpponentRating float64 `json:"new_opponent_rating"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if res.WinnerDelta+res.LoserDelta != 0 {
		t.Fatalf("delta not zero-sum: %v + %v", res.WinnerDelta, res.LoserDelta)
	}
	if res.NewSelfRating <= 1500.0 {
		t.Fatalf("winner new rating = %v, want > 1500.0", res.NewSelfRating)
	}
}

func TestListGlobalMatches(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")
	_ = ts.do(http.MethodPost, "/api/matches", `{"opponent_username":"bob","result":"win"}`, cookie)

	w := ts.do(http.MethodGet, "/api/matches", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var arr []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &arr); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if len(arr) != 1 {
		t.Fatalf("global matches len = %d, want 1; body=%s", len(arr), w.Body.String())
	}
}

func TestDeleteMatch_RequiresAdmin_403(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	w := ts.do(http.MethodDelete, "/api/matches/1", "", cookie)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
	var b struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &b)
	if b.Error.Code != "ADMIN_REQUIRED" {
		t.Fatalf("code = %q, want ADMIN_REQUIRED", b.Error.Code)
	}
}

func TestDeleteMatch_NoCookie_403(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodDelete, "/api/matches/1", "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestAdminDeleteAndRestoreFlow(t *testing.T) {
	ts := newTestServer(t)
	pw := ts.adminPlaintext
	if pw == "" {
		t.Fatalf("admin plaintext not generated")
	}
	playerCookie := ts.login("alice")
	_ = ts.login("bob")
	rec := ts.do(http.MethodPost, "/api/matches", `{"opponent_username":"bob","result":"win"}`, playerCookie)
	var created struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	adminCookie := ts.adminLogin(pw)

	// 删除
	del := ts.do(http.MethodDelete, "/api/matches/1", "", adminCookie)
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", del.Code, del.Body.String())
	}

	// 全局列表应为空
	list := ts.do(http.MethodGet, "/api/matches", "", playerCookie)
	var arr []map[string]any
	_ = json.Unmarshal(list.Body.Bytes(), &arr)
	if len(arr) != 0 {
		t.Fatalf("global matches after delete = %d, want 0", len(arr))
	}

	// 已删除列表应有 1 条
	deleted := ts.do(http.MethodGet, "/api/admin/matches/deleted", "", adminCookie)
	if deleted.Code != http.StatusOK {
		t.Fatalf("deleted list status = %d, want 200; body=%s", deleted.Code, deleted.Body.String())
	}

	// 恢复
	restore := ts.do(http.MethodPost, "/api/admin/matches/1/restore", "", adminCookie)
	if restore.Code != http.StatusNoContent {
		t.Fatalf("restore status = %d, want 204; body=%s", restore.Code, restore.Body.String())
	}
}
