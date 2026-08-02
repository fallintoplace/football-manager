# Touchline analytics service

This service receives matchdays from the Touchline simulator, turns the event, replay, rating, and match streams into Arrow tables, and appends them to ClickHouse and Iceberg.

From this directory, run `go run .` after the local stack is available. From the project root, `docker compose up --build` starts ClickHouse, MinIO, the Iceberg REST catalog, and the service on port 8787.
