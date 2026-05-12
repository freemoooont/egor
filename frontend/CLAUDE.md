# Claude guide — Micocards frontend

Read the root `/CLAUDE.md` first for cross-cutting rules (orchestrator-only, Playwright + Figma verification, DDD-first backend). This file is the **authoritative** per-area guide for everything under `frontend/`.

## Toolchain

- **Vite+** (`vite-plus` package) is the toolchain. Use `pnpm` to invoke its `vp` CLI through package.json scripts:
  - `pnpm install` — install deps (Vite+ accepts pnpm; the spec rule against direct package managers is for `vp add` / `vp remove`; raw `pnpm install` is fine).
  - `pnpm dev` → `vp dev` — start dev server on `:5173` with proxy to the Go API on `:8080` (override via `VITE_API_TARGET`).
  - `pnpm build` → `tsc -b && vp build` — typecheck then build to `dist/`.
  - `pnpm check` → `vp check` — fmt + lint + typecheck. **Run before declaring a task done.**
  - `pnpm test` → `vp test` — Vitest via Vite+.
  - `pnpm lint` → `vp lint` — Oxlint with type-aware rules.
  - `pnpm typecheck` → `tsc --noEmit` — fast standalone TS check.
- **Adding/removing deps**: run `pnpm dlx shadcn@latest add <name>` for shadcn components; for everything else mirror the lockfile-managed approach used by `kot` (edit `package.json` and run `pnpm install`). The reference repo prefers `vp add`; this repo currently uses raw `pnpm` because the contributors run on stock Node + pnpm.
- **Never start `pnpm dev` from inside an agent session** — it doesn't exit. The verifier owns that; CI/agents only run `pnpm install`, `pnpm typecheck`, and `pnpm build`.
- **MSW is gated by `VITE_USE_MOCKS=true`. Default dev points at the Go backend on :8080 via the Vite proxy.**

## Architecture: Feature-Sliced Design v2.1

Path alias: `@/` → `src/` (configured in `vite.config.ts` and `tsconfig.app.json`).

```
src/
├── app/        # shell, providers, router root, global styles, route registry
├── pages/      # one slice per route; index.ts re-exports the *Route
├── widgets/    # composite UI blocks reused on 2+ pages (e.g. AppShell)
├── features/   # reusable user interactions (login, create-deck, rate-card)
├── entities/   # reusable business domain models (deck, card, user)
└── shared/     # infra: ui (shadcn), api, router root, config, lib, auth
```

**Layer rules (MUST):**

- **Import direction is top-down only**: `app → pages → widgets → features → entities → shared`. Upward imports are forbidden. So is cross-importing between slices on the same layer.
- **Public API**: every slice exposes its surface via `index.ts`. Consumers import from `@/pages/decks-list` or `@/widgets/app-shell` — never deep-import a slice's `ui/...`.
- **Domain-named files**, not technical roles. `model/deck.ts`, `api/fetch-deck.ts`. Never `types.ts`, `utils.ts`, `helpers.ts`.
- **No business logic in `shared/`** — it's infrastructure: UI kit, utils, API client, route root, auth tokens, env config.
- **Pages-first**: when a feature/entity has only one consumer, keep it inside that page slice. Extract to `features/` or `entities/` when 2+ consumers appear (Steiger flags single-use slices as `insignificant-slice`).

**Cross-import resolution order**: merge slices → extract to entities → compose in a higher layer (IoC) → `@x` notation (entities only, last resort).

Validate with `pnpm dlx @feature-sliced/steiger src` when in doubt.

## Reatom v1000 — use the most advanced applicable pattern

**Read `frontend/llms/reatom.md` end-to-end every session.** The v1000 API evolves; "obvious" idioms are often anti-patterns. Pick the heaviest applicable abstraction, not the toy example.

- **Async reads** → `computed(async () => ...).extend(withAsyncData())`. Never `useEffect` + `fetch`. Never imperative loaders.
- **Mutations** → `action(async () => ...).extend(withAsync())`. Use `onFulfill`/`onReject` for toasts and chained state.
- **Forms** → `reatomForm` + `reatomField` with Zod schemas. Wire validation, async submit, error mapping, dirty/touched through Reatom — not `useState`.
- **Routes** → `reatomRoute` with typed Zod params. Loaders, nested routes for master-detail. Use `self.outlet()` correctly.
- **Components** → `reatomComponent(...)` for every component to get auto-tracked re-renders. Never `React.memo + useState`.
- **Atoms / actions / computed are module-scope**, never created inside components. Always **named**: `atom(0, 'feature.counter')` — unnamed atoms are invisible in devtools.
- **Atomization**: lift mutable fields into atoms inside an immutable record (see `llms/reatom.md` § Atomization). Never normalize backend data into a separate "selected" / "loading" sidecar list.
- **Identity actions are forbidden** (an action that just forwards into `atom.set` — call `atom.set` directly).
- **Memoize expensive computeds**, split big atoms into focused ones — fine-grained reactivity is the whole point.

