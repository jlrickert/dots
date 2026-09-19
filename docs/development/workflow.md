# Docs-First Workflow

The documentation in [docs/](../) is the source of truth for how dots behaves. Code follows the docs, not the other way around. [dots-design.md](../../dots-design.md) is the original design record; where it conflicts with `docs/`, `docs/` wins.

## Making a Behavior Change

1. **Doc** — Update or add the user-facing doc (`docs/*.md`) describing the new behavior: fields, flags, examples, edge cases.
2. **Schema** — If the doc adds or changes a `Config`, `Manifest`, or `Profile` field, update the matching file in `schemas/`. See [schemas.md](schemas.md).
3. **Code** — Implement in `pkg/dots` / `pkg/dotsctl` / `pkg/cli` so it matches the doc.
4. **Tests** — Write tests that exercise the documented examples. See [testing.md](testing.md).
5. **Status** — Update the feature status table in [README.md](../../README.md): mark the row Done, or add a Partial/Planned row if the doc is ahead of the code.

## Resolving Drift

When code and docs disagree:

- Assume the doc is correct and fix the code.
- If the doc is actually wrong, change the doc on purpose as a spec change (`docs:` commit) and then bring the code in line.
- If the code can't catch up yet, keep the doc as written and record the gap in the README status table as Partial or Planned.
- Never quietly edit a doc to describe what the code happens to do.

## Commits

- Spec changes use `docs:`.
- The `feat:` / `fix:` that implements a doc can land in the same commit or a follow-up.
- Conventional commits, summary ≤72 chars, no co-author lines.
