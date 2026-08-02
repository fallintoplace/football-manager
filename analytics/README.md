# Touchline analytics service

This service receives matchdays from the Touchline simulator, turns the match, event, replay, rating, standings, and player-snapshot streams into Arrow tables, and appends them to ClickHouse and Iceberg.

From this directory, run `go run .` after the local stack is available. From the project root, `docker compose up --build` starts ClickHouse, MinIO, the Iceberg REST catalog, and the service on port 8787.

ClickHouse 26.2 exposes ClickStack at `http://localhost:8123/clickstack`. Touchline creates one stable `run_id` per career and season, shows it in the Analytics Lab, and sends it with every matchday. Replace `replace-with-run-id` in `clickstack-queries.sql` with the displayed run ID, then use the queries as ClickStack charts for the final table, points race, xG trend, player development, event volume, and replay density. The custom pitch heatmap remains in Touchline, where the replay canvas can render spatial bins directly.

ClickHouse repairs incomplete match facts before marking a match ingested. Iceberg writes are run-aware and overwrite the `(run_id, season, round)` slice on retry, so a failed multi-table append can be replayed without duplicating a round. The `_v2` ClickHouse fact tables keep the old local schema isolated while new data uses run-aware sorting keys.
