# Agents

## Project Context

Lorecraft is a graph-backed knowledge management system with an MCP interface. Read `ARCHITECTURE.md` before making any structural decisions — it is the authoritative design document for this project.

## Language and Tooling

- **Go** — this is a Go project. Use standard library where possible. Minimise dependencies.
- **PostgreSQL** — relational database with full-text search, accessed via `github.com/jackc/pgx/v5`. No ORM. Raw SQL queries with parameterised inputs.
- **Cobra** — CLI framework for command structure.
- **MCP** — the MCP server communicates over stdio using JSON-RPC. Use an existing Go MCP library if a mature one is available; otherwise implement the protocol directly.

## Code Style

- Follow standard Go conventions: `gofmt`, `go vet`, no lint warnings.
- Error handling: return errors, don't panic. Wrap errors with context using `fmt.Errorf("doing thing: %w", err)`.
- No global state. Pass dependencies explicitly (config, store, schema).
- Interfaces at consumption site, not at definition site.
- Test files live alongside the code they test (`foo_test.go` next to `foo.go`).
- Keep packages focused. If a file is growing past 300 lines, consider whether it should split.

## Architecture Rules

- **`ARCHITECTURE.md` is the source of truth for design decisions.** If you're unsure about structure, check there first. If the architecture doc doesn't cover it, ask.
- **Markdown files are the source of truth for content. The database is a materialised view.** Never write logic that treats the database as authoritative over the files.
- **The codebase must contain no domain-specific knowledge.** No hardcoded entity types, relationship types, or D&D concepts. Everything domain-specific comes from `schema.yaml` configuration.
- **The CLI and MCP server share internal packages.** Don't duplicate logic between them. Both entrypoints use the same `store.Store` interface, `config`, `parser`, and `validate` packages.
- **All database interactions go through `internal/store`.** No other package should import pgx directly. Consumer packages depend on the `store.Store` interface.

## Workflow

- **Ask before adding new dependencies.** Justify why the standard library or an existing dependency can't do the job.
- **Ask before changing the project structure** defined in `ARCHITECTURE.md`. Propose the change and the reasoning.
- **Write tests for non-trivial logic** — particularly ingestion pipeline stages, schema validation, and SQL query construction. Integration tests against PostgreSQL can use a test container or a dedicated test database.
- **Start small, iterate.** Follow the build sequence in `ARCHITECTURE.md`. Don't jump ahead to Phase 3 before Phase 1 is working.

## Build Sequence

The architecture doc defines a phased build. Respect the phases:

1. **Phase 1:** PostgreSQL running, config/schema loading, parser, basic ingestion, entities visible in database
2. **Phase 2:** Full ingestion with field mappings, database query methods, MCP server, OpenCode integration
3. **Phase 3:** Validation, CLI query commands, incremental ingestion, polish
4. **Phase 4:** Temporal/event support, campaign layers, `lorecraft init`

Each phase has a checkpoint. Hit the checkpoint before moving on.

## Testing

- Use table-driven tests where appropriate.
- For store tests, create a `testdata/` directory with small fixture markdown files that exercise known entity types and relationships.
- Mock the `store.Store` interface for unit testing packages that depend on it.
- Integration tests that hit PostgreSQL should be tagged (`//go:build integration`) so they don't run without a database present.

## PostgreSQL

- PostgreSQL runs in Docker via the `compose.yaml` at the project root.
- Use `INSERT ... ON CONFLICT` for upserts — ingestion must be idempotent.
- Always use parameterised queries (`$1`, `$2`), never string interpolation into SQL.
- Entity types are stored as a column value, not as dynamic table names. No SQL injection risk for entity types.