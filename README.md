# Lorecraft

A graph-backed knowledge management system for world-building and campaign management.

## Motivation

I'm developing an RPG setting and planning to run a campaign in it. I was using
OpenCode with markdown files to build out the world, and it worked well at
first, but it didn't scale. AI agents kept losing track of what existed,
contradicting earlier content, and forgetting relationships between entities. I
needed a structured knowledge graph that the AI could query mid-session, so it
could answer "who controls Westport?" or "what factions operate in the
Westlands?" without me having to paste context into every prompt.

Lorecraft is the toolset that came out of that need. Markdown files remain the
source of truth (you still write and edit them like normal), but lorecraft
parses them into a PostgreSQL database and exposes that data to AI agents via the
Model Context Protocol. The database is a materialised view that can always be
destroyed and rebuilt from the source files.

While the initial use case is a fantasy RPG setting, lorecraft is
domain-agnostic. Entity types, relationships, and properties are all defined in
a `schema.yaml` file. Swap the schema and it works for fiction writing, product
knowledge bases, or anything else with interconnected entities.

## How it works

```
Markdown files       Ingestion (Go CLI)       PostgreSQL (Docker)
  with YAML    --->  lorecraft ingest    --->  Relational database
  frontmatter                                       |
                                                     |
AI agents        MCP Server (stdio)                 |
  (OpenCode,  <-->  lorecraft serve   <-------------+
    Claude, etc)
```

1. You write markdown files with YAML frontmatter that define entities
   (NPCs, locations, factions, whatever your schema declares).
2. `lorecraft ingest` parses those files and upserts them into PostgreSQL as
   entity records and relationships.
3. `lorecraft serve` starts an MCP server over stdio that AI agents can
   query to look up entities, search, and traverse relationships.
4. The CLI also provides direct query and validation commands.

Ingestion is incremental by default. Files that haven't changed since the last
run are skipped based on content hashes stored in the database.

## Prerequisites

- Go 1.22+
- Docker (for PostgreSQL)

## Quickstart

Clone the repo and start PostgreSQL:

```sh
git clone https://github.com/yourusername/lorecraft.git
cd lorecraft
make db-up
```

Wait a few seconds for PostgreSQL to start. The repo includes a working example
project under `example/`.

Build and run the example:

```sh
make build
./bin/lorecraft --version
cd example
../bin/lorecraft ingest
../bin/lorecraft validate
```

If you want to use OpenCode with the example, open `example/` as your project
root. The MCP config lives at `example/.opencode/opencode.json`.

To start your own project from scratch, create a new directory and copy the
schema template:

```sh
mkdir -p my-setting
cp schemas/fantasy-rpg.yaml my-setting/schema.yaml
```

Create `lorecraft.yaml` inside `my-setting/`:

```yaml
project: my-setting
version: 1

database:
  dsn: "postgres://lorecraft:changeme@localhost:5432/lorecraft?sslmode=disable"

layers:
  - name: setting
    paths:
      - ./lore/
    canonical: true
```

Then add content under `my-setting/lore/` and run lorecraft from that
directory:

```sh
cd my-setting
../bin/lorecraft ingest
```

## Configuration

Lorecraft uses two configuration files in the project directory (the directory
where you run `lorecraft`).

### lorecraft.yaml

The project config. Defines the database connection, content layers, and
file exclusions.

```yaml
project: my-setting
version: 1

database:
  dsn: "postgres://lorecraft:changeme@localhost:5432/lorecraft?sslmode=disable"

layers:
  - name: setting
    paths: [./lore/]
    canonical: true

  - name: campaign
    paths: [./campaigns/shadow-war/]
    canonical: false
    depends_on: [setting]

exclude:
  - ./assets/
```

Layers are processed in order. A layer with `depends_on` can reference
entities from its parent layers. Canonical layers are the persistent source of
truth; non-canonical layers track what happened during a specific campaign.

### schema.yaml

Defines entity types, their properties, field-to-relationship mappings, and
relationship types. A bundled template lives at `schemas/fantasy-rpg.yaml`.

Entity types declare which frontmatter fields are stored as properties,
which are mapped to relationships, and what validation rules apply:

```yaml
entity_types:
  - name: npc
    properties:
      - { name: role, type: string }
      - { name: status, type: enum, values: [alive, dead, unknown], default: alive }
    field_mappings:
      - { field: location, relationship: LOCATED_IN, target_type: [settlement, region] }
      - { field: faction, relationship: MEMBER_OF, target_type: [faction] }
```

