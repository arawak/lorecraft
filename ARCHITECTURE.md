# Lorecraft Architecture Document

## Overview

Lorecraft is a graph-backed knowledge management system with an MCP interface, designed for managing interconnected lore and world-building content. While the initial use case is a D&D campaign setting, the system is domain-agnostic — entity types, relationship types, and content structure are defined through configuration, not code.

The core insight: **markdown files are the source of truth; the database is a materialised view.** Content is authored and maintained as markdown with structured YAML frontmatter. Lorecraft parses these files, builds a graph of entities and relationships in PostgreSQL, and exposes that graph to AI coding agents via an MCP server. The database can always be destroyed and rebuilt from the source files.

## Key Concepts

### Layers

Content is organised into **layers** with explicit dependency relationships and canonicity:

- **Canonical layers** (e.g., a campaign setting) are the persistent, authoritative source of truth. They define what *is*.
- **Ephemeral layers** (e.g., an adventure or campaign) reference canonical layers and track state changes over time. They define what *happened*.

A campaign layer can reference setting entities but cannot override them directly. State changes flow through **event nodes** — if an NPC dies in a campaign, the setting node is unchanged; the campaign layer records an event that affects that entity.

Layers are composable. Multiple campaigns can depend on the same setting. A "what if" variant can layer over a base setting with overrides. Non-D&D use cases map naturally: a fiction writer has a "world bible" (canonical) and "novel drafts" (ephemeral); a product team has "architecture" (canonical) and "sprint work" (ephemeral).

### Schema-Driven Configuration

All entity types, relationship types, property definitions, and frontmatter-to-graph field mappings are defined in a `schema.yaml` file. The codebase contains no domain-specific knowledge — a D&D setting, a sci-fi novel, and a product knowledge base all use the same lorecraft binary with different schema configurations.

Templates (e.g., `fantasy-rpg`, `fiction`, `knowledge-base`) provide sensible starter schemas, but these are just YAML files the user customises.

### Entities and Relationships

Entities are records created from markdown files. Each file with valid frontmatter containing a recognised `type` field becomes an entity. Entity properties come from frontmatter fields.

Relationships are edges created from frontmatter field mappings defined in the schema. For example, a schema might define that an `npc` entity's `faction` frontmatter field produces a `MEMBER_OF` edge to a `faction` entity. Relationships can be directional, have inverses, or be symmetric.

### Events and Temporal State

Ephemeral layers introduce time through event nodes. An event records something that happened during a campaign session and its consequences — which entities were affected and how their properties changed.

The system can compute **current state** for any entity by starting with the canonical layer's definition and applying all events from a given campaign layer in session order. This allows the AI to answer "what is the state of Port Valdris right now in this campaign?" without the setting itself being modified.

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│  Markdown    │────▶│  Ingestion   │────▶│ PostgreSQL │
│  Files       │     │  (Go CLI)    │     │  (Docker)  │
└─────────────┘     └──────────────┘     └──────┬──────┘
                                                │
┌─────────────┐     ┌──────────────┐            │
│  OpenCode    │◀──▶│  MCP Server  │◀───────────┘
│  + agents    │     │  (Go, stdio) │
└─────────────┘     └──────────────┘
```

All components talk to the same PostgreSQL instance. The CLI is the human operator's interface; the MCP server is the AI's interface. Both share the same internal Go packages.

### Components

**PostgreSQL** — Relational database with full-text search, runs as a Docker container on the host. Stores all entities, relationships, and events. Accessed via port 5432.

**Lorecraft CLI** — Single Go binary providing all commands: `init`, `ingest`, `validate`, `query`, and `serve`. The CLI reads `lorecraft.yaml` for project configuration and `schema.yaml` for the domain model.

**Lorecraft MCP Server** — Started via `lorecraft serve`. Runs as a long-lived process communicating over stdin/stdout using the MCP JSON-RPC protocol. Exposes query tools to AI agents (OpenCode, subagents). Holds a persistent connection to PostgreSQL for the lifetime of the process.

## Project Structure (Go)

```
lorecraft/
├── cmd/
│   └── lorecraft/              # Single binary entrypoint
│       └── main.go
├── internal/
│   ├── config/                 # Project + schema YAML loading
│   ├── parser/                 # Markdown frontmatter parsing
│   ├── store/                  # Storage interface + types
│   │   └── postgres/           # PostgreSQL implementation
│   ├── ingest/                 # Pipeline: parse → validate → upsert
│   ├── validate/               # Schema + consistency checks
│   └── mcp/                    # MCP tool definitions + handlers
├── schemas/                    # Bundled starter schema templates
│   ├── fantasy-rpg.yaml
│   └── fiction.yaml
├── compose.yaml                # PostgreSQL service definition
└── go.mod
```

The `cmd/lorecraft/main.go` uses cobra to define subcommands. All subcommands share the `internal/` packages. The MCP server (`lorecraft serve`) and CLI commands (`lorecraft ingest`, `lorecraft query`, etc.) use the same `store.Store` interface, `config.Schema`, and `parser` packages.

## Configuration Files

### `lorecraft.yaml` — Project Configuration

Lives at the project root. Defines the project name, database connection DSN, layer definitions, and exclusion rules.

```yaml
project: westlands
version: 1

