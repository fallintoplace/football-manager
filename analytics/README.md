# Touchline analytics service

This service receives matchdays from the Touchline simulator, turns the event, replay, rating, and match streams into Arrow tables, and appends them to ClickHouse and Iceberg.

From this directory, run `go run .` after the local stack is available. From the project root, `docker compose up --build` starts ClickHouse, MinIO, the Iceberg REST catalog, and the service on port 8787.

ClickHouse 26.2 exposes ClickStack at `http://localhost:8123/clickstack`. The starter queries in `clickstack-queries.sql` cover timestamped xG, event volume, player form, and replay-frame density. Use the `match_id`, `team_id`, and `player_id` columns as dashboard filters. The custom pitch heatmap remains in the Touchline UI, where the replay canvas can render spatial bins directly.