### CRITICAL: the `wrap` rule

`setup.ts` calls `clearStack()`, so **every async boundary that touches atoms or actions must be wrapped** or it fails silently with `ReatomError: missing async stack`.

Wrap on:

- DOM handlers: `onClick={wrap(() => atom.set(...))}`.
- `.then()` callbacks, `addEventListener`, `setTimeout`.
- Curried handlers: wrap the **inner** function. ✅ `(id) => wrap(() => doStuff(id))`. ❌ `wrap((id) => () => doStuff(id))`.
- Promise chains: don't chain after `wrap`; wrap each step.

Examples:

```ts
// ✅ Good
const submit = action(async (payload: LoginDto) => {
  const response = await wrap(apiFetch('/auth/login', { method: 'POST', body: JSON.stringify(payload) }));
  // ...
}, 'auth.submit').extend(withAsync());

<Button onClick={wrap(() => submit({ email, password }))}>Войти</Button>
```

```ts
// ❌ Bad — chained `.then` outside `wrap`, atom write fails silently
fetch('/api/me').then((res) => userAtom.set(res.json()));
```

## Tailwind v4 + shadcn/ui

- **Tailwind v4** is configured via `@tailwindcss/vite`. Tokens live in `src/app/styles/global.css` (`@theme { --color-brand-500: ... }`) and are mirrored in `tailwind.config.ts` for IDE intellisense.
- **Brand orange** is `#F97316` (Tailwind `orange-500`) — the canonical shade until the screen pass tightens it from Figma. Reference it as `bg-brand-500` / `text-brand-500` / `ring-brand-500`. **Never put hex literals in TSX.**
- **shadcn/ui** components are owned source files in `src/shared/ui/`. Generate them via `pnpm dlx shadcn@latest add <name>` (button, input, label, tabs, card, switch, avatar, dialog, dropdown-menu, sonner, separator, skeleton, scroll-area, progress are the MVP set). Edit them as needed — they're our code.
- **Icons** come from `lucide-react`. Don't pull other icon packs.

## Visual verification — non-negotiable

Every screen implementation MUST be verified with:

1. `mcp__plugin_playwright_playwright__browser_resize` to **1440x900** (desktop) and **390x844** (mobile, where the design has both).
2. `mcp__plugin_playwright_playwright__browser_navigate` to the route, then `browser_take_screenshot` saved to `.agent/tasks/micocards-mvp/raw/screenshots/<screen>-<viewport>.png`.
3. `mcp__plugin_figma_figma__get_screenshot` of the corresponding Figma node from `docs/design.md` (fileKey `FDkPi2WrDztxvhQeWQtgWo`) saved to `<screen>-<viewport>-figma.png`.
4. Update `INDEX.md` with a one-line "matches / differs (reason)" entry per pair.

The verifier compares the two PNGs side by side; differences must be acknowledged in `INDEX.md`.

## Routes

The 7 MVP routes are registered in `src/app/routes.ts` and live as separate slices under `src/pages/`:

| Path                       | Slice                       | Figma nodes                          |
| -------------------------- | --------------------------- | ------------------------------------ |
| `/login`                   | `pages/auth-login/`         | 1:674, 1:740, 1:760, 1:695           |
| `/register`                | `pages/auth-register/`      | 1:719                                |
| `/decks`                   | `pages/decks-list/`         | 1:816, 1:852                         |
| `/account`                 | `pages/account-settings/`   | 1:891, 1:939                         |
| `/decks/new`               | `pages/deck-create/`        | 1:990, 1:1054                        |
| `/decks/:id/practice`      | `pages/deck-practice/`      | 1:1122, 1:1473, 1:1202, 1:1285       |
| `/decks/:id/results`       | `pages/practice-results/`   | 1:1369, 1:1419                       |

## Conventions checklist (run before declaring done)

- [ ] Atoms / actions / computed are named.
- [ ] No `useEffect + fetch`; reads via `withAsyncData`, mutations via `withAsync`.
- [ ] Every DOM handler that mutates state is wrapped with `wrap(...)`.
- [ ] No hex literals in TSX — only `bg-brand-500` / token utilities.
- [ ] Imports respect FSD direction; slices expose `index.ts` only.
- [ ] `pnpm typecheck` and `pnpm build` succeed.
- [ ] Playwright PNGs at 1440x900 + 390x844 saved alongside the Figma reference.