database:
  dsn: "postgres://lorecraft:changeme@localhost:5432/lorecraft?sslmode=disable"

layers:
  - name: setting
    paths:
      - ./lore/
    canonical: true

  # Example ephemeral layer (uncomment when starting a campaign):
  # - name: campaign-shadow-war
  #   paths:
  #     - ./campaigns/shadow-war/
  #   canonical: false
  #   depends_on: [setting]

exclude:
  - ./assets/
```

**Layer fields:**
- `name` — unique identifier for the layer
- `paths` — list of directories to recursively scan for markdown files
- `canonical` — whether this layer is authoritative (true) or ephemeral (false)
- `depends_on` — list of layer names this layer can reference (ephemeral layers only)

**Exclusions:** Lorecraft only ingests markdown files whose frontmatter contains a `type` field matching a type defined in the schema. Files without frontmatter or without a recognised type are silently skipped. The `exclude` list provides an additional mechanism to skip entire directories or specific files.

### `schema.yaml` — Domain Model

Defines entity types, their properties, allowed relationships, and how frontmatter fields map to graph relationships.

```yaml
version: 1

entity_types:
  - name: region
    properties:
      - { name: climate, type: string }
      - { name: terrain, type: string }
      - { name: population, type: string }
    field_mappings:
      - { field: parent_region, relationship: PART_OF, target_type: [region] }

  - name: settlement
    properties:
      - { name: size, type: enum, values: [hamlet, village, town, city] }
      - { name: government, type: string }
    field_mappings:
      - { field: region, relationship: PART_OF, target_type: [region] }

  - name: npc
    properties:
      - { name: role, type: string }
      - { name: status, type: enum, values: [alive, dead, unknown], default: alive }
      - { name: visibility, type: enum, values: [player, gm], default: gm }
    field_mappings:
      - { field: location, relationship: LOCATED_IN, target_type: [settlement, region] }
      - { field: faction, relationship: MEMBER_OF, target_type: [faction] }

  - name: faction
    properties:
      - { name: faction_type, type: string }
      - { name: influence, type: enum, values: [local, regional, continental] }
    field_mappings:
      - { field: operates_in, relationship: OPERATES_IN, target_type: [settlement, region] }

  - name: lore
    properties:
      - { name: topic, type: string }

  - name: event
    properties:
      - { name: session, type: integer }
      - { name: date_in_world, type: string }
    field_mappings:
      - { field: participants, relationship: INVOLVES, target_type: [npc, faction] }
      - { field: location, relationship: OCCURS_IN, target_type: [settlement, region] }
      - { field: affects, relationship: AFFECTS }

  - name: rumour
    properties:
      - { name: verified, type: enum, values: [true, false, unknown], default: unknown }
      - { name: source_npc, type: string }
    field_mappings:
      - { field: about, relationship: ABOUT }

