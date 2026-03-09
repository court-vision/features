# Court Vision — Features Service

Go services for compute-intensive features in the [Court Vision](https://github.com/jameskendrick/cv) fantasy basketball analytics platform. The primary service is **lineup generation**: given a manager's current roster and the available free agent pool, it determines the optimal sequence of daily add/drop moves to maximize projected fantasy points scored over a given week.

The service runs an HTTP server on **port 8080**.

---

## Tech Stack

- **Language**: Go 1.22+ (v3 targets Go 1.23)
- **HTTP**: `net/http` standard library — no external framework
- **Concurrency**: `sync.WaitGroup` + goroutines for parallel population evolution (v2)
- **Data**: Static NBA schedule JSON bundled at build time
- **Container**: Multi-stage Docker build (`golang:1.22.4` builder → `scratch` final image)

---

## Repository Structure

```
features/
├── main.go                        # Stub entry point (imports commented out)
├── go.mod                         # Root module (module: features, go 1.22.4)
├── Dockerfile                     # Builds and runs lineup-generation/v2
└── lineup-generation/
    ├── v1/                        # Original genetic algorithm (fetches ESPN data directly)
    │   ├── main.go
    │   ├── go.mod
    │   ├── functions/             # evolution_funcs, initpop_funcs, pregen_funcs, structs, utils
    │   ├── static/                # schedule2024-2025.json, schedule2025-2026.json
    │   └── tests/
    ├── v2/                        # Refactored genetic algorithm (caller-supplied player data)
    │   ├── v2.go                  # Entry point + OptimizeStreaming()
    │   ├── go.mod
    │   ├── data/                  # Schedule loader, Player/DroppedPlayer types
    │   ├── population/            # EvolutionManager, Chromosome, Gene
    │   ├── team/                  # BaseTeam (pre-computed slotting metadata)
    │   ├── utils/                 # ReqBody, Response, Bench, helpers
    │   ├── resources/             # Mock JSON fixtures for testing
    │   ├── static/                # schedule25-26.json (bundled in Docker image)
    │   └── tests/
    └── v3/                        # Beam-search rewrite (in progress)
        ├── main.go
        ├── go.mod
        ├── run_tests.sh
        ├── helpers/               # http.go, player.go, schedule.go, setup_state.go, state.go
        ├── static/                # schedule2025-2026.json
        └── tests/                 # schedule_test.go, state_test.go
```

**v2 is the version deployed in production** (see `Dockerfile`). v3 is under active development.

---

## Setup & Running

### Prerequisites

- Go 1.22 or later (`go version`)

### Run locally (v2 — current production version)

```bash
cd features/lineup-generation/v2
go run .
# Server started on port 8080
```

### Run locally (v3 — in development)

```bash
cd features/lineup-generation/v3
go run .
# Server started on port 8080
```

### Build and run with Docker

The `Dockerfile` at the repo root builds v2:

```bash
# From features/
docker build -t cv-features .
docker run -p 8080:8080 cv-features
```

The Docker build produces a minimal `scratch`-based image (~7 MB). It copies the compiled binary and the 2025–26 schedule JSON; no runtime dependencies.

---

## API

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
      "valid_positions": ["PF", "C", "F"],
      "injured": false
    }
  ],
  "free_agent_data": [
    {
      "name": "Isaiah Hartenstein",
      "avg_points": 28.4,
      "team": "OKC",
      "valid_positions": ["C", "PF"],
      "injured": false
    }
  ],
  "threshold": 30.0,
  "week": 20
}
```

| Field | Type | Description |
|---|---|---|
| `roster_data` | `[]Player` | All players currently on the manager's roster |
| `free_agent_data` | `[]Player` | Available free agents to consider adding |
| `threshold` | `float64` | Average fantasy points per game cutoff. Players **above** this are treated as core (non-streamable) starters; players **at or below** it are candidates for streaming moves |
| `week` | `int` | Fantasy week number (1-indexed). Used to look up the NBA game schedule for that week |

#### Response body

```json
{
  "Lineup": [
    {
      "Day": 0,
      "Additions": [],
      "Removals": [],
      "Roster": {
        "PG": { "Name": "...", "AvgPoints": 45.1, "Team": "OKC" },
        "SG": { "Name": "...", "AvgPoints": 60.0, "Team": "PHX" }
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
| `Lineup` | `[]SlimGene` | One entry per day in the week's game span. Each entry contains the day's roster map, and any additions/removals vs. the previous day |
| `Improvement` | `int` | Projected fantasy point gain over making no streaming moves at all |
| `Timestamp` | `string` | Server time when the response was generated |
| `Week` | `int` | Echo of the requested week |
| `StreamingSlots` | `int` | Number of streaming slots used (v2); `Threshold` (v3) |

CORS headers are set to `*` on all responses, so the backend can call this service without a proxy.

> **Note**: v1 used a different endpoint path (`POST /optimize/`) and accepted ESPN league credentials (`league_id`, `espn_s2`, `swid`, `team_name`) to fetch player data itself. v2 and v3 require the caller (the backend) to supply `roster_data` and `free_agent_data` directly.

---

## Algorithm Overview

The core problem: given a roster with some "core" high-value players and some low-value "streamable" slots, and a pool of available free agents, find the sequence of daily add/drop transactions that maximizes total fantasy points scored across the week — subject to the weekly acquisition limit.

### Pre-computation: Optimal Slotting (`SetupStateMetadata` / `BaseTeam`)

Before any optimization runs, the service determines the **optimal position assignment** for core (non-streamable) players for each day of the week:

1. Filter injured players out of the roster.
2. Split the roster on `threshold`: players above it are "core"; at or below are "streamable".
3. For each day in the week's game span, collect which core players are playing (using the NBA schedule JSON).
4. Run a **recursive backtracking** algorithm (`FitPlayers`) that assigns players to roster slots using a most-restrictive-first ordering (`PG → SG → SF → PF → G → F → C → UT1 → UT2 → UT3 → BE1 → BE2 → BE3`). This finds the slotting that leaves the maximum number of flexible slots (utility/bench) open for streamers.
5. The remaining open slots become the **streaming slots** that the optimizer fills each day.

### v1 — Single-population Genetic Algorithm

- Population of 75 chromosomes, each representing a full week of add/drop decisions.
- 25 generations of selection → crossover → mutation.
- Fetched player data directly from the ESPN Fantasy API on each request (required `espn_s2` / `swid` cookies).
- Sequential evolution, no concurrency.

### v2 — Parallel Dual-population Genetic Algorithm (production)

The genetic algorithm is restructured for performance with goroutine-based parallelism:

1. **Initialization**: Two independent populations of 20 chromosomes are created concurrently (each chromosome generated in its own goroutine).
2. **Parallel evolution**: Both populations evolve for 10 generations simultaneously via `sync.WaitGroup`.
3. **Merge**: The two populations are combined (40 chromosomes total) and evolved for another 10 generations.
4. **Selection**: The best chromosome that respects the weekly acquisition limit is selected.

Each generation applies:
- **Roulette-wheel selection** (parent 1) — fitness-proportional, with a power-law probability curve.
- **Tournament selection** (parent 2) — 3 tournaments of 5, picking the runner-up.
- **Crossover**: genes from both parents are merged; new players from each parent's daily transactions are sampled and inserted into the child.
- **Mutation** (20% probability): random add/drop perturbations on individual days.
- **Elitism**: the best chromosome from each generation is carried forward unchanged.

Player data is supplied by the caller; the service no longer calls ESPN directly.

### v3 — Beam Search (in development)

v3 is a ground-up rewrite replacing the genetic algorithm with a **beam search** approach:

- `State` represents the current lineup plan: a list of `Lineup` objects (one per day), the set of current streamers, remaining acquisition budget, and dropped player cooldowns.
- `SetupStateMetadata` pre-computes optimal slotting and identifies unused positions per day — the slots available for streamers.
- The search will explore the space of daily pickup decisions by maintaining a beam of the top-K states, expanding forward day by day.
- The `main.go` endpoint scaffolding is in place; the beam search traversal is the remaining implementation work (currently returns an empty lineup).

### Schedule Data

Game schedules are stored as static JSON files (`static/schedule2025-2026.json`) with the structure:

```json
{
  "20": {
    "startDate": "03/09/2026",
    "endDate": "03/15/2026",
    "gameSpan": 7,
    "games": {
      "LAL": [0, 2, 4, 6],
      "GSW": [1, 3, 5],
      ...
    }
  }
}
```

Each key is the fantasy week number. `games` maps a team abbreviation to a list of 0-indexed day offsets within the week on which that team plays. `gameSpan` is the total number of days in the week (typically 6 or 7).

---

## Running Tests

### v3 (current)

```bash
cd features/lineup-generation/v3
./run_tests.sh
```

`run_tests.sh` runs four passes in sequence: verbose output, coverage report, and benchmark suite — all against `./tests/`.

You can also run individual passes:

```bash
go test -v ./tests/                    # verbose
go test -v -cover ./tests/             # with coverage
go test -v -bench=. ./tests/           # with benchmarks
```

The test suite covers:
- `TestLoadWeekSchedule` — parses a temp schedule JSON, validates week data, and tests with the real `static/schedule2025-2026.json`.
- `TestInitWeekScheduleWithInvalidWeek/File/JSON` — error path handling.
- `BenchmarkInitWeekSchedule` — schedule loading throughput.
- `TestInitSetupStateMetadata` — verifies the optimal slotting pre-computation with mock roster data.
- `TestInitState` — verifies beam search state initialization including streamer slotting.

### v2

```bash
cd features/lineup-generation/v2
go test -v ./tests/
```

Test files cover `BaseTeam`, `Chromosome`, `Gene`, `Population`, `Schedule`, and utility helpers.

---

## How It Fits Into Court Vision

```
Frontend (Next.js)
    │  POST /v1/lineup/generate  (with Clerk JWT)
    ▼
Backend (FastAPI, port 8000)
    │  Fetches roster + free agents from DB / ESPN API
    │  POST /generate-lineup  (no auth)
    ▼
Features Service (Go, port 8080)
    │  Runs genetic algorithm
    │  Returns optimized week plan
    ▼
Backend  →  Frontend
```

The backend acts as a proxy and data provider: it resolves ESPN roster data and free agent lists, then forwards them to this service. The features service is stateless — every request is self-contained. The result is returned to the frontend via the backend's `/v1/lineup/generate` route.

In the frontend, lineup generation is triggered from the `/lineup-generation` page and the analytics terminal's `lineup-optimizer` panel. The relevant backend service is `backend/services/lineup_service.py`.

---

## Version History

| Version | Algorithm | ESPN fetch | Endpoint | Status |
|---|---|---|---|---|
| v1 | Single-population GA, 75 chromosomes, 25 generations | Yes (direct ESPN API call per request) | `POST /optimize/` | Archived |
| v2 | Dual-population GA, 2×20 chromosomes, 30 generations, concurrent | No (caller supplies data) | `POST /generate-lineup` | **Production** |
| v3 | Beam search (stateful day-by-day search) | No (caller supplies data) | `POST /generate-lineup` | In development |
