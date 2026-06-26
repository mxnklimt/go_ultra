package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go_ultra/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type errBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestRespondError_DomainError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ctxLogger, zerolog.Nop())

	respondError(c, domain.ErrPlayerNotFound)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	var body errBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body %q: %v", w.Body.String(), err)
	}
	if body.Error.Code != "PLAYER_NOT_FOUND" {
		t.Fatalf("code = %q, want PLAYER_NOT_FOUND", body.Error.Code)
	}
	if body.Error.Message != "玩家不存在" {
		t.Fatalf("message = %q, want 玩家不存在", body.Error.Message)
	}
}

func TestRespondError_DomainErrorWithCause(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ctxLogger, zerolog.Nop())

	wrapped := domain.ErrInvalidParam.WithCause(errors.New("bad parse"))
	respondError(c, wrapped)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var body errBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body %q: %v", w.Body.String(), err)
	}
	if body.Error.Code != "INVALID_PARAM" {
		t.Fatalf("code = %q, want INVALID_PARAM", body.Error.Code)
	}
}

func TestRespondError_NonDomainError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ctxLogger, zerolog.Nop())

	respondError(c, errors.New("some random failure"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	var body errBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body %q: %v", w.Body.String(), err)
	}
	if body.Error.Code != "INTERNAL" {
		t.Fatalf("code = %q, want INTERNAL", body.Error.Code)
	}
	if body.Error.Message != "服务器内部错误" {
		t.Fatalf("message = %q, want 服务器内部错误", body.Error.Message)
	}
}

func TestRespondError_NilError(t *testing.T) {
	// nil 也按 500 兜底处理，避免空指针。
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ctxLogger, zerolog.Nop())

	respondError(c, nil)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}
