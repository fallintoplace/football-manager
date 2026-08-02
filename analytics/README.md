# Touchline analytics service

This service receives matchdays from the Touchline simulator, turns the match, event, replay, rating, standings, and player-snapshot streams into Arrow tables, and appends them to ClickHouse and Iceberg.

From this directory, run `go run .` after the local stack is available. From the project root, `docker compose up --build` starts ClickHouse, MinIO, the Iceberg REST catalog, and the service on port 8787.

ClickHouse 26.2 exposes ClickStack at `http://localhost:8123/clickstack`. Touchline creates one stable `run_id` per seed and season, shows it in the Analytics Lab, and sends it with every matchday. Replace `s7321-season-1` in `clickstack-queries.sql` with the displayed run ID, then use the queries as ClickStack charts for the final table, points race, xG trend, player development, event volume, and replay density. The custom pitch heatmap remains in Touchline, where the replay canvas can render spatial bins directly.

ClickHouse de-duplicates a retried matchday within the same run by `run_id` and `match_id`. Iceberg keeps the long-form Arrow history, while run-aware standings and player snapshot tables support season-level comparisons without mixing careers.
