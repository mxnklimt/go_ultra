package service

import (
	"testing"
	"time"
)

func TestAdminService_EnsurePassword_GenerateThenIdempotent(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewAdminService(q, sqlDB)
	ctx := testCtx(t)

	pw, generated, err := svc.EnsurePassword(ctx)
	if err != nil {
		t.Fatalf("EnsurePassword error: %v", err)
	}
	if !generated {
		t.Fatalf("first EnsurePassword should report generated=true")
	}
	if len(pw) != 16 {
		t.Fatalf("generated password length = %d, want 16", len(pw))
	}
	// 生成的明文必须能被 VerifyPassword 接受
	ok, err := svc.VerifyPassword(ctx, pw)
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if !ok {
		t.Fatalf("generated password failed verification")
	}

	// 二次调用：generated=false，明文为空
	pw2, generated2, err := svc.EnsurePassword(ctx)
	if err != nil {
		t.Fatalf("second EnsurePassword error: %v", err)
	}
	if generated2 {
		t.Fatalf("second EnsurePassword should report generated=false")
	}
	if pw2 != "" {
		t.Fatalf("second EnsurePassword should return empty plaintext, got %q", pw2)
	}
	// 原密码仍然有效
	ok, _ = svc.VerifyPassword(ctx, pw)
	if !ok {
		t.Fatalf("original password should still verify after second EnsurePassword")
	}
}

func TestAdminService_VerifyPassword_WrongPassword(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewAdminService(q, sqlDB)
	ctx := testCtx(t)

	if _, _, err := svc.EnsurePassword(ctx); err != nil {
		t.Fatalf("EnsurePassword error: %v", err)
	}
	ok, err := svc.VerifyPassword(ctx, "definitely-wrong")
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if ok {
		t.Fatalf("wrong password should not verify")
	}
}

func TestAdminService_Session_CreateAndCheck(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewAdminService(q, sqlDB)
	ctx := testCtx(t)

	token, expiresAt, err := svc.CreateAdminSession(ctx)
	if err != nil {
		t.Fatalf("CreateAdminSession error: %v", err)
	}
	if token == "" {
		t.Fatalf("empty token")
	}
	// 过期时间应在约 30 分钟后（容差 1 分钟）
	wantMin := time.Now().UTC().Add(29 * time.Minute)
	wantMax := time.Now().UTC().Add(31 * time.Minute)
	if expiresAt.Before(wantMin) || expiresAt.After(wantMax) {
		t.Fatalf("expires_at out of range: %v", expiresAt)
	}

	ok, exp, err := svc.CheckAdminSession(ctx, token)
	if err != nil {
		t.Fatalf("CheckAdminSession error: %v", err)
	}
	if !ok {
		t.Fatalf("valid session should check ok")
	}
	if !exp.Equal(expiresAt) {
		t.Fatalf("checked expires_at = %v, want %v", exp, expiresAt)
	}

	// 不存在的 token
	ok, _, err = svc.CheckAdminSession(ctx, "no-such-token")
	if err != nil {
		t.Fatalf("CheckAdminSession(unknown) error: %v", err)
	}
	if ok {
		t.Fatalf("unknown token should not check ok")
	}
}

func TestAdminService_SoftDelete_HidesFromQueries_RestoreBringsBack(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	asvc := NewAdminService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	bob, _ := psvc.LoginOrCreate(ctx, "bob")

	at := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	res, err := msvc.Record(ctx, alice.ID, "bob", "win", at)
	if err != nil {
		t.Fatalf("Record error: %v", err)
	}

	// 删除前：全局/各玩家视角都能看到
	if g, _ := msvc.ListGlobal(ctx, 50, 0); len(g) != 1 {
		t.Fatalf("before delete: ListGlobal expected 1, got %d", len(g))
	}
	if pv, _ := msvc.ListByPlayer(ctx, alice.ID, 50, 0); len(pv) != 1 {
		t.Fatalf("before delete: alice ListByPlayer expected 1, got %d", len(pv))
	}
	if dv, _ := asvc.ListDeleted(ctx); len(dv) != 0 {
		t.Fatalf("before delete: ListDeleted expected 0, got %d", len(dv))
	}

	// 软删除
	if err := asvc.SoftDelete(ctx, res.MatchID); err != nil {
		t.Fatalf("SoftDelete error: %v", err)
	}

	// 删除后：普通查询不返回，已删除列表返回
	if g, _ := msvc.ListGlobal(ctx, 50, 0); len(g) != 0 {
		t.Fatalf("after delete: ListGlobal expected 0, got %d", len(g))
	}
	if pv, _ := msvc.ListByPlayer(ctx, alice.ID, 50, 0); len(pv) != 0 {
		t.Fatalf("after delete: alice ListByPlayer expected 0, got %d", len(pv))
	}
	if pv, _ := msvc.ListByPlayer(ctx, bob.ID, 50, 0); len(pv) != 0 {
		t.Fatalf("after delete: bob ListByPlayer expected 0, got %d", len(pv))
	}
	deleted, err := asvc.ListDeleted(ctx)
	if err != nil {
		t.Fatalf("ListDeleted error: %v", err)
	}
	if len(deleted) != 1 {
		t.Fatalf("after delete: ListDeleted expected 1, got %d", len(deleted))
	}
	if deleted[0].ID != res.MatchID {
		t.Fatalf("deleted match id = %d, want %d", deleted[0].ID, res.MatchID)
	}
	// deleted_by 应为 NULL（管理员非 player）
	if deleted[0].DeletedBy != nil {
		t.Fatalf("deleted_by should be nil, got %v", *deleted[0].DeletedBy)
	}

	// 恢复
	if err := asvc.Restore(ctx, res.MatchID); err != nil {
		t.Fatalf("Restore error: %v", err)
	}
	if g, _ := msvc.ListGlobal(ctx, 50, 0); len(g) != 1 {
		t.Fatalf("after restore: ListGlobal expected 1, got %d", len(g))
	}
	if dv, _ := asvc.ListDeleted(ctx); len(dv) != 0 {
		t.Fatalf("after restore: ListDeleted expected 0, got %d", len(dv))
	}
}
