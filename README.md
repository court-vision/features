# Court Vision — Features Service

Go services for compute-intensive features in the [Court Vision](https://github.com/court-vision) fantasy basketball analytics platform. The one service in production is **lineup generation**: given a manager's roster and the available free agent pool, it finds the sequence of daily add/drop moves that maximises projected value over a fantasy week.

The service runs an HTTP server on **port 8080**.

---

## Tech Stack

- **Language**: Go 1.22+
- **HTTP**: `net/http` standard library — no external framework, no third-party modules
- **Concurrency**: `sync.WaitGroup` + goroutines for parallel population evolution
- **Data**: Static NBA schedule JSON bundled into the image and loaded once at startup
- **Container**: Multi-stage Docker build (`golang:1.22.4` builder → `scratch` final image)

---

## Repository Structure

```
features/
├── main.go                        # Stub entry point (imports commented out)
├── go.mod                         # Root module (module: features, go 1.22.4)
├── Dockerfile                     # Builds and runs lineup-generation/v2
└── lineup-generation/
    ├── v2/                        # Production service
    │   ├── v2.go                  # Entry point, HTTP handlers, OptimizeStreaming()
    │   ├── v2_test.go             # Handler tests (httptest)
    │   ├── go.mod                 # module v2
    │   ├── data/                  # Schedule loader (LoadSchedule, HasWeek, WeekCount), Player types
    │   ├── population/            # EvolutionManager, Chromosome, Gene
    │   ├── team/                  # BaseTeam (pre-computed slotting metadata)
    │   ├── utils/                 # ReqBody, Response, Bench, helpers
    │   ├── resources/             # Mock JSON fixtures used by tests
    │   ├── static/                # Season schedule JSON files (schedule26-27.json, ...)
    │   ├── testutil/              # Resolves fixture/schedule paths relative to the source tree
    │   └── tests/                 # Unit tests for data, team and population packages
    └── v3/                        # Empty stub reserved for a future rewrite; nothing runs here
```

**v2 is the version deployed in production** (see `Dockerfile`).

---

## Setup & Running

### Prerequisites

- Go 1.22 or later (`go version`)

### Run locally

```bash
cd features/lineup-generation/v2
go run .
# Loaded schedule ./static/schedule26-27.json (24 weeks)
# Server started on :8080
```

The server reads the schedule once at startup and **exits immediately if it cannot** (missing file, malformed JSON, or no weeks). Point it at a different file with `SCHEDULE_FILE`:

```bash
SCHEDULE_FILE=./static/schedule25-26.json go run .
```

| Variable | Default | Description |
|---|---|---|
| `SCHEDULE_FILE` | `./static/schedule26-27.json` (`/app/static/schedule26-27.json` in Docker) | Path of the season schedule JSON to load |

### Build and run with Docker

The `Dockerfile` at the repo root builds v2:

```bash
# From features/
docker build -t cv-features .
docker run -p 8080:8080 cv-features
curl localhost:8080/healthz
```

The image is `scratch`-based and contains only the compiled binary, CA certificates, and the whole `static/` directory. A season rollover therefore only needs the new schedule file committed under `static/` and the `SCHEDULE_FILE` default (in `v2.go` and the `Dockerfile`) bumped — or the variable overridden on the Railway service.

---

## API

### `GET /healthz`

Liveness check that also reports which schedule the process loaded, so a deploy with a missing or stale schedule is visible from the health check rather than from empty lineup plans.

```json
{ "status": "ok", "schedule_file": "/app/static/schedule26-27.json", "weeks": 24 }
```

### `POST /generate-lineup`

Accepts roster and free agent data pre-fetched by the backend and returns an optimized day-by-day lineup plan for the requested fantasy week.

#### Request body

```json
{
  "roster_data": [
    {
      "name": "Giannis Antetokounmpo",
      "avg_points": 65.2,
      "team": "MIL",
      "valid_positions": ["PF", "C", "F", "UT1", "UT2", "UT3"],
      "injured": false
    }
  ],
  "free_agent_data": [
    {
      "name": "Isaiah Hartenstein",
      "avg_points": 28.4,
      "team": "OKC",
      "valid_positions": ["C", "UT1", "UT2", "UT3"],
      "injured": false
    }
  ],
  "streaming_slots": 3,
  "week": 20
}
```

| Field | Type | Description |
|---|---|---|
| `roster_data` | `[]Player` | All players currently on the manager's roster |
| `free_agent_data` | `[]Player` | Available free agents to consider adding |
| `streaming_slots` | `int` | How many roster spots to treat as streamable. The N lowest-`avg_points` healthy roster players are the initial streamers; everyone else is a core starter who is never dropped |
| `week` | `int` | Fantasy week number (1-indexed) in the loaded schedule. Must satisfy `1 <= week <= weeks` from `/healthz` |

`Player` fields: `name` (string, unique key), `avg_points` (float, the scalar being maximised), `team` (NBA tricode matching the schedule file), `valid_positions` (ordered most-restrictive-first from `PG, SG, SF, PF, C, G, F, UT1, UT2, UT3`), `injured` (bool; injured roster players are ignored, injured free agents are never picked up).

**Category leagues.** The optimizer is a scalar maximiser and has no notion of categories. For H2H-category leagues the backend computes a non-negative "category value" per player and sends it as `avg_points`; nothing in this service changes. A true multi-category objective belongs to a future rewrite.

#### Response body

```json
{
  "Lineup": [
    {
      "Day": 0,
      "Additions": [ { "Name": "Isaiah Hartenstein", "AvgPoints": 28.4, "Team": "OKC" } ],
      "Removals":  [ { "Name": "Vince Williams Jr.", "AvgPoints": 22.3, "Team": "MEM" } ],
      "Roster": {
        "PG": { "Name": "...", "AvgPoints": 45.1, "Team": "OKC" },
        "C":  { "Name": "Isaiah Hartenstein", "AvgPoints": 28.4, "Team": "OKC" }
      }
    }
  ],
  "Improvement": 42,
  "Timestamp": "3/9/2026 7:14PM",
  "Week": 20,
  "StreamingSlots": 3
}
```

| Field | Type | Description |
|---|---|---|
| `Lineup` | `[]SlimGene` | One entry per day of the week's game span (`Day` is 0-indexed). `Roster` maps position → player for everyone who plays that day; `Additions`/`Removals` are the transactions made that morning |
| `Improvement` | `int` | Projected gain over making no streaming moves at all |
| `Timestamp` | `string` | Server time when the response was generated |
| `Week` | `int` | Echo of the requested week |
| `StreamingSlots` | `int` | Echo of the requested streaming slots |

Response keys are PascalCase (Go field names; the response structs have no JSON tags).

#### Errors

| Status | Body | When |
|---|---|---|
| `400` | `Failed to decode request body` (text) | Request is not valid JSON for the shape above |
| `400` | `{"error": "unknown week", "week": 99, "weeks": 24}` | `week` is not in the loaded schedule. Without this check the optimizer would treat the week as a zero-day span and return an empty plan |
| `500` | `Failed to encode response` (text) | Should not happen |

CORS headers are set to `*` on all `/generate-lineup` responses and `OPTIONS` preflights are answered, so the service can be called without a proxy.

> **Note**: the retired v1 used a different endpoint path (`POST /optimize/`) and accepted ESPN league credentials to fetch player data itself. v2 deliberately does not: the caller (the backend) supplies `roster_data` and `free_agent_data`, so no league credentials ever reach this service.

---

## Algorithm Overview

The problem: given a roster with core players and `streaming_slots` streamable spots, and a pool of free agents, find the sequence of daily add/drop transactions that maximises total `avg_points` across the week's active lineups, subject to the weekly acquisition limit (one per day of the game span).

### Pre-computation: Optimal Slotting (`BaseTeam`)

Before any optimization runs, the service determines the **optimal position assignment** for core players for each day of the week:

1. Filter injured players out of the roster.
2. Sort the healthy roster by `avg_points`; the bottom `streaming_slots` players are the initial streamers, the rest are core.
3. For each day in the week's game span, collect which core players are playing (using the schedule JSON).
4. Run a **recursive backtracking** search (`FitPlayers`) that assigns players to roster slots most-restrictive-first (`PG → SG → SF → PF → G → F → C → UT1 → UT2 → UT3 → BE1 → BE2 → BE3`), leaving the maximum number of flexible slots open.
5. The open slots per day (`UnusedPositions`) are what streamers and pickups can fill.

### Parallel Dual-population Genetic Algorithm

1. **Initialization**: Two independent populations of 20 chromosomes are created concurrently. A chromosome is one gene per day; each gene holds that day's streamer roster, bench, additions and drops.
2. **Parallel evolution**: Both populations evolve for 10 generations simultaneously via `sync.WaitGroup`.
3. **Merge**: The two populations are combined (40 chromosomes) and evolved for another 10 generations.
4. **Selection**: The fittest chromosome whose total acquisitions respect the weekly limit is returned; core players are added back into its daily rosters.

Each generation applies:
- **Roulette-wheel selection** (parent 1) — fitness-proportional, with a power-law probability curve.
- **Tournament selection** (parent 2) — 3 tournaments of 5, picking the runner-up.
- **Crossover**: new players from each parent's daily transactions are sampled and inserted into the child.
- **Mutation** (20% probability): random add/drop perturbations on individual days.
- **Elitism**: the best chromosome from each generation is carried forward unchanged.

Fitness is the sum of `avg_points` over every rostered player on every day, scaled down by `1.3^(excess)` when acquisitions exceed the game span. Dropped players carry a short cooldown before they can be re-added.

### Schedule Data

Game schedules are static JSON files under `lineup-generation/v2/static/` (one per season, e.g. `schedule26-27.json`) with the structure:

```json
{
  "schedule": {
    "1": {
      "startDate": "10/20/2026",
      "endDate": "10/25/2026",
      "gameSpan": 6,
      "games": {
        "OKC": { "0": true, "2": true, "4": true },
        "LAL": { "0": true, "3": true, "5": true }
      }
    },
    "2": { "...": "..." }
  }
}
```

- Keys of `schedule` are fantasy week numbers as strings, numbered `1..N` (24 weeks in a normal season).
- `gameSpan` is the number of days in the week — 7 normally, 6 for the opening week, and 14 for the week that absorbs the All-Star break.
- `games` maps an NBA team tricode to the set of 0-indexed day offsets within the week on which that team plays. Only days with a game are present.

The file is regenerated every season by `backend/scripts/build_season_calendar.py --season {YYYY-YY} --fetch --check`, which writes it straight into `lineup-generation/v2/static/` (along with `backend/static` and `data-platform/static`). Commit it under `static/`, then bump the `SCHEDULE_FILE` default in `v2.go` and the `Dockerfile`. See `docs/SEASON_ROLLOVER.md` for the full rollover. The server will refuse to start on a missing file and `/healthz` reports which file is loaded.

---

## Running Tests

```bash
cd features/lineup-generation/v2
go test ./...          # everything, including the full genetic algorithm through the HTTP handler
go test -short ./...   # skips the end-to-end optimizer run
```

Paths to fixtures and schedules are resolved relative to the source tree (`testutil.RepoPath`), so the suite runs from any checkout. Tests load the newest schedule present under `static/` and derive week counts and game spans from it rather than hardcoding them, so they keep passing across season rollovers.

The suite covers:
- `tests/schedule_test.go` — schedule shape, `HasWeek`/`WeekCount`, and `LoadSchedule` errors (missing file, malformed JSON, no weeks) without clobbering a loaded schedule.
- `tests/base_team_test.go` — optimal slotting, unused positions, and streamer selection.
- `tests/gene_test.go`, `tests/chromosome_test.go` — streamer slotting, free agent insertion, chromosome bookkeeping invariants, and the slimmed response.
- `tests/population_test.go` — population initialization and evolution invariants.
- `v2_test.go` — `/healthz`, the `400` for unknown weeks, CORS preflight, and a full week-1 plan whose day count equals the week's game span.

---

## How It Fits Into Court Vision

```
Frontend (Next.js)
    │  POST /v1/lineup/generate  (with Clerk JWT)
    ▼
Backend (FastAPI, port 8000)
    │  Fetches roster + free agents from DB / ESPN / Yahoo
    │  POST /generate-lineup  (no auth; FEATURES_SERVER_ENDPOINT)
    ▼
Features Service (Go, port 8080)
    │  Runs genetic algorithm
    │  Returns optimized week plan
    ▼
Backend  →  Frontend
```

The backend acts as a proxy and data provider: it resolves roster data and free agent lists (and, for category leagues, computes the per-player value it sends as `avg_points`), then forwards them to this service. The features service is stateless — every request is self-contained. The relevant backend code is `backend/services/lineup_service.py` and `backend/services/optimize_service.py`.

In the frontend, lineup generation is triggered from the `/lineup-generation` page and the analytics terminal's `lineup-optimizer` panel.

---

## Version History

| Version | Algorithm | ESPN fetch | Endpoint | Status |
|---|---|---|---|---|
| v1 | Single-population GA, 75 chromosomes, 25 generations | Yes (direct ESPN API call per request) | `POST /optimize/` | Removed — recover from git history if ever needed |
| v2 | Dual-population GA, 2×20 chromosomes, 30 generations, concurrent | No (caller supplies data) | `POST /generate-lineup` | **Production** |
| v3 | — | — | — | Empty stub; a future rewrite may replace the scalar objective with a category-aware one |
