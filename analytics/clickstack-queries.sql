WITH latest_clubs AS
(
    SELECT
        season,
        club_id,
        argMax(short_name, ingested_at) AS club_name
    FROM touchline_clubs
    GROUP BY season, club_id
)
SELECT
    min(match_time) AS ts,
    season,
    round,
    club_id,
    any(club_name) AS club_name,
    round(avg(xg), 2) AS xg
FROM
(
    SELECT
        m.ingested_at AS match_time,
        m.season,
        m.round,
        m.home_id AS club_id,
        coalesce(c.club_name, m.home_id) AS club_name,
        m.home_xg AS xg
    FROM touchline_matches AS m
    LEFT JOIN latest_clubs AS c ON c.season = m.season AND c.club_id = m.home_id
    UNION ALL
    SELECT
        m.ingested_at AS match_time,
        m.season,
        m.round,
        m.away_id AS club_id,
        coalesce(c.club_name, m.away_id) AS club_name,
        m.away_xg AS xg
    FROM touchline_matches AS m
    LEFT JOIN latest_clubs AS c ON c.season = m.season AND c.club_id = m.away_id
)
GROUP BY season, round, club_id
ORDER BY ts, club_name;

SELECT
    min(ingested_at) AS ts,
    season,
    round,
    match_id,
    event_type,
    team_id,
    any(player_name) AS player_name,
    count() AS events
FROM touchline_match_events
GROUP BY season, round, match_id, event_type, team_id
ORDER BY ts, match_id, event_type;

SELECT
    min(ingested_at) AS ts,
    season,
    club_id,
    player_id,
    any(player_name) AS player_name,
    count() AS matches,
    round(avg(rating), 2) AS average_rating
FROM touchline_player_ratings
GROUP BY season, club_id, player_id
ORDER BY average_rating DESC;

SELECT
    min(ingested_at) AS ts,
    season,
    round,
    match_id,
    team_id,
    player_id,
    any(player_name) AS player_name,
    intDiv(toInt32(player_x), 5) * 5 AS x_bin,
    intDiv(toInt32(player_y), 5) * 5 AS y_bin,
    count() AS intensity
FROM touchline_player_frames
GROUP BY season, round, match_id, team_id, player_id, x_bin, y_bin
ORDER BY ts, match_id, team_id, player_id, intensity DESC;
