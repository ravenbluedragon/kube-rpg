SELECT
    r.id
,   r.name
,   r.size
,   r.speed
,   langs
FROM races r
LEFT JOIN LATERAL (
    SELECT
        rl.race_id
    ,   array_agg(name) langs
    FROM languages l
    INNER JOIN race_language rl
    ON l.id = rl.language_id
    GROUP BY rl.race_id
) l ON r.id = l.race_id
;