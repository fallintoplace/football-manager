SET mutations_sync = 2;

ALTER TABLE touchline_club_round_metrics
DELETE WHERE startsWith(run_id, 'scale-lab-');

ALTER TABLE touchline_matches_v2
DELETE WHERE startsWith(run_id, 'scale-lab-');

SET log_queries = 1;
SET log_comment = 'touchline-scale-insert-20-seasons';

INSERT INTO touchline_matches_v2 (
    run_id,
    match_id,
    season,
    round,
    home_id,
    away_id,
    home_goals,
    away_goals,
    home_xg,
    away_xg,
    home_shots,
    away_shots,
    home_shots_on_target,
    away_shots_on_target,
    home_possession,
    home_pressure,
    away_pressure,
    home_territory,
    result
)
SELECT
    concat('scale-lab-', toString(n.number), ':season:1'),
    concat('scale-lab-', toString(n.number), ':', m.match_id),
    m.season,
    m.round,
    m.home_id,
    m.away_id,
    m.home_goals,
    m.away_goals,
    m.home_xg,
    m.away_xg,
    m.home_shots,
    m.away_shots,
    m.home_shots_on_target,
    m.away_shots_on_target,
    m.home_possession,
    m.home_pressure,
    m.away_pressure,
    m.home_territory,
    m.result
FROM touchline_matches_v2 AS m
CROSS JOIN numbers(20) AS n
WHERE m.run_id = 'replace-with-run-id';

SYSTEM FLUSH LOGS;

EXPLAIN indexes = 1
SELECT round, count()
FROM touchline_matches_v2
WHERE run_id = 'scale-lab-19:season:1'
GROUP BY round
ORDER BY round;

SET log_comment = 'touchline-scale-raw-read';
SELECT
    round,
    count() AS matches,
    sum(home_goals + away_goals) AS goals,
    round(sum(home_xg + away_xg), 2) AS xg
FROM touchline_matches_v2
WHERE run_id = 'scale-lab-19:season:1'
GROUP BY round
ORDER BY round;

SET log_comment = 'touchline-scale-mv-read';
SELECT
    round,
    sum(matches) AS matches,
    sum(goals_for + goals_against) AS goals,
    round(sum(xg_for + xg_against), 2) AS xg
FROM touchline_club_round_metrics
WHERE run_id = 'scale-lab-19:season:1'
GROUP BY round
ORDER BY round;

SET log_comment = 'touchline-scale-mv-final';
SELECT
    'without_final' AS read_path,
    count() AS rows,
    sum(matches) AS matches
FROM touchline_club_round_metrics
WHERE run_id = 'scale-lab-19:season:1'
UNION ALL
SELECT
    'with_final' AS read_path,
    count() AS rows,
    sum(matches) AS matches
FROM touchline_club_round_metrics FINAL
WHERE run_id = 'scale-lab-19:season:1';

SET log_comment = 'touchline-scale-all-runs';
SELECT
    uniqExact(run_id) AS generated_runs,
    count() AS generated_matches
FROM touchline_matches_v2
WHERE startsWith(run_id, 'scale-lab-');

SYSTEM FLUSH LOGS;

SELECT
    log_comment,
    query_duration_ms,
    read_rows,
    read_bytes,
    written_rows,
    written_bytes,
    memory_usage
FROM system.query_log
WHERE type = 'QueryFinish'
  AND log_comment IN (
      'touchline-scale-insert-20-seasons',
      'touchline-scale-raw-read',
      'touchline-scale-mv-read',
      'touchline-scale-mv-final',
      'touchline-scale-all-runs'
  )
ORDER BY event_time_microseconds DESC;

SELECT
    table,
    countIf(active) AS active_parts,
    sumIf(rows, active) AS active_rows,
    formatReadableSize(sumIf(bytes_on_disk, active)) AS active_size
FROM system.parts
WHERE database = 'default'
  AND table IN ('touchline_matches_v2', 'touchline_club_round_metrics')
GROUP BY table
ORDER BY table;

SELECT
    database,
    table,
    elapsed,
    progress,
    num_parts,
    result_part_name
FROM system.merges
WHERE database = 'default'
ORDER BY elapsed DESC;

SELECT
    table,
    name,
    type,
    default_kind
FROM system.columns
WHERE database = 'default'
  AND table IN ('touchline_matches_v2', 'touchline_club_round_metrics')
ORDER BY table, name;
