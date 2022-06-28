package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func env(keys ...string) []string {
	var missing []string
	values := make([]string, len(keys))
	for i, key := range keys {
		val, ok := os.LookupEnv(key)
		if ok {
			values[i] = val
		} else {
			missing = append(missing, key)
		}
	}
	if len(missing) != 0 {
		log.Fatalln("Missing Env Vars", strings.Join(missing, ", "))
	}
	return values
}

func main() {
	db_env := env("POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB")
	data_url := env("DATA_URL")[0]
	c := conn{
		host: db_env[0],
		port: db_env[1],
		user: db_env[2],
		pass: db_env[3],
		name: db_env[4],
	}

	db, err := connect(&c)
	if err != nil {
		log.Fatalln("Failed to connect to DB", err)
	}
	defer db.Close()

	if err = init_db(db); err != nil {
		log.Fatalln("Failed to init DB", err)
		return
	}

	populate_from_api(db, data_url)
	run_server(db)
}

type Race struct {
	id        int
	name      string
	languages []string
	size      sql.NullString
	speed     sql.NullInt32
}

func (r Race) String() string {
	size := r.size.String
	if !r.size.Valid {
		size = "<NULL>"
	}
	speed := strconv.Itoa(int(r.speed.Int32))
	if !r.speed.Valid {
		speed = "<NULL>"
	}
	languages := strings.Join(r.languages, ", ")
	return fmt.Sprintf("Race %s (%d) Size: %s, Speed: %s, Languages: %s", r.name, r.id, size, speed, languages)
}

func populate_from_api(db *sql.DB, data_url string) {
	data, err := collect_from_api(data_url)
	if err != nil {
		log.Println("Failed to collect data from api", err)
		return
	}

	err = write_data(db, data)
	if err != nil {
		log.Println("Failed to write api data", err)
		return
	}
}
