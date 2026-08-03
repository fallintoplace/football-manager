SELECT
    s.run_id,
    s.season,
    s.round,
    s.club_id,
    s.rank,
    s.played,
    s.points,
    s.goals_for,
    s.goals_against,
    s.goal_difference,
    round(if(m.home_id = s.club_id, m.home_xg, m.away_xg), 2) AS xg_for,
    round(if(m.home_id = s.club_id, m.away_xg, m.home_xg), 2) AS xg_against,
    round(if(m.home_id = s.club_id, m.home_possession, 100 - m.home_possession), 1) AS possession,
    if(m.home_id = s.club_id, m.home_shots, m.away_shots) AS shots_for,
    if(m.home_id = s.club_id, m.away_shots, m.home_shots) AS shots_against,
    if(m.home_id = s.club_id, m.home_pressure, m.away_pressure) AS pressure,
    if(m.home_id = s.club_id, m.home_territory, 100 - m.home_territory) AS territory
FROM (SELECT * FROM touchline_standings FINAL) AS s
LEFT JOIN touchline_matches_v2 AS m
    ON m.run_id = s.run_id
   AND m.season = s.season
   AND m.round = s.round
   AND (m.home_id = s.club_id OR m.away_id = s.club_id)
WHERE s.run_id = 'replace-with-run-id'
  AND s.club_id = 'replace-with-club-id'
ORDER BY s.round;

SELECT
    run_id,
    season,
    round,
    rank,
    club_id,
    points,
    won,
    drawn,
    lost,
    goals_for,
    goals_against,
    goal_difference,
    form
FROM touchline_standings FINAL
WHERE run_id = 'replace-with-run-id'
  AND round = (
      SELECT max(round)
      FROM touchline_standings
      WHERE run_id = 'replace-with-run-id'
  )
ORDER BY rank;

SELECT
    run_id,
    season,
    round,
    club_id,
    rank,
    points,
    goal_difference,
    goals_for
FROM touchline_standings FINAL
WHERE run_id = 'replace-with-run-id'
ORDER BY round, rank;

WITH latest_clubs AS
(
    SELECT
        run_id,
        season,
        club_id,
        argMax(short_name, ingested_at) AS club_name
    FROM touchline_clubs_v2
    WHERE run_id = 'replace-with-run-id'
    GROUP BY run_id, season, club_id
), club_xg AS
(
    SELECT
        m.run_id,
        m.season,
        m.round,
        m.home_id AS club_id,
        m.home_xg AS xg
    FROM touchline_matches_v2 AS m
    WHERE m.run_id = 'replace-with-run-id'
    UNION ALL
    SELECT
        m.run_id,
        m.season,
        m.round,
        m.away_id AS club_id,
        m.away_xg AS xg
    FROM touchline_matches_v2 AS m
    WHERE m.run_id = 'replace-with-run-id'
)
SELECT
    x.run_id,
    x.season,
    x.round,
    x.club_id,
    coalesce(any(c.club_name), x.club_id) AS club_name,
    round(avg(x.xg), 2) AS average_xg
FROM club_xg AS x
LEFT JOIN latest_clubs AS c
    ON c.run_id = x.run_id
   AND c.season = x.season
   AND c.club_id = x.club_id
GROUP BY x.run_id, x.season, x.round, x.club_id
ORDER BY x.round, club_name;

SELECT
    run_id,
    season,
    club_id,
    player_id,
    any(player_name) AS player_name,
    count() AS matches,
    round(avg(rating), 2) AS average_rating,
    round(max(rating), 2) AS best_rating
FROM touchline_player_ratings_v2
WHERE run_id = 'replace-with-run-id'
GROUP BY run_id, season, club_id, player_id
ORDER BY average_rating DESC, matches DESC
LIMIT 50;

SELECT
    run_id,
    season,
    player_id,
    any(player_name) AS player_name,
    any(position) AS position,
    argMin(overall, round) AS opening_overall,
    argMax(overall, round) AS closing_overall,
    closing_overall - opening_overall AS overall_change,
    argMin(form, round) AS opening_form,
    argMax(form, round) AS closing_form,
    argMin(fitness, round) AS opening_fitness,
    argMax(fitness, round) AS closing_fitness,
    argMax(average_rating, round) AS last_match_rating
FROM touchline_player_snapshots FINAL
WHERE run_id = 'replace-with-run-id'
GROUP BY run_id, season, player_id
ORDER BY overall_change DESC, closing_overall DESC
LIMIT 50;

SELECT
    run_id,
    season,
    round,
    event_type,
    team_id,
    count() AS events,
    round(sum(xg), 3) AS event_xg
FROM touchline_match_events_v2
WHERE run_id = 'replace-with-run-id'
GROUP BY run_id, season, round, event_type, team_id
ORDER BY round, events DESC;

SELECT
    run_id,
    season,
    round,
    match_id,
    team_id,
    player_id,
    any(player_name) AS player_name,
    intDiv(toInt32(player_x), 5) * 5 AS x_bin,
    intDiv(toInt32(player_y), 5) * 5 AS y_bin,
    count() AS intensity
FROM touchline_player_frames_v2
WHERE run_id = 'replace-with-run-id'
  AND team_id = 'arsenal'
GROUP BY run_id, season, round, match_id, team_id, player_id, x_bin, y_bin
ORDER BY round, match_id, intensity DESC;
