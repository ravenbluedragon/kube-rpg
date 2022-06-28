INSERT INTO races (name, size, speed)
VALUES ($1, $2, $3)
RETURNING id