# Breaking Changes in Yao 2.0

**Status:** Draft — Yao 2.0 is under active development.
**Last updated:** 2026-11-09

## Summary

Yao 2.0 is a **clean, intentionally incompatible release**.

It removes the 1.0 agent implementation, the 1.0 Tai integration, and the 1.0
workspace implementation. There will be **no runtime compatibility layer** and
**no drop-in upgrade path** inside the Yao binary.

If you maintain a fork, deploy Yao in production, or have custom code that
imports Yao packages, **do not merge `main` blindly after 2.0 development
starts**. Read this document first.

## What will be removed

The following 1.0 areas are planned for removal in 2.0:

- `agent/` — the 1.0 agent stack (assistants, robots, boards, inbox, memory,
  context, search, agent LLM providers, agent sandbox, etc.)
- `tai/` — the 1.0 Tai runtime integration (proxy, tunnel, runtime, volume,
  webproxy, vnc, hostexec, registry, etc.)
- `workspace/` — the 1.0 workspace implementation
- related packages under `openapi/`, `cmd/`, `grpc/`, `tools/`, `sui/`, and
  `unit-test/`
- related CI workflows and test fixtures

Custom code importing paths such as:

```go
github.com/yaoapp/yao/agent/...
github.com/yaoapp/yao/tai/...
github.com/yaoapp/yao/workspace/...
```

will need to be ported to the 2.0 APIs. The old import paths and types are not
guaranteed to exist in 2.0.

## What replaces the 1.0 Tai integration

Yao 2.0 will only keep the **`tai-golang` bridge** for Tai-related integration:

- https://github.com/yaoapp/tai-golang

The 1.0 in-tree `tai/` implementation will **not** be migrated. If your
deployment depends on the 1.0 Tai integration, plan to integrate through
`tai-golang` instead.

## Why this is necessary

Yao 2.0 is a structural rewrite. Keeping 1.0 compatibility inside the runtime
would preserve obsolete data models, workspace layouts, and agent/tai coupling,
and would make the 2.0 codebase permanently harder to evolve.

The migration is therefore handled by a **separate upgrade program**, not by
the Yao runtime.

## Migration plan

The upgrade path is a standalone tool/program that converts:

1. **Data**
   - database schema and records
   - 1.0 → 2.0 table/collection mapping
   - ID and relationship mapping
   - historical agent/robot/task/inbox data: migrate, archive, or drop (TBD per domain)
2. **Workspace**
   - 1.0 workspace file trees
   - paths, permissions, git/config credential layout
   - references to workspace paths stored in the database

Planned characteristics of the upgrade program:

- runs **outside** the Yao runtime
- reads from 1.0, writes to a **new** 2.0 database/workspace (no in-place overwrite by default)
- requires a backup before starting
- supports `--dry-run`, resume, and idempotent re-runs
- produces a migration report (success / skipped / failed items)
- verifies row counts, relationships, and workspace checksums
- keeps the 1.0 environment intact for rollback

The schema diff is derived from a frozen 1.0 baseline branch and the completed
2.0 models. The upgrade tool and its documentation will be published separately
(TBD).

## For fork maintainers and self-hosted users

Recommended actions:

1. **Pin to a 1.0 release/tag or your fork's stable branch.** Do not track
   Yao 2.0 `main` unless you are actively porting.
2. **Do not merge 2.0 `main` into a production fork** without reviewing the
   removed packages and migration requirements.
3. **Back up the database and workspace** before any 2.0 trial.
4. **Test the upgrade on a copy** of your data, in a staging environment.
5. **Port custom code** that imports `agent/`, `tai/`, or `workspace/`.
   There will be no compatibility shim.
6. Watch **https://github.com/yaoapp/tai-golang** for the new Tai bridge.
7. Watch the Yao release notes for the 2.0 upgrade tool when it becomes
   available.

Merging Yao 2.0 code into a 1.0 fork without migrating data/workspace will
break the application. Updating the binary alone is **not** a supported upgrade
path.

## Status and timeline

- Yao 2.0 is under development; no stable release date is announced here.
- 1.0 remains the supported release line until 2.0 and its upgrade tool are
  ready.
- This document will be updated when the 2.0 migration tool, target data
  formats, and upgrade commands are finalized.

## Version baseline branch

The frozen pre-2.0 snapshot branch used for data-structure comparison during the
1.0 → 2.0 migration is:

- https://github.com/YaoApp/yao/tree/v2.0.0-prepare

## Links

- Tai bridge: https://github.com/yaoapp/tai-golang
- Yao repository: https://github.com/yaoapp/yao
- Yao 2.0 development note: see the v2.0 notice in [`README.md`](README.md)