relationship_types:
  - { name: PART_OF, inverse: CONTAINS }
  - { name: LOCATED_IN, inverse: HAS_PRESENT }
  - { name: MEMBER_OF, inverse: HAS_MEMBER }
  - { name: OPERATES_IN, inverse: HAS_FACTION }
  - { name: ALLIED_WITH, symmetric: true }
  - { name: HOSTILE_TO, symmetric: true }
  - { name: GOVERNS, inverse: GOVERNED_BY }
  - { name: KNOWS, symmetric: true }
  - { name: RELATED_TO, symmetric: true }
  - { name: AFFECTS, inverse: AFFECTED_BY }
  - { name: INVOLVES, inverse: INVOLVED_IN }
  - { name: OCCURS_IN, inverse: HAS_EVENT }
  - { name: TRADES_WITH, symmetric: true }
  - { name: CONTROLS, inverse: CONTROLLED_BY }
  - { name: ABOUT, inverse: SUBJECT_OF }
```

**Field mappings** define how frontmatter fields on a given entity type translate into relationships. When the ingestion pipeline encounters an NPC with `faction: Bureau of Civic Affairs` in its frontmatter, it looks up the field mapping for `faction` on the `npc` entity type and creates a `MEMBER_OF` edge to the entity matching that name.

**The `related` field** is available on all entity types as a catch-all that produces `RELATED_TO` edges. It does not need to be declared per entity type — it is a built-in convention. All other relationship-producing fields must be explicitly mapped in the schema.

**Relationship types** define directionality. A relationship with `inverse` creates edges in one direction but allows traversal and querying from either end using the inverse name. A `symmetric` relationship has no inherent direction.

### Frontmatter Conventions

Every markdown file that lorecraft should ingest must have YAML frontmatter containing at minimum:

```yaml
---
title: <Display name of the entity>
type: <Entity type matching schema.yaml>
---
```

All other fields are optional and depend on the entity type's property definitions and field mappings in the schema. Example for an NPC:

```yaml
---
title: Bureau Director Lysa Quent
type: npc
role: Bureau Director
status: alive
visibility: gm
location: Westport
faction: Bureau of Civic Affairs
related: [Overlord Rellan Harth, Selin Hale]
tags: [bureaucracy, politics, power-broker]
---
```

**Field value resolution:** Relationship fields (those with field mappings) reference target entities by their `title`. The ingestion pipeline matches these by case-insensitive name lookup. If a referenced entity does not yet exist, the pipeline creates a placeholder entity and logs a warning. Subsequent ingestion runs will resolve the placeholder when the target file is created.

**List fields:** Fields that reference multiple entities use YAML list syntax. The pipeline creates one edge per list item.

**Tags:** The `tags` field is a reserved field available on all entity types. Tags are stored on the entity and indexed for full-text search. They do not produce relationships.

### Event Frontmatter (Ephemeral Layers)

Events in campaign layers track state changes and record what happened during play sessions:

```yaml
---
title: The Fall of Port Valdris
type: event
session: 12
date_in_world: "3rd Age, Year 847, Month of Storms"
date_real: 2025-02-15
participants: [Kira Ashwood, The Corsair Brotherhood]
location: Port Valdris
affects: [Port Valdris, Corsair Brotherhood]
consequences:
  - entity: Port Valdris
    property: status
    value: destroyed
  - entity: Corsair Brotherhood
    property: territory
    add: Iron Coast
