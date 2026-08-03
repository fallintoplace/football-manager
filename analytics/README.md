# Touchline analytics service

This service receives matchdays from the Touchline simulator, turns the match, event, replay, rating, standings, and player-snapshot streams into Arrow tables, and appends them to ClickHouse and Iceberg.

From this directory, run `go run .` after the local stack is available. From the project root, `docker compose up --build` starts ClickHouse, MinIO, the Iceberg REST catalog, and the service on port 8787.

ClickHouse 26.2 exposes ClickStack at `http://localhost:8123/clickstack`. Touchline creates one stable `run_id` per career and season, shows it in the Analytics Lab, and sends it with every matchday. Replace `replace-with-run-id` and `replace-with-club-id` in `clickstack-queries.sql` with the displayed values, then use the queries as ClickStack charts for the season story, final table, points race, xG trend, player development, event volume, and replay density. The frontend reads the same round-level dataset from `/api/analytics/timeline`. The custom pitch heatmap remains in Touchline, where the replay canvas can render spatial bins directly.

The ClickHouse learning lab is in `performance-lab.sql`. The analytics schema creates `touchline_club_round_metrics` with a `SummingMergeTree` and `touchline_club_round_metrics_mv`; the view handles new match inserts, while the lab uses an explicit run-scoped backfill for existing facts. The file compares raw and materialized results, captures `system.query_log` metrics, runs `EXPLAIN indexes = 1`, and inspects `system.parts`, `system.merges`, and `system.columns`.

`scale-lab.sql` safely replaces only the `scale-lab-*` slice with 20 deterministic copies of one stored season, then measures insert cost, raw versus materialized reads, parts, merges, and column types. `clickstack-alerts.sql` is the xG-underperformance SQL for ClickStack alerting; the local 26.2 UI currently advertises alerts as unavailable, so the query is kept version-independent and ready to paste into a newer ClickStack deployment.

ClickHouse repairs incomplete match facts before marking a match ingested. Iceberg writes are run-aware and overwrite the `(run_id, season, round)` slice on retry, so a failed multi-table append can be replayed without duplicating a round. The `_v2` ClickHouse fact tables keep the old local schema isolated while new data uses run-aware sorting keys.
