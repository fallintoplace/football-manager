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
```

## Next Systems

- Transfer market and contracts
- Scouting reports with incomplete information
- Injuries, suspensions, and rotation pressure
- Player relationship events
- More explainable tactical counters between clubs
- Save slots and named careers
