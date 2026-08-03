-- Replace both placeholders before running this ClickHouse career archive lab.
-- This is the SQL shape behind /api/analytics/season-comparison.

WITH run_registry AS
(
    SELECT
        run_id,
        any(career_id) AS career_id,
        any(season) AS season,
        argMax(status, updated_at) AS status,
        argMax(last_round, updated_at) AS last_round
    FROM touchline_runs_v2
    WHERE touchline_runs_v2.career_id = 'replace-with-career-id'
    GROUP BY run_id
), club_facts AS
(
    SELECT
        f.run_id,
        f.season,
        count() AS matches,
        sum(f.goals_for) AS goals_for,
        sum(f.goals_against) AS goals_against,
        sum(f.xg_for) AS xg_for,
        avg(f.possession) AS average_possession,
        avg(f.pressure) AS average_pressure,
        avg(f.press_wins) AS average_press_wins,
        avg(f.box_entries) AS average_box_entries
    FROM touchline_match_club_facts_v2 AS f
    INNER JOIN run_registry AS r ON r.run_id = f.run_id AND r.season = f.season
    WHERE f.club_id = 'replace-with-club-id'
    GROUP BY f.run_id, f.season
), opponents AS
(
    SELECT
        f.run_id,
        f.season,
        sum(f.xg_for) AS xg_against
    FROM touchline_match_club_facts_v2 AS f
    INNER JOIN run_registry AS r ON r.run_id = f.run_id AND r.season = f.season
    WHERE f.opponent_id = 'replace-with-club-id'
    GROUP BY f.run_id, f.season
), latest_standings AS
(
    SELECT
        s.run_id,
        s.season,
        argMax(s.rank, s.round) AS rank,
        argMax(s.points, s.round) AS points,
        argMax(s.won, s.round) AS won,
        argMax(s.drawn, s.round) AS drawn,
        argMax(s.lost, s.round) AS lost
    FROM (SELECT * FROM touchline_standings FINAL) AS s
    INNER JOIN run_registry AS r ON r.run_id = s.run_id AND r.season = s.season
    WHERE s.club_id = 'replace-with-club-id'
    GROUP BY s.run_id, s.season
)
SELECT
    r.season,
    r.status,
    r.last_round,
    ifNull(s.rank, 0) AS rank,
    ifNull(s.points, 0) AS points,
    ifNull(s.won, 0) AS won,
    ifNull(s.drawn, 0) AS drawn,
    ifNull(s.lost, 0) AS lost,
    ifNull(f.matches, 0) AS matches,
    round(ifNull(f.xg_for, 0), 2) AS xg_for,
    round(ifNull(o.xg_against, 0), 2) AS xg_against,
    round(ifNull(f.average_possession, 0), 1) AS average_possession,
    round(ifNull(f.average_pressure, 0), 1) AS average_pressure,
    round(ifNull(f.average_press_wins, 0), 1) AS average_press_wins,
    round(ifNull(f.average_box_entries, 0), 1) AS average_box_entries
FROM run_registry AS r
LEFT JOIN latest_standings AS s ON s.run_id = r.run_id AND s.season = r.season
LEFT JOIN club_facts AS f ON f.run_id = r.run_id AND f.season = r.season
LEFT JOIN opponents AS o ON o.run_id = r.run_id AND o.season = r.season
ORDER BY r.season;
