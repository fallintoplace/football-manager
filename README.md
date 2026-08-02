# Touchline

A fast fictional football management prototype focused on visible tactical consequences, squad mood, and a compact season loop.

## Current Slice

- 12-club fictional league with generated squads
- Club switching, board trust, fan mood, budget, standings, and fixtures
- Matchday Hub with opponent brief, readiness checks, tactical plan, and team sheet
- Manual starting XI selection with legal formation validation
- One-click recommended XI
- Tactical controls for formation, mentality, pressing, tempo, and defensive line
- Training focus that affects player attributes, morale, and fitness
- Phase-based match simulation with buildup, midfield duels, final-third entries, box entries, counters, xG, pressure, and tactical reports
- Spatial replay trace with real starters, ball position, player intent, and phase-aware movement
- Animated 2D canvas pitch on the match view
- Captain selection, player form, morale, fitness, value, and wages
- Local save/reset through browser storage
- Season rollover after the fixture list completes
- Analytics Lab with selected-club match, event, replay-frame, and player-rating aggregates

## Analytics Stack

The optional local data stack receives each simulated matchday from the browser. The analytics service preserves four fact streams: matches, events, player ratings, and spatial replay frames. It uses Arrow-shaped batches for Iceberg writes, ClickHouse for low-latency aggregates, Iceberg REST for table metadata, and MinIO for the warehouse.

Start it from the project root:

```bash
docker compose up --build
```

The service listens on `http://localhost:8787`. The game still runs without the stack; the Analytics Lab will show an offline state until the service is available.

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
