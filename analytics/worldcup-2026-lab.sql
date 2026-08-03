-- World Cup 2026 ClickHouse lab. The importer keeps match and goal facts separate
-- so the same tournament can power catalog, bracket, scorer, and timing queries.

-- Tournament footprint. Keep the team UNION separate so match totals are not
-- multiplied by arrayJoin.
SELECT
    (SELECT count()
     FROM touchline_worldcup_2026_matches_v1
     WHERE source = 'openfootball' AND tournament = 'world-cup-2026') AS matches,
    (SELECT uniqExact(team)
     FROM
     (
         SELECT home_team AS team
         FROM touchline_worldcup_2026_matches_v1
         WHERE source = 'openfootball' AND tournament = 'world-cup-2026'
         UNION ALL
         SELECT away_team AS team
         FROM touchline_worldcup_2026_matches_v1
         WHERE source = 'openfootball' AND tournament = 'world-cup-2026'
     )) AS teams,
    (SELECT sum(home_goals + away_goals)
     FROM touchline_worldcup_2026_matches_v1
     WHERE source = 'openfootball' AND tournament = 'world-cup-2026') AS goals,
    (SELECT uniqExact(venue)
     FROM touchline_worldcup_2026_matches_v1
     WHERE source = 'openfootball' AND tournament = 'world-cup-2026') AS venues,
    (SELECT avg(home_goals + away_goals)
     FROM touchline_worldcup_2026_matches_v1
     WHERE source = 'openfootball' AND tournament = 'world-cup-2026') AS goals_per_match;

-- Match catalog and bracket source
SELECT
    match_number,
    round,
    group_name,
    match_date,
    venue,
    home_team,
    home_goals,
    away_goals,
    away_team,
    shootout_home_goals,
    shootout_away_goals
FROM touchline_worldcup_2026_matches_v1
WHERE source = 'openfootball' AND tournament = 'world-cup-2026'
ORDER BY match_number;

-- Top scorers, excluding own goals
SELECT
    player_name,
    team_name,
    count() AS goals,
    countIf(is_penalty = 1) AS penalty_goals,
    uniqExact(match_number) AS matches
FROM touchline_worldcup_2026_goals_v1
WHERE source = 'openfootball' AND tournament = 'world-cup-2026' AND is_own_goal = 0
GROUP BY player_name, team_name
ORDER BY goals DESC, penalty_goals ASC, player_name
LIMIT 20;

-- Goal timing, including extra time
SELECT
    multiIf(
        minute_value <= 15, '0-15',
        minute_value <= 30, '16-30',
        minute_value <= 45, '31-45+',
        minute_value <= 60, '46-60',
        minute_value <= 75, '61-75',
        minute_value <= 90, '76-90+',
        minute_value <= 105, '91-105',
        '106-120'
    ) AS timing_bucket,
    count() AS goals
FROM touchline_worldcup_2026_goals_v1
WHERE source = 'openfootball' AND tournament = 'world-cup-2026'
GROUP BY timing_bucket
ORDER BY timing_bucket;

-- Venue footprint
SELECT
    venue,
    count() AS matches,
    sum(home_goals + away_goals) AS goals,
    avg(home_goals + away_goals) AS goals_per_match
FROM touchline_worldcup_2026_matches_v1
WHERE source = 'openfootball' AND tournament = 'world-cup-2026'
GROUP BY venue
ORDER BY matches DESC, goals DESC;

-- Index experiment
EXPLAIN indexes = 1
SELECT count()
FROM touchline_worldcup_2026_goals_v1
WHERE source = 'openfootball' AND tournament = 'world-cup-2026' AND match_number BETWEEN 73 AND 104;
