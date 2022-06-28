SELECT
    id
,   name
,   size
,   speed
,   (
        SELECT array_agg(name) langs
        FROM languages l
        INNER JOIN race_language rl
        ON l.id = rl.language_id
        WHERE rl.race_id = $1
    ) languages
FROM races r
WHERE id = $1
