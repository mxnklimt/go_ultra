package service

import (
	"testing"
	"time"

	"go_ultra/internal/domain"
)

// TestPlayerService_GetByID covers GetByID happy path and not-found error path.
func TestPlayerService_GetByID(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")

	got, err := psvc.GetByID(ctx, alice.ID)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if got.ID != alice.ID || got.Username != "alice" {
		t.Fatalf("GetByID returned wrong player: %+v", got)
	}

	_, err = psvc.GetByID(ctx, 999999)
	if err != domain.ErrPlayerNotFound {
		t.Fatalf("GetByID(missing) error = %v, want ErrPlayerNotFound", err)
	}
}

// TestPlayerService_SessionLifecycle covers CreatePlayerSession, GetSession (valid &
// unknown token) and DeletePlayerSession (logout makes the token invalid).
func TestPlayerService_SessionLifecycle(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")

	token, expiresAt, err := psvc.CreatePlayerSession(ctx, alice.ID)
	if err != nil {
		t.Fatalf("CreatePlayerSession error: %v", err)
	}
	if token == "" {
		t.Fatalf("empty token")
	}
	// PlayerSessionTTL 是 30 天，过期时间应远在未来。
	if !expiresAt.After(time.Now().UTC().Add(29 * 24 * time.Hour)) {
		t.Fatalf("expires_at too soon: %v", expiresAt)
	}

	pid, ok, err := psvc.GetSession(ctx, token)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if !ok {
		t.Fatalf("valid session should be ok")
	}
	if pid != alice.ID {
		t.Fatalf("GetSession player id = %d, want %d", pid, alice.ID)
	}

	// 未知 token：ok=false，无错误。
	_, ok, err = psvc.GetSession(ctx, "no-such-token")
	if err != nil {
		t.Fatalf("GetSession(unknown) error: %v", err)
	}
	if ok {
		t.Fatalf("unknown token should not be ok")
	}

	// 登出：删除后该 token 不再有效。
	if err := psvc.DeletePlayerSession(ctx, token); err != nil {
		t.Fatalf("DeletePlayerSession error: %v", err)
	}
	_, ok, err = psvc.GetSession(ctx, token)
	if err != nil {
		t.Fatalf("GetSession after delete error: %v", err)
	}
	if ok {
		t.Fatalf("deleted session should not be ok")
	}

	// 删除不存在的 token 仍幂等（无错误）。
	if err := psvc.DeletePlayerSession(ctx, "no-such-token"); err != nil {
		t.Fatalf("DeletePlayerSession(unknown) error: %v", err)
	}
}

// TestAdminService_DeleteAdminSession covers admin logout: after delete the token no
// longer checks ok; deleting an unknown token is idempotent.
func TestAdminService_DeleteAdminSession(t *testing.T) {
	sqlDB, q := newTestDB(t)
	asvc := NewAdminService(q, sqlDB)
	ctx := testCtx(t)

	token, _, err := asvc.CreateAdminSession(ctx)
	if err != nil {
		t.Fatalf("CreateAdminSession error: %v", err)
	}

	ok, _, err := asvc.CheckAdminSession(ctx, token)
	if err != nil || !ok {
		t.Fatalf("session should be valid before delete: ok=%v err=%v", ok, err)
	}

	if err := asvc.DeleteAdminSession(ctx, token); err != nil {
		t.Fatalf("DeleteAdminSession error: %v", err)
	}
	ok, _, err = asvc.CheckAdminSession(ctx, token)
	if err != nil {
		t.Fatalf("CheckAdminSession after delete error: %v", err)
	}
	if ok {
		t.Fatalf("deleted admin session should not be ok")
	}

	// 幂等：删除不存在的 token 无错误。
	if err := asvc.DeleteAdminSession(ctx, "no-such-token"); err != nil {
		t.Fatalf("DeleteAdminSession(unknown) error: %v", err)
	}
}

// TestMatchService_ListByPlayer_LossPerspective exercises the loss branch of
// ListByPlayer (opponent = winner) so both perspective branches are covered.
func TestMatchService_ListByPlayer_LossPerspective(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")

	// alice records a loss vs bob → bob is winner.
	at := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	if _, err := msvc.Record(ctx, alice.ID, "bob", "loss", at); err != nil {
		t.Fatalf("Record error: %v", err)
	}

	views, err := msvc.ListByPlayer(ctx, alice.ID, 50, 0)
	if err != nil {
		t.Fatalf("ListByPlayer error: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(views))
	}
	if views[0].Result != "loss" || views[0].Opponent != "bob" {
		t.Fatalf("loss-perspective view wrong: %+v", views[0])
	}
}
