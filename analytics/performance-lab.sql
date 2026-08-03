EXPLAIN indexes = 1
SELECT
    round,
    home_id,
    away_id,
    home_xg,
    away_xg
FROM touchline_matches_v2
WHERE run_id = 'replace-with-run-id'
  AND (home_id = 'replace-with-club-id' OR away_id = 'replace-with-club-id')
ORDER BY round;

EXPLAIN indexes = 1
SELECT
    round,
    sum(goals_for) AS goals_for,
    sum(goals_against) AS goals_against,
    round(sum(xg_for), 3) AS xg_for,
    round(sum(xg_against), 3) AS xg_against,
    round(avg(possession), 1) AS possession
FROM touchline_club_round_metrics
WHERE run_id = 'replace-with-run-id'
  AND club_id = 'replace-with-club-id'
GROUP BY round
ORDER BY round;

SET log_queries = 1;
SET log_comment = 'touchline-raw-season-baseline';
SELECT
    round,
    sum(if(home_id = 'replace-with-club-id', home_goals, away_goals)) AS goals_for,
    sum(if(home_id = 'replace-with-club-id', away_goals, home_goals)) AS goals_against,
    round(sum(if(home_id = 'replace-with-club-id', toFloat64(home_xg), toFloat64(away_xg))), 3) AS xg_for,
    round(sum(if(home_id = 'replace-with-club-id', toFloat64(away_xg), toFloat64(home_xg))), 3) AS xg_against,
    round(avg(if(home_id = 'replace-with-club-id', toFloat64(home_possession), 100.0 - toFloat64(home_possession))), 1) AS possession
FROM touchline_matches_v2
WHERE run_id = 'replace-with-run-id'
  AND (home_id = 'replace-with-club-id' OR away_id = 'replace-with-club-id')
GROUP BY round
ORDER BY round;

SYSTEM FLUSH LOGS;
SET log_comment = 'touchline-materialized-season-baseline';
SELECT
    round,
    sum(goals_for) AS goals_for,
    sum(goals_against) AS goals_against,
    round(sum(xg_for), 3) AS xg_for,
    round(sum(xg_against), 3) AS xg_against,
    round(avg(possession), 1) AS possession
FROM touchline_club_round_metrics
WHERE run_id = 'replace-with-run-id'
  AND club_id = 'replace-with-club-id'
GROUP BY round
ORDER BY round;

SYSTEM FLUSH LOGS;
SET log_comment = 'touchline-performance-report';
SELECT
    log_comment,
    query_duration_ms,
    read_rows,
    read_bytes,
    memory_usage
FROM system.query_log
WHERE type = 'QueryFinish'
  AND log_comment IN ('touchline-raw-season-baseline', 'touchline-materialized-season-baseline')
ORDER BY event_time_microseconds DESC
LIMIT 10;

WITH raw AS
(
    SELECT
        round,
        sum(if(home_id = 'replace-with-club-id', home_goals, away_goals)) AS goals_for,
        sum(if(home_id = 'replace-with-club-id', away_goals, home_goals)) AS goals_against,
        round(sum(if(home_id = 'replace-with-club-id', toFloat64(home_xg), toFloat64(away_xg))), 3) AS xg_for,
        round(sum(if(home_id = 'replace-with-club-id', toFloat64(away_xg), toFloat64(home_xg))), 3) AS xg_against,
        round(avg(if(home_id = 'replace-with-club-id', toFloat64(home_possession), 100.0 - toFloat64(home_possession))), 1) AS possession
    FROM touchline_matches_v2
    WHERE run_id = 'replace-with-run-id'
      AND (home_id = 'replace-with-club-id' OR away_id = 'replace-with-club-id')
    GROUP BY round
), materialized AS
(
    SELECT
        round,
        sum(goals_for) AS goals_for,
        sum(goals_against) AS goals_against,
        round(sum(xg_for), 3) AS xg_for,
        round(sum(xg_against), 3) AS xg_against,
        round(avg(possession), 1) AS possession
    FROM touchline_club_round_metrics
    WHERE run_id = 'replace-with-run-id'
      AND club_id = 'replace-with-club-id'
    GROUP BY round
)
SELECT
    count() AS compared_rounds,
    countIf(raw.goals_for != materialized.goals_for
        OR raw.goals_against != materialized.goals_against
        OR raw.xg_for != materialized.xg_for
        OR raw.xg_against != materialized.xg_against
        OR raw.possession != materialized.possession) AS mismatched_rounds,
    max(abs(raw.xg_for - materialized.xg_for)) AS max_xg_delta
FROM raw
INNER JOIN materialized USING (round);

SELECT
    table,
    countIf(active) AS active_parts,
    sumIf(rows, active) AS active_rows,
    formatReadableSize(sumIf(bytes_on_disk, active)) AS active_size
FROM system.parts
WHERE database = 'default'
  AND table IN ('touchline_matches_v2', 'touchline_club_round_metrics', 'touchline_standings')
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
    type
FROM system.columns
WHERE database = 'default'
  AND table IN ('touchline_matches_v2', 'touchline_club_round_metrics', 'touchline_standings')
ORDER BY table, name;
