package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorString(t *testing.T) {
	e := &Error{Code: "PLAYER_NOT_FOUND", Message: "玩家不存在", Status: 404}
	got := e.Error()
	if !strings.Contains(got, "PLAYER_NOT_FOUND") || !strings.Contains(got, "玩家不存在") {
		t.Fatalf("Error() = %q, want it to contain code and message", got)
	}
}

func TestWithCauseReturnsCopy(t *testing.T) {
	cause := errors.New("disk exploded")
	withCause := ErrInternal.WithCause(cause)

	if withCause.Cause != cause {
		t.Fatalf("WithCause did not attach cause; got %v", withCause.Cause)
	}
	if ErrInternal.Cause != nil {
		t.Fatalf("WithCause mutated the original sentinel; ErrInternal.Cause = %v", ErrInternal.Cause)
	}
	if withCause.Code != ErrInternal.Code || withCause.Message != ErrInternal.Message || withCause.Status != ErrInternal.Status {
		t.Fatalf("WithCause changed code/message/status: %+v", withCause)
	}
	if withCause == ErrInternal {
		t.Fatalf("WithCause returned the same pointer, expected a copy")
	}
}

func TestWithCauseErrorStringStillWorks(t *testing.T) {
	e := ErrPlayerNotFound.WithCause(errors.New("row not found"))
	if !strings.Contains(e.Error(), "PLAYER_NOT_FOUND") {
		t.Fatalf("Error() after WithCause = %q", e.Error())
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		err    *Error
		code   string
		status int
	}{
		{ErrPlayerNotFound, "PLAYER_NOT_FOUND", 404},
		{ErrMatchNotFound, "MATCH_NOT_FOUND", 404},
		{ErrSelfMatch, "SELF_MATCH", 409},
		{ErrNotAuthenticated, "NOT_AUTHENTICATED", 401},
		{ErrAdminRequired, "ADMIN_REQUIRED", 403},
		{ErrInvalidBody, "INVALID_BODY", 400},
		{ErrInvalidParam, "INVALID_PARAM", 400},
		{ErrInternal, "INTERNAL", 500},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			if tt.err == nil {
				t.Fatalf("predefined error %s is nil", tt.code)
			}
			if tt.err.Code != tt.code {
				t.Fatalf("Code = %q, want %q", tt.err.Code, tt.code)
			}
			if tt.err.Status != tt.status {
				t.Fatalf("Status = %d, want %d (code %s)", tt.err.Status, tt.status, tt.code)
			}
			if tt.err.Message == "" {
				t.Fatalf("Message empty for code %s", tt.code)
			}
		})
	}
}
