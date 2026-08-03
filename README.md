# Touchline

A fast football management prototype focused on visible tactical consequences, squad mood, and a compact season loop.

## Current Slice

- 20-club Premier League snapshot with real player names, positions, and club badges
- Club switching, board trust, fan mood, budget, standings, and fixtures
- Matchday Hub with opponent brief, readiness checks, tactical plan, and team sheet
- Manual starting XI selection with legal formation validation
- One-click recommended XI
- One-click full-season simulation that fills the remaining 38-match league schedule and lands on the final table
- Tactical controls for formation, mentality, pressing, tempo, and defensive line
- Training focus that affects player attributes, morale, and fitness
- Phase-based match simulation with buildup, midfield duels, final-third entries, box entries, counters, xG, pressure, and tactical reports
- Spatial replay trace with real starters, ball position, player intent, and phase-aware movement
- Animated 2D canvas pitch on the match view
- Captain selection, player form, morale, fitness, value, and wages
- FIFA/Football Manager-style player model with 0–99 OVR, potential, and position-specific technical, mental, physical, defensive, attacking, and goalkeeper attributes
- Local save/reset through browser storage
- Season rollover after the fixture list completes
- Analytics Lab with selected-club match, event, replay-frame, and player-rating aggregates
- Season Command Center with ClickHouse-backed round charts, historical league-table slider, match replay links, and player development deltas
- Run-aware season sync with a stable career/season ID, progress feedback, standings snapshots, and player development snapshots for ClickStack exploration

The 2026/27 Premier League roster snapshot is pinned in `src/game/premierLeagueData.ts`. Player names, positions, age, availability, and recent form inputs come from the Fantasy Premier League bootstrap feed captured on 2026-08-02; badge images use the API-Football crest CDN. OVR and potential are derived game ratings rather than official EA or Football Manager ratings. Match events, player ratings, and tactical outcomes remain simulated so the local demo is deterministic and can feed the ClickHouse and Iceberg pipeline without live-match credentials.

## Analytics Stack

The optional local data stack receives each simulated matchday from the browser. The analytics service preserves match, event, player-rating, spatial replay, standings, and player-development streams. It uses Arrow-shaped batches for Iceberg writes, ClickHouse for low-latency aggregates, a run-aware materialized-view lab for club-round metrics, Iceberg REST for table metadata, and MinIO for the warehouse.

Start it from the project root:

```bash
docker compose up --build
```

The service listens on `http://localhost:8787`. The game still runs without the stack; the Analytics Lab will show an offline state until the service is available.

Analytics contract v2 also records one run registry row per season sync, one club-relative fact row per match, and one player-match fact row per rated starter. Query `http://localhost:8787/api/analytics/runs` to inspect run progress, or `http://localhost:8787/api/analytics/season-comparison?career_id=...&club_id=...` to compare seasons in one career; the ClickHouse tables preserve tactical counters, tactic settings, player minutes, shots, goals, and event xG for the upcoming season-comparison and what-if labs.

The Analytics Lab also includes a tactical matchup table. Query `http://localhost:8787/api/analytics/tactical-matchup?run_id=...&club_id=...` to compare every opponent, or add `opponent_id` for one matchup. This is a self-join over the club-relative facts, which makes the simulator's tactical ledger inspectable in ClickHouse and ClickStack.

The Real Data Lab keeps historical public event data isolated from simulated career data. Run `node scripts/import-statsbomb.mjs --limit=20` to import a deterministic 20-match slice from StatsBomb Open Data, or use `--match-id=3754217` for the Chelsea-Arsenal sample. The Analytics tab exposes the imported catalog through a match selector, shot map, pass links, and player action profiles; `analytics/real-data-lab.sql` contains the equivalent ClickStack queries. Real rows retain provider, season, match, player, team, possession, coordinate-system, and source-event provenance.

The World Cup 2026 Lab adds a second real-data-shaped workload for learning ClickHouse and ClickStack. Run `node scripts/import-worldcup2026.mjs` to ingest the 104-match, 48-team openfootball snapshot. The Analytics tab then shows the tournament KPIs, group standings, golden boot, goal timing, venue footprint, final, and knockout bracket. The backend stores separate match and goal facts in ClickHouse and Iceberg; `analytics/worldcup-2026-lab.sql` contains the matching SQL exercises.

Every newly simulated match also emits a structured action feed with possession/sequence IDs, player IDs, pass recipients, coordinates, outcomes, and qualifiers. ClickHouse stores it in `touchline_match_actions_v2`; `analytics/action-lab.sql` turns it into pass networks, possession chains, shot context, and spatial action bins.

The Analytics Lab's Action intelligence panel reads `GET /api/analytics/action-insights` to turn those actions into pass completion, progressive passing, possession volume, shot xG, action mix, passing connections, and a manager-facing analyst note.

It also maps each involved player to a primary role from their action evidence, making the data useful for lineup decisions and future recruitment/scouting systems.

The Analytics Lab also reads `http://localhost:8787/api/analytics/iceberg/history?table=player_match_facts` and shows the Iceberg snapshot count, so the local stack now exposes both the hot ClickHouse path and the lakehouse history path.

ClickHouse 26.2 includes the embedded ClickStack UI. After a full season sync, open `http://localhost:8123/clickstack` from the Analytics Lab, copy the displayed career/season run ID into `analytics/clickstack-queries.sql`, and build the final-table, season-story, points-race, xG, player-development, event-volume, and replay-density charts over that isolated run. The frontend reads the same round-level data from `/api/analytics/timeline`; the custom pitch heatmap remains in Touchline.

For the ClickHouse learning lab, replace the same placeholders in `analytics/performance-lab.sql`. Run it with `docker compose exec -T clickhouse clickhouse-client --password clickhouse --multiquery < analytics/performance-lab.sql` after backfilling `touchline_club_round_metrics` for the selected run. The lab compares raw match facts with the materialized-view path, records query-log metrics, shows index pruning, and inspects parts, merges, and data types.

The scale experiment is run with `analytics/scale-lab.sql`; it replaces only `scale-lab-*` rows and generates 20 copies of the selected season. `analytics/clickstack-alerts.sql` contains the xG-underperformance alert query. In local ClickStack, the dashboard supports a global SQL `WHERE` filter, so the saved Touchline Season Command Center uses `run_id` and `club_id` without changing tile SQL; the local banner notes that alert scheduling is not included in this build.

## Run

```bash
npm install
npm run dev
```

Then open the local URL printed by Vite.

## Checks

```bash
npm run lint
npm run build
(cd analytics && go test ./...)
```

## Next Systems

- Transfer market and contracts
- Scouting reports with incomplete information
- Injuries, suspensions, and rotation pressure
- Player relationship events
- More explainable tactical counters between clubs
- Save slots and named careers
