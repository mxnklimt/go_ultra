package handler

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestLogin_ImplicitRegister_SetsCookie(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodPost, "/api/login", `{"username":"alice"}`)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 200/201; body=%s", w.Code, w.Body.String())
	}
	var hasCookie bool
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "go_ultra_session" && ck.Value != "" {
			hasCookie = true
			if !ck.HttpOnly {
				t.Fatalf("session cookie not HttpOnly")
			}
		}
	}
	if !hasCookie {
		t.Fatalf("login did not set go_ultra_session cookie")
	}

	var body struct {
		Player struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Rating   int    `json:"rating"`
		} `json:"player"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if body.Player.Username != "alice" {
		t.Fatalf("username = %q, want alice", body.Player.Username)
	}
	if body.Player.Rating != 1500 {
		t.Fatalf("rating = %d, want 1500", body.Player.Rating)
	}
}

func TestLogin_InvalidBody_400(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodPost, "/api/login", `{"username":""}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestMe_RequiresAuth(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodGet, "/api/me", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Error.Code != "NOT_AUTHENTICATED" {
		t.Fatalf("code = %q, want NOT_AUTHENTICATED", body.Error.Code)
	}
}

func TestMe_WithCookie_ReturnsPlayer(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("bob")
	w := ts.do(http.MethodGet, "/api/me", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Player struct {
			Username string `json:"username"`
		} `json:"player"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if body.Player.Username != "bob" {
		t.Fatalf("username = %q, want bob", body.Player.Username)
	}
}

func TestLogout_204(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("carol")
	w := ts.do(http.MethodPost, "/api/logout", "", cookie)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", w.Code, w.Body.String())
	}
}

func TestAdminStatus_Unauthed(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodGet, "/api/admin/status", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Authed bool `json:"authed"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if body.Authed {
		t.Fatalf("authed = true, want false")
	}
}

func TestAdminLogin_WrongPassword_400(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodPost, "/api/admin/login", `{"password":"definitely-wrong"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}
