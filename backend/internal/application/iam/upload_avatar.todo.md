# UploadAvatar — deferred

This use case is intentionally **not implemented in v1**. The HTTP route for
`POST /api/me/avatar` returns `501 Not Implemented` per
`docs/backend/use-cases.md` → `UploadAvatar` and the spec assumption block.

When v2 picks this up, the work plan is:

1. Add a `UploadAvatar` struct in `application/iam/upload_avatar.go` with the
   shape `Handle(ctx, UploadAvatarInput) (UploadAvatarOutput, error)`.
2. Add an `AvatarStore` port (Put/Get) and an infra adapter under
   `internal/infrastructure/storage/`.
3. Validate content type (PNG/JPEG) and size cap (≤2MB) at the HTTP edge.
4. Mutate `iam.User.SetAvatar(...)` and persist via `iam.Users.Save`.
5. Register the route in `internal/interfaces/http/router.go` and remove the
   501 stub.
6. Add Idempotency-Key support per ADR 0005 (the route is already on the list
   of tagged endpoints).

Layer 1 deliberately leaves no Go code for this use case so the v2 owner can
build it test-first against the same DDD contract.
