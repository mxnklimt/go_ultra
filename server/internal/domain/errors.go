package domain

// Error 是统一的领域错误类型：service 层只抛它，handler 层据此映射 HTTP 响应。
type Error struct {
	Code    string
	Message string
	Status  int
	Cause   error
}

// Error 实现 error 接口，输出便于日志排查的字符串。
func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Code + ": " + e.Message + ": " + e.Cause.Error()
	}
	return e.Code + ": " + e.Message
}

// WithCause 返回一个带上底层原因的副本，不修改原始（预定义 sentinel）值。
func (e *Error) WithCause(err error) *Error {
	cp := *e
	cp.Cause = err
	return &cp
}

var (
	ErrPlayerNotFound   = &Error{Code: "PLAYER_NOT_FOUND", Message: "玩家不存在", Status: 404}
	ErrMatchNotFound    = &Error{Code: "MATCH_NOT_FOUND", Message: "对局不存在", Status: 404}
	ErrSelfMatch        = &Error{Code: "SELF_MATCH", Message: "不能和自己对局", Status: 409}
	ErrNotAuthenticated = &Error{Code: "NOT_AUTHENTICATED", Message: "未登录", Status: 401}
	ErrAdminRequired    = &Error{Code: "ADMIN_REQUIRED", Message: "需要管理员权限", Status: 403}
	ErrInvalidBody      = &Error{Code: "INVALID_BODY", Message: "请求体无效", Status: 400}
	ErrInvalidParam     = &Error{Code: "INVALID_PARAM", Message: "参数无效", Status: 400}
	ErrInternal         = &Error{Code: "INTERNAL", Message: "服务器内部错误", Status: 500}
	ErrRateLimited      = &Error{Code: "RATE_LIMITED", Message: "尝试过于频繁，请稍后", Status: 429}
)