Relationship types can have inverses or be symmetric:

```yaml
relationship_types:
  - { name: MEMBER_OF, inverse: HAS_MEMBER }
  - { name: ALLIED_WITH, symmetric: true }
```

## Writing content

Each markdown file with valid frontmatter becomes an entity in the database.

Required frontmatter fields:
- `title` -- the entity name
- `type` -- must match an entity type in your schema

Optional built-in fields:
- `tags` -- a list of tags for categorisation and full-text search
- `related` -- a list of entity names; creates `RELATED_TO` edges

Any other frontmatter field that matches a property in the schema is stored as
a property. Fields that match a `field_mapping` in the schema create
relationships to the named target entities.

The markdown body text after the frontmatter is stored and indexed for full-text
search.

Example:

```markdown
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

Lysa Quent is the calculating director of the Bureau of Civic Affairs.
```

This creates an NPC entity with `role` and `status` as properties,
`LOCATED_IN` and `MEMBER_OF` edges to the referenced entities, and
`RELATED_TO` edges to the listed related entities. If a target entity doesn't
exist yet, a placeholder is created and resolved on the next ingestion.

## CLI reference

### ingest

Synchronise the database with markdown source files.

```sh
lorecraft ingest          # incremental (skips unchanged files)
lorecraft ingest --full   # force full re-ingestion
```

### validate

Run consistency checks against the database. Reports dangling placeholders,
orphaned entities, duplicate names, invalid enum values, and missing required
properties.

```sh
lorecraft validate
```

### query entity

Display a single entity and its properties.

```sh
lorecraft query entity "Westport"
lorecraft query entity "Lysa Quent" --type npc
```

### query relations

Display relationships for an entity.

```sh
lorecraft query relations "Westport"
lorecraft query relations "Westport" --depth 2
lorecraft query relations "Westport" --type PART_OF --direction incoming
```

### query list

List entities, optionally filtered.

```sh
lorecraft query list
lorecraft query list --type npc
lorecraft query list --layer setting --tag politics
```

### query search

Full-text search across entity names, tags, and body text. Returns snippets
with highlighted matches.

```sh
lorecraft query search "bureau"
lorecraft query search "port" --type settlement
```

### query state

Compute current state for an entity from campaign events.

```sh
lorecraft query state "Westport" --layer campaign-shadow-war
```

### query sql

Execute a raw SQL query against the database.

```sh
lorecraft query sql "SELECT name, entity_type FROM entities LIMIT 10"
lorecraft query sql "SELECT name FROM entities WHERE layer = \$1" --param 1=setting
```

### serve

Start the MCP server over stdio.

```sh
lorecraft serve
```

### init

Scaffold a new project in the current directory.

```sh
lorecraft init --name my-setting
```

## MCP server

Lorecraft exposes the database to AI agents via the Model Context Protocol. The
server communicates over stdio and provides these tools:

- `search_lore` -- full-text search across entity names, tags, and body text with snippets
- `get_entity` -- retrieve a single entity with all properties and body text
- `get_relationships` -- traverse relationships from an entity with configurable depth and direction
- `list_entities` -- list entities filtered by type, layer, or tag
- `get_schema` -- return the full schema definition
- `get_current_state` -- compute current state for an entity in a campaign layer
- `get_timeline` -- return ordered campaign events for a layer
- `check_consistency` -- return entity, relationships, and events for review

To configure lorecraft as an MCP server for OpenCode, create
`.opencode/opencode.json` in your project directory:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "lorecraft": {
      "type": "local",
      "command": ["../bin/lorecraft", "serve"],
      "enabled": true
    }
  }
}
```

Other MCP-compatible clients (Claude Desktop, etc.) can use the same
`lorecraft serve` command with their own configuration format.
If your binary lives elsewhere, adjust the command path accordingly.

## Development

The Makefile provides common targets:

```sh
make build     # compile to bin/lorecraft
make test      # run unit tests
make fmt       # go fmt
make vet       # go vet
make tidy      # go mod tidy
make db-up     # start PostgreSQL via Docker Compose
make db-down   # stop PostgreSQL
make db-logs   # tail PostgreSQL logs
```

## License

MIT. Thanks to all the folks out there who have made the software I use.