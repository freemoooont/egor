// Package iam wires the iam application use cases to HTTP. Each handler
// translates a DTO → use-case input, calls the use case, and writes a DTO
// back. Errors propagate through the central errorMapper in middleware/errors.go.
package iam

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	appiam "github.com/micocards/api/internal/application/iam"
	"github.com/micocards/api/internal/domain/iam"
	dtoauth "github.com/micocards/api/internal/interfaces/http/dto/auth"
	"github.com/micocards/api/internal/interfaces/http/middleware"
)

// Deps groups the use cases this package needs.
type Deps struct {
	Register       appiam.RegisterUser
	Login          appiam.LoginUser
	Refresh        appiam.RefreshAccessToken
	Logout         appiam.LogoutUser
	GetMe          appiam.GetCurrentUser
	UpdateProfile  appiam.UpdateProfile
	ChangePassword appiam.ChangePassword
	DBPool         *pgxpool.Pool // health endpoint pings it; may be nil
	StartTime      time.Time
}

// Handlers holds the wired Deps.
type Handlers struct{ d Deps }

// New builds the handler set.
func New(d Deps) *Handlers {
	if d.StartTime.IsZero() {
		d.StartTime = time.Now().UTC()
	}
	return &Handlers{d: d}
}

// Healthz answers GET /api/healthz with `{"status":"ok","db":"reachable","time":...}`.
// When the DB pool is nil or unreachable, db is reported as "unreachable" but the
// endpoint still returns 200 — liveness vs readiness.
func (h *Handlers) Healthz(w http.ResponseWriter, r *http.Request) {
	dbStatus := "skipped"
	if h.d.DBPool != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := h.d.DBPool.Ping(ctx); err != nil {
			dbStatus = "unreachable"
		} else {
			dbStatus = "reachable"
		}
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"db":     dbStatus,
		"time":   time.Now().UTC().Format(time.RFC3339Nano),
		"uptime": time.Since(h.d.StartTime).String(),
	})
}

// Register handles POST /api/auth/register.
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var body dtoauth.RegisterRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	if err := validateRegister(body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	out, err := h.d.Register.Handle(r.Context(), appiam.RegisterUserInput{
		Email: body.Email, Password: body.Password, DisplayName: body.DisplayName,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, toAuthResponse(out.Auth, out.User))
}

// Login handles POST /api/auth/login.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var body dtoauth.LoginRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(body.Email) == "" || body.Password == "" {
		middleware.WriteError(w, r, middleware.NewValidationError("validation_failed",
			"email and password are required",
			map[string]string{"email": "required", "password": "required"}))
		return
	}
	out, err := h.d.Login.Handle(r.Context(), appiam.LoginUserInput{
		Email: body.Email, Password: body.Password,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, toAuthResponse(out.Auth, out.User))
}

// Refresh handles POST /api/auth/refresh.
func (h *Handlers) Refresh(w http.ResponseWriter, r *http.Request) {
	var body dtoauth.RefreshRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(body.RefreshToken) == "" {
		middleware.WriteError(w, r, iam.ErrRefreshTokenInvalid)
		return
	}
	out, err := h.d.Refresh.Handle(r.Context(), appiam.RefreshAccessTokenInput{
		RefreshToken: body.RefreshToken,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, dtoauth.RefreshResponse{
		AccessToken:           out.Auth.AccessToken,
		RefreshToken:          out.Auth.RefreshToken,
		AccessTokenExpiresAt:  out.Auth.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: out.Auth.RefreshTokenExpiresAt,
	})
}

// Logout handles POST /api/auth/logout.
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	var body dtoauth.RefreshRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	out, err := h.d.Logout.Handle(r.Context(), appiam.LogoutUserInput{RefreshToken: body.RefreshToken})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]bool{"ok": out.OK})
}

// GetMe handles GET /api/me.
func (h *Handlers) GetMe(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	out, err := h.d.GetMe.Handle(r.Context(), appiam.GetCurrentUserInput{UserID: uid})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, toUser(out.User))
}

// UpdateMe handles PUT /api/me (we accept PUT here even though OpenAPI says
// PATCH — the spec listed PUT in the deliverable; we register both verbs).
func (h *Handlers) UpdateMe(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	var body dtoauth.UpdateProfileRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	in := appiam.UpdateProfileInput{UserID: uid}
	if body.Email != nil {
		v := *body.Email
		in.Email = &v
	}
	if body.DisplayName != nil {
		v := *body.DisplayName
		in.DisplayName = &v
	}
	out, err := h.d.UpdateProfile.Handle(r.Context(), in)
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, toUser(out.User))
}

// ChangePassword handles POST /api/auth/change-password.
func (h *Handlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	var body dtoauth.ChangePasswordRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	if body.CurrentPassword == "" || body.NewPassword == "" {
		middleware.WriteError(w, r, middleware.NewValidationError(
			"validation_failed", "currentPassword and newPassword are required",
			map[string]string{"currentPassword": "required", "newPassword": "required"}))
		return
	}
	out, err := h.d.ChangePassword.Handle(r.Context(), appiam.ChangePasswordInput{
		UserID: uid, CurrentPassword: body.CurrentPassword, NewPassword: body.NewPassword,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]bool{"ok": out.OK})
}

// Avatar is a 501 stub — see spec assumption "avatar upload deferred".
func (h *Handlers) Avatar(w http.ResponseWriter, r *http.Request) {
	middleware.WriteJSON(w, http.StatusNotImplemented, map[string]string{
		"error":   "not_implemented",
		"message": "avatar upload deferred",
	})
}

func validateRegister(body dtoauth.RegisterRequest) error {
	missing := map[string]string{}
	if strings.TrimSpace(body.Email) == "" {
		missing["email"] = "required"
	}
	if body.Password == "" {
		missing["password"] = "required"
	}
	if strings.TrimSpace(body.DisplayName) == "" {
		missing["displayName"] = "required"
	}
	if len(missing) > 0 {
		return middleware.NewValidationError("validation_failed",
			"missing required field(s)", missing)
	}
	return nil
}

func toUser(u appiam.UserView) dtoauth.User {
	ref := u.AvatarRef
	if u.AvatarKind == iam.AvatarRefNone || ref == "" {
		ref = "none"
	}
	return dtoauth.User{
		ID: u.ID, Email: u.Email, DisplayName: u.DisplayName,
		AvatarRef: ref, RegisteredAt: u.RegisteredAt,
	}
}

func toAuthResponse(b appiam.AuthBundle, u appiam.UserView) dtoauth.AuthResponse {
	return dtoauth.AuthResponse{
		AccessToken:           b.AccessToken,
		RefreshToken:          b.RefreshToken,
		AccessTokenExpiresAt:  b.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: b.RefreshTokenExpiresAt,
		User:                  toUser(u),
	}
}

// Marshal makes the package importable in tests where we want to encode the
// canonical wire shape directly. Useful in handler-level golden tests.
var _ = json.Marshal
