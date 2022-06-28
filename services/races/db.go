package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"

	"github.com/lib/pq"
)

type conn struct {
	host string
	port string
	user string
	pass string
	name string
}

func connect(c *conn) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		c.host, c.port, c.user, c.pass, c.name)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	log.Printf("Connected Successfully to DB %s!\n", c.name)
	return db, nil
}

//go:embed sql/init.sql
var sql_init string

func init_db(db *sql.DB) error {
	_, err := db.Exec(sql_init)
	return err
}

//go:embed sql/write_race.sql
var sql_write_race string

//go:embed sql/write_lang.sql
var sql_write_language string

// write race to db
func write_race(db *sql.DB, race Race) (int, error) {
	txn, err := db.Begin()
	if err != nil {
		log.Println("Failed to create transaction", err)
		return 0, err
	}
	err = db.QueryRow(sql_write_race, race.name, race.size, race.speed).Scan(&race.id)
	if err != nil {
		log.Printf("Error writing race (%s) to DB: (%v)", race.name, err)
		txn.Rollback()
		return 0, err
	}
	ins_lang, err := txn.Prepare(sql_write_language)
	if err != nil {
		log.Printf("Error writing lang (%s) to DB: (%v)", race.name, err)
		txn.Rollback()
		return 0, err
	}
	for _, lang := range race.languages {
		_, err = ins_lang.Exec(lang, race.id)
		if err != nil {
			log.Printf("Error writing language (%s) to DB: (%v)", lang, err)
			txn.Rollback()
			return 0, err
		}
	}
	if err = ins_lang.Close(); err != nil {
		log.Println("Failed to close statement", err)
		txn.Rollback()
		return 0, err
	}
	if err = txn.Commit(); err != nil {
		log.Println("Failed to commit", err)
		txn.Rollback()
		return 0, err
	}
	return race.id, nil
}

//go:embed sql/truncate_races.sql
var sql_truncate_races string

func truncate_races(db *sql.DB) error {
	_, err := db.Exec(sql_truncate_races)
	if err != nil {
		log.Println("Truncate failed:", err)
	}
	return nil
}

//go:embed sql/read_races.sql
var sql_read_races string

// read_races all data from db
func read_races(db *sql.DB) ([]Race, error) {
	rows, err := db.Query(sql_read_races)
	if err != nil {
		log.Println("Failed to query races", err)
		return nil, err
	}
	defer rows.Close()
	var races []Race
	var race Race
	for rows.Next() {
		if err = rows.Scan(&race.id, &race.name, &race.size, &race.speed, pq.Array(&race.languages)); err != nil {
			log.Println("Error scanning race", err)
			return nil, err
		}
		races = append(races, race)
	}
	return races, nil
}

// bulk write date to db
func write_data(db *sql.DB, data []Race) error {
	txn, err := db.Begin()
	if err != nil {
		log.Println("Failed to create transaction")
		return err
	}

	if err = populate_races(txn, data); err != nil {
		if e := txn.Rollback(); e != nil {
			log.Println("Rollback Fail", e)
		}
		return err
	}

	if err = populate_languages(txn, data); err != nil {
		if e := txn.Rollback(); e != nil {
			log.Println("Rollback Fail", e)
		}
		return err
	}

	for _, r := range data {
		fmt.Println(">", r)
	}

	if err = txn.Commit(); err != nil {
		log.Println("Commit Transaction fail", err)
	}
	return nil
}

// populate main races table
func populate_races(txn *sql.Tx, data []Race) error {
	stmt, err := txn.Prepare(pq.CopyIn("races", "name", "size", "speed"))
	if err != nil {
		log.Println("Failed to prepare statement", err)
		return err
	}

	for _, row := range data {
		if _, err = stmt.Exec(row.name, row.size, row.speed); err != nil {
			// note that error may not relate to specific row
			log.Println("Copy fail", err)
			return err
		}
	}

	if _, err = stmt.Exec(); err != nil {
		log.Println("Flush fail", err)
		return err
	}

	if err = stmt.Close(); err != nil {
		log.Println("Statement close fail", err)
	}
	return nil
}

//go:embed sql/pop_lang.sql
var lang_qry string

//go:embed sql/pop_link.sql
var link_qry string

// populate languages table
func populate_languages(txn *sql.Tx, data []Race) error {
	lang_stmt, err := txn.Prepare(lang_qry)
	if err != nil {
		log.Println("Failed to prepare lang statement", err)
		return err
	}
	link_stmt, err := txn.Prepare(link_qry)
	if err != nil {
		log.Println("Failed to prepare link statement", err)
		return err
	}

	seen_lang := make(map[string]bool)

	for _, row := range data {
		for _, lang := range row.languages {
			if !seen_lang[lang] {
				// note that error may not relate to specific row
				if _, err = lang_stmt.Exec(lang); err != nil {
					log.Println("Populate fail", err)
					return err
				}
			}
			if _, err = link_stmt.Exec(row.name, lang); err != nil {
				log.Println("Populate link fail", err)
				return err
			}
		}
	}

	if err = lang_stmt.Close(); err != nil {
		log.Println("Statement close fail", err)
		return err
	}
	if err = link_stmt.Close(); err != nil {
		log.Println("Statement close fail", err)
		return err
	}
	return nil
}

//go:embed sql/read_languages.sql
var sql_read_languages string

// read all languages
func read_languages(db *sql.DB) ([]string, error) {
	var data []string
	var name string
	rows, err := db.Query(sql_read_languages)
	if err != nil {
		log.Println("Failed to get languages", err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&name)
		if err != nil {
			log.Println("Failed to scan row", err)
			return nil, err
		}
		data = append(data, name)
	}
	return data, nil
}

//go:embed sql/read_race_id.sql
var sql_read_race_id string

// read_race data from db
func read_race(db *sql.DB, id int) (Race, bool, error) {
	var race Race
	err := db.QueryRow(sql_read_race_id, id).Scan(&race.id, &race.name, &race.size, &race.speed, pq.Array(&race.languages))
	if err != nil {
		if err == sql.ErrNoRows {
			return race, false, nil
		}
		log.Println("Failed to query race", err)
		return race, false, err
	}
	return race, true, nil
}

//go:embed sql/add_lang.sql
var sql_add_lang string

// add lang to db
func add_lang(db *sql.DB, name string) (int, bool, error) {
	var id int
	err := db.QueryRow(sql_add_lang, name).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return id, false, nil
		}
		log.Println("Failed to create lang", name, err)
		return id, false, err
	}
	return id, true, nil
}
