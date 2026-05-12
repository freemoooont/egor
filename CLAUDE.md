# Claude guide — Micocards (root)

This is the **authoritative** agent guide for the repository. The project intentionally consolidates rules into `CLAUDE.md` (root + per-area) instead of `AGENTS.md` per the user's preference.

## Global hard rules (apply to every agent and every session)

1. **Git commits are allowed.** Standard git workflow is in use (the prior project-wide ban was lifted on 2026-05-12 when the working tree was published to `github.com/freemoooont/egor`). Still avoid `--no-verify`, `--force` against `main`, and any destructive history rewrite without explicit user approval. The general "executing actions with care" guidance applies.
2. **Orchestrator-only main thread.** The top-level Claude session is an architect-lead orchestrator and MUST NOT call Edit/Write/NotebookEdit on production code or large docs. All implementation goes through Agent subagents. When the orchestrator's context fills up, it dumps state to `~/.claude/projects/-Users-vladislav-molotsilo-WebstormProjects-megaproject-egor/memory/` and hands off to the next orchestrator (a fresh Agent invocation).
3. **Frontend changes are verified via MCP Playwright + Figma reference.** Every implemented screen must be loaded with `mcp__plugin_playwright_playwright__browser_navigate` at desktop 1440x900 and mobile 390x844 (when the design has both), and a `mcp__plugin_figma_figma__get_screenshot` of the matching node from `docs/design.md` must be saved alongside, so the verifier can compare.
4. **Backend by TDD (pragmatic).** Domain and use-case layers must have unit tests written first. Infrastructure may follow integration-test-first via testcontainers. No "tests later" merges.
5. **DDD docs frozen before backend code.** `docs/backend/` must contain `ubiquitous-language.md`, `bounded-contexts.md`, `aggregates.md`, `domain-events.md`, `use-cases.md`, and `adr/0001-*.md` (plus subsequent ADRs) BEFORE any production Go code is written.
6. **Per-area `CLAUDE.md`.** `frontend/CLAUDE.md` and `backend/CLAUDE.md` are the authoritative per-area guides. When working inside either tree, read the local `CLAUDE.md` first; this root file only carries cross-cutting rules.
7. **Stack and design docs are authoritative.** `docs/stack.md` is the technology source of truth. `docs/design.md` is the screen source of truth.
8. **Repo-task-proof-loop is mandatory.** Use `scripts/task_loop.py` (in the skill `~/.claude/skills/repo-task-proof-loop/`) for non-trivial work. The active task is `TASK_ID=micocards-mvp`; artifacts live under `.agent/tasks/micocards-mvp/` (gitignored — kept locally as proof, not pushed).

If any rule conflicts with a managed block below, the rule above wins.

<!-- repo-task-proof-loop:start -->
## Repo task proof loop

For substantial features, refactors, and bug fixes, use the repo-task-proof-loop workflow.

Required artifact path:
- Keep all task artifacts in `.agent/tasks/<TASK_ID>/` inside this repository.

Required sequence:
1. Freeze `.agent/tasks/<TASK_ID>/spec.md` before implementation.
2. Implement against explicit acceptance criteria (`AC1`, `AC2`, ...).
3. Create `evidence.md`, `evidence.json`, and raw artifacts.
4. Run a fresh verification pass against the current codebase and rerun checks.
5. If verification is not `PASS`, write `problems.md`, apply the smallest safe fix, and reverify.

Hard rules:
- Do not claim completion unless every acceptance criterion is `PASS`.
- Verifiers judge current code and current command results, not prior chat claims.
- Fixers should make the smallest defensible diff.

Installed workflow agents:
- `.claude/agents/task-spec-freezer.md`
- `.claude/agents/task-builder.md`
- `.claude/agents/task-verifier.md`
- `.claude/agents/task-fixer.md`

Claude Code note:
- If `init` just created or refreshed these files during a running Claude Code session, start a new Claude Code session before relying on the updated agent list.
- Use `/agents` to inspect the available agents.
- Keep this block in the root `CLAUDE.md`. If the workflow needs longer repo guidance, prefer `@path` imports or `.claude/rules/*.md` instead of expanding this block.
<!-- repo-task-proof-loop:end -->
