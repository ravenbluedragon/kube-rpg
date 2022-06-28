INSERT INTO race_language
(   race_id
,   language_id
)
SELECT
    (SELECT id FROM races WHERE name = $1)
,   (SELECT id FROM languages WHERE name = $2)