---
```

The `consequences` field is special — it defines how this event modifies the properties of other entities. The `get_current_state` operation uses these to compute the effective state of an entity at any point in a campaign's timeline.

## CLI Commands

### `lorecraft init [--name <name>] [--template <template>]`

Scaffolds a new lorecraft project by creating `lorecraft.yaml` and `schema.yaml` in the current directory. If `--template` is specified, the schema is pre-populated with entity types and relationships appropriate to the domain (e.g., `fantasy-rpg`, `fiction`, `knowledge-base`). Does not create directories or modify existing files.

### `lorecraft ingest`

Reads `lorecraft.yaml` and `schema.yaml`, connects to PostgreSQL, and synchronises the database with the current state of the markdown files.

**Process:**
1. Load project config and schema
2. Connect to PostgreSQL
3. For each configured layer, recursively walk the specified paths
4. For each `.md` file, parse YAML frontmatter
5. Skip files without frontmatter or without a recognised `type` field
6. Validate entity properties against the schema
7. Upsert the entity in PostgreSQL (by name_normalized + layer)
8. For each field mapping, resolve the target entity and upsert the relationship
9. Remove entities whose source files no longer exist (track by file path stored as entity property)
10. Print summary: entities created/updated/deleted, relationships created, warnings

**Incremental ingestion:** On subsequent runs, lorecraft compares file modification times (or content hashes) against the last-ingested timestamp stored on each entity. Only changed files are re-processed. A `--full` flag forces complete re-ingestion.

**Dangling references:** If a frontmatter field references an entity that doesn't exist yet, lorecraft creates a placeholder entity and logs a warning. Placeholder entities are resolved on subsequent ingestion runs when the target file is created.

### `lorecraft validate`

Runs consistency checks against the database and reports issues:

- **Schema violations** — entity has a property value not permitted by the schema (e.g., invalid enum value), or a relationship type not allowed for that entity type
- **Dangling references** — placeholder entities that remain unresolved
- **Orphaned entities** — entities with no relationships (may be intentional but worth flagging)
- **Cross-layer violations** — ephemeral layer entities directly modifying canonical layer entity properties (should go through events)
- **Duplicate names** — two entities with the same title in the same layer
- **Missing required properties** — as defined in the schema

Output is human-readable with file paths for each issue so the user can locate and fix the source file.

### `lorecraft query <subcommand>`

Direct database interrogation from the terminal.

- `lorecraft query entity <name>` — display an entity's properties and body text
- `lorecraft query relations <name> [--depth N]` — display relationships up to N hops (default 1)
- `lorecraft query list [--type <type>] [--layer <layer>] [--tag <tag>]` — filtered entity listing
- `lorecraft query state <name> --layer <layer>` — display the computed current state of an entity in a given campaign layer (canonical properties + all events applied in session order)
- `lorecraft query search <text>` — full-text search across entity names, tags, and body text (returns snippets)
- `lorecraft query sql <query>` — execute a raw SQL query (power user escape hatch)

### `lorecraft serve`

Starts the MCP server. Reads `lorecraft.yaml` and `schema.yaml`, connects to PostgreSQL, and listens on stdin/stdout for MCP JSON-RPC messages. Intended to be configured as an MCP server in the AI agent's configuration (e.g., OpenCode's MCP settings).

The server remains running until the parent process terminates or stdin is closed.

## MCP Server

### Transport

MCP uses stdio transport — the host process (e.g., OpenCode) spawns `lorecraft serve` as a child process and communicates via JSON-RPC over stdin/stdout. No HTTP, no ports, no network configuration.

### Tool Definitions

The MCP server exposes the following tools to the AI agent. Tool descriptions are generated at startup from the schema, so the AI receives domain-aware descriptions (e.g., "Search for lore about regions, NPCs, factions..." rather than "Search for entities of configured types").

#### `search_lore`
Full-text search across entity names, tags, and body text.

**Parameters:**
- `query` (string, required) — search terms
- `layer` (string, optional) — restrict to a specific layer
- `type` (string, optional) — restrict to a specific entity type

**Returns:** List of matching entities with name, type, layer, score, and a snippet of matching text.

#### `get_entity`
Retrieve a specific entity and its properties.

**Parameters:**
- `name` (string, required) — entity name (case-insensitive match)
- `type` (string, optional) — disambiguate if multiple entities share a name

**Returns:** Entity name, type, layer, all properties, body text, source file path.

#### `get_relationships`
Traverse relationships from an entity and return connected nodes.

**Parameters:**
- `name` (string, required) — starting entity name
- `type` (string, optional) — relationship type filter (e.g., only `MEMBER_OF`)
- `depth` (integer, optional, default 1) — maximum traversal depth (1-5)
- `direction` (string, optional) — `outgoing`, `incoming`, or `both` (default `both`)

**Returns:** List of connected entities with the relationship type and direction at each hop.

#### `get_current_state`
Compute the effective state of an entity in a specific campaign layer by applying all events in session order to the canonical base.

**Parameters:**
- `name` (string, required) — entity name
- `layer` (string, required) — campaign layer name

**Returns:** Base properties from the canonical layer, list of events that affected this entity (in order), and the computed current property values.

#### `get_timeline`
Retrieve events in a campaign layer, optionally filtered by entity or time range.

**Parameters:**
- `layer` (string, required) — campaign layer name
- `entity` (string, optional) — filter to events involving a specific entity
- `from_session` (integer, optional) — start of session range
- `to_session` (integer, optional) — end of session range

**Returns:** List of events in session order with participants, location, and consequences.

#### `list_entities`
Filtered listing of entities.

**Parameters:**
- `type` (string, optional) — entity type filter
- `layer` (string, optional) — layer filter
- `tag` (string, optional) — tag filter

**Returns:** List of entity names and types matching the filters.

#### `check_consistency`
Given a proposed fact about an entity, check for potential contradictions.

**Parameters:**
- `entity` (string, required) — entity name
- `claim` (string, required) — the proposed fact in natural language

**Returns:** Any existing properties, relationships, or events that might conflict with the claim, or confirmation that no contradictions were found.

**Implementation note:** This tool queries the entity, its immediate relationships, and any relevant events, then returns them as context. The AI agent performs the actual consistency reasoning — lorecraft provides the evidence, not the judgement.

#### `get_schema`
Return the current schema definition so the AI knows what entity types, relationship types, and properties exist.

**Parameters:** None

**Returns:** The full schema as structured data.

## PostgreSQL Data Model

### Tables

**entities** — Stores all entity records:

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT | Primary key |
| name | TEXT | Entity title |
| name_normalized | TEXT | Lowercase name for case-insensitive lookup |
| entity_type | TEXT | Entity type from schema |
| layer | TEXT | Layer name |
| source_file | TEXT | Path to source markdown file |
| source_hash | TEXT | Content hash for incremental ingestion |
| tags | TEXT[] | Array of tags |
| properties | JSONB | Custom properties |
| body | TEXT | Markdown prose after frontmatter |
| is_placeholder | BOOLEAN | Whether this is an unresolved reference |
| last_ingested | TIMESTAMPTZ | Last ingestion timestamp |
| search_vector | TSVECTOR | Full-text search vector |

**edges** — Stores relationships between entities:

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT | Primary key |
| src_id | BIGINT | Source entity ID (FK) |
| dst_id | BIGINT | Destination entity ID (FK) |
| rel_type | TEXT | Relationship type |

**events** — Stores event data for temporal queries:

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT | Primary key |
| entity_id | BIGINT | Event entity ID (FK) |
| layer | TEXT | Layer name |
| session | INTEGER | Session number |
| date_in_world | TEXT | In-world date string |
| consequences | JSONB | Array of consequences |

### Indexes

- Unique constraint on `(name_normalized, layer)` — no duplicate entity names within a layer
- GIN index on `search_vector` for full-text search
- Index on `layer` for efficient layer-scoped queries
- Index on `source_file` for ingestion lookups
- GIN index on `tags` for tag filtering
- Indexes on edge columns for relationship traversal

### Full-Text Search

The `search_vector` column uses weighted tsvector:
- Weight A: entity name (`simple` dictionary — no stemming, preserves fantasy proper nouns)
- Weight B: tags (`english` dictionary)
- Weight C: body text (`english` dictionary)

Search uses `websearch_to_tsquery` for forgiving query parsing and `ts_headline` for snippet generation with `**` markers around matches.

## Internal Go Packages

### `internal/config`
Loads and validates `lorecraft.yaml` and `schema.yaml`. Provides typed structs for project configuration, layer definitions, entity type definitions, field mappings, and relationship type definitions.

### `internal/parser`
Parses markdown files to extract YAML frontmatter. Returns a `Document` struct containing the frontmatter as a map, the entity type, and the prose body. Does not interpret field mappings — that is the ingestion pipeline's responsibility.

### `internal/store`
Defines the `Store` interface that abstracts storage operations. All consumer packages (ingest, validate, MCP) depend on this interface, not on the concrete PostgreSQL implementation.

Domain types (`Entity`, `EntityInput`, `Relationship`, `SearchResult`, etc.) are defined here.

### `internal/store/postgres`
PostgreSQL implementation of `store.Store`. Uses `pgx` for connection pooling. Exposes domain operations:
- `UpsertEntity(entity)` — insert or update entity by name_normalized + layer
- `UpsertRelationship(from, to, relType)` — insert edge, creating placeholder if needed
- `GetEntity(name, type)` — fetch entity and properties
- `GetRelationships(name, depth, relType, direction)` — iterative BFS traversal
- `Search(query, layer, type)` — full-text search with snippets
- `GetCurrentState(name, layer)` — canonical base + events composited
- `GetTimeline(layer, entity, fromSession, toSession)` — event queries
- `ListEntities(type, layer, tag)` — filtered listing
- `RunSQL(query, params)` — raw SQL escape hatch

All methods accept `context.Context` and return domain-typed results.

### `internal/ingest`
Orchestrates the ingestion pipeline. Walks directory trees, invokes the parser, resolves field mappings against the schema, calls store operations to upsert entities and edges, handles incremental updates via content hashing, and tracks deletions.

The pipeline is structured as a series of stages:
1. **Parse** — extract frontmatter from markdown
2. **Validate** — check entity type and properties against schema
3. **Upsert Entity** — create or update the entity record
4. **Resolve Relationships** — map frontmatter fields to edges via schema field mappings
5. **Upsert Edges** — create or update relationships
6. **Cleanup** — remove entities for deleted files

This stage structure allows future extension (e.g., NLP entity extraction, custom date parsing) by inserting additional stages into the pipeline.

### `internal/validate`
Implements consistency checks. Queries the store to detect schema violations, dangling references, orphaned entities, cross-layer violations, and duplicates. Returns a structured report of issues with severity levels and source file paths.

### `internal/mcp`
MCP protocol handling and tool definitions. Defines the JSON-RPC message handling, tool registration, and parameter/result marshalling. Each tool handler delegates to `store.Store` methods. Tool descriptions are generated from the loaded schema so the AI receives domain-aware documentation.

## Future Considerations (Not in Scope for Initial Build)

- **Semantic search via embeddings** — add an `internal/embeddings` package with an `Embedder` interface. Generate embeddings via any provider API (OpenAI, Voyage, etc.) and store using pgvector. Adds a `search_lore_semantic` tool to the MCP server. The embedding provider is configured in `lorecraft.yaml` and implementation swapped via the interface.

- **File watching** — a `lorecraft watch` daemon mode that monitors content directories and re-ingests changed files automatically, keeping the database continuously in sync.

- **Export** — `lorecraft export` to dump the database or subgraphs as structured data (JSON, markdown tables, gazetteers) for session prep, player handouts, or integration with other tools.

- **NLP entity extraction** — an ingestion pipeline stage that analyses prose body content to discover implicit entity references not captured in frontmatter, and suggests or auto-creates relationships.

- **Graph visualisation export** — export subgraphs in formats compatible with visualisation tools (e.g., Mermaid diagrams, Graphviz DOT).

## Build Sequence

### Phase 1: Prove the Database (Friday Night)
- PostgreSQL running via Docker Compose
- Go module initialised with project structure
- Config and schema YAML loading
- Basic frontmatter parser
- Store interface with `UpsertEntity` and `UpsertRelationship`
- Enough of `lorecraft ingest` to populate entities and see them in the database

**Checkpoint:** Can you see your entities and relationships in the database? Does the data structure reveal useful things about your setting?

### Phase 2: MCP Server (Saturday)
- Complete ingestion pipeline with field mapping resolution
- Store query methods: `GetEntity`, `GetRelationships`, `ListEntities`, `Search`
- MCP server with tool definitions wired to store
- Integration with OpenCode

**Checkpoint:** Can the AI answer "what do I know about Westport?" by querying the database through MCP?

### Phase 3: Validation and Polish (Sunday)
- `lorecraft validate` with core consistency checks
- `lorecraft query` CLI subcommands
- Layer system basics with dependency enforcement
- Incremental ingestion (content hashing, file deletion tracking)
- Error handling, logging, edge cases

**Checkpoint:** Is this actually useful in your daily workflow? Does the AI stay more consistent when writing new content?

### Phase 4: Temporal and Campaign Support (Post-Weekend)
- Event entity ingestion and consequence processing
- `get_current_state` and `get_timeline` MCP tools
- Cross-layer validation rules
- `lorecraft init` with templates