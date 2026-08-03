-- Replace the run ID before running this file.
-- This lab validates the contract-v2 fact grain against the original match facts.

SELECT
    run_id,
    any(career_id) AS career_id,
    any(season) AS season,
    max(last_round) AS last_round,
    max(rounds_completed) AS rounds_completed,
    sum(matches_in_round) AS reported_matches,
    any(clubs_expected) AS clubs_expected,
    argMax(status, updated_at) AS status,
    argMax(schema_version, updated_at) AS schema_version
FROM touchline_runs_v2
WHERE run_id = 'replace-with-run-id'
GROUP BY run_id;

SELECT
    count() AS matches,
    countIf(club_fact_count = 2) AS matches_with_two_club_facts,
    countIf(player_fact_count = rating_count) AS matches_with_complete_player_facts,
    countIf(club_fact_count != 2 OR player_fact_count != rating_count) AS incomplete_matches
FROM
(
    SELECT
        m.match_id,
        countDistinct(cf.club_id) AS club_fact_count,
        countDistinct(pf.player_id) AS player_fact_count,
        countDistinct(r.player_id) AS rating_count
    FROM touchline_matches_v2 AS m
    LEFT JOIN touchline_match_club_facts_v2 AS cf
        ON cf.run_id = m.run_id AND cf.match_id = m.match_id
    LEFT JOIN touchline_player_match_facts_v2 AS pf
        ON pf.run_id = m.run_id AND pf.match_id = m.match_id
    LEFT JOIN touchline_player_ratings_v2 AS r
        ON r.run_id = m.run_id AND r.match_id = m.match_id
    WHERE m.run_id = 'replace-with-run-id'
    GROUP BY m.match_id
);

SELECT
    count() AS compared_club_facts,
    countIf(abs(cf.xg_for - if(cf.is_home, m.home_xg, m.away_xg)) > 0.0001) AS xg_mismatches,
    countIf(cf.shots != if(cf.is_home, m.home_shots, m.away_shots)) AS shot_mismatches,
    countIf(abs(cf.possession - if(cf.is_home, m.home_possession, 100 - m.home_possession)) > 0.0001) AS possession_mismatches
FROM touchline_match_club_facts_v2 AS cf
INNER JOIN touchline_matches_v2 AS m
    ON m.run_id = cf.run_id AND m.match_id = cf.match_id
WHERE cf.run_id = 'replace-with-run-id';

SELECT
    club_id,
    position,
    appearances,
    minutes,
    round(average_rating, 2) AS average_rating,
    goals,
    shots,
    round(total_xg, 3) AS xg,
    round(total_xg / greatest(minutes, 1) * 90, 3) AS xg_per_90
FROM
(
    SELECT
        club_id,
        position,
        count() AS appearances,
        sum(minutes_played) AS minutes,
        avg(rating) AS average_rating,
        sum(goals) AS goals,
        sum(shots) AS shots,
        sum(xg) AS total_xg
    FROM touchline_player_match_facts_v2
    WHERE run_id = 'replace-with-run-id'
    GROUP BY club_id, position
)
ORDER BY average_rating DESC;

SELECT
    formation,
    mentality,
    round(avg(pressing), 1) AS pressing,
    round(avg(tempo), 1) AS tempo,
    round(avg(defensive_line), 1) AS defensive_line,
    round(avg(xg_for), 2) AS average_xg,
    round(avg(press_wins), 1) AS average_press_wins,
    round(avg(box_entries), 1) AS average_box_entries,
    count() AS club_matches
FROM touchline_match_club_facts_v2
WHERE run_id = 'replace-with-run-id'
GROUP BY formation, mentality
ORDER BY average_xg DESC;
