package main

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var _db *sql.DB // not ideal but quick

type route struct {
	path    string
	handler func(http.ResponseWriter, *http.Request)
}

var routes = []route{
	{"/", root},
	{"/races", list_races},
	{"/races/delete", delete_races},
	{"/race/", show_race},
	{"/race/new", add_race},
	{"/languages", list_languages},
	{"/language/new", add_language},
}

func run_server(db *sql.DB) {
	_db = db
	for _, route := range routes {
		http.HandleFunc(route.path, route.handler)
	}
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalln("Server Error", err)
	}
}

type raceExport struct {
	Id        int      `json:"id"`
	Name      string   `json:"name"`
	Size      *string  `json:"size,omitempty"`
	Speed     *int     `json:"speed,omitempty"`
	Languages []string `json:"languages,omitempty"`
}

func (r *Race) export() raceExport {
	re := raceExport{Id: r.id, Name: r.name, Languages: r.languages}
	if r.size.Valid {
		size := r.size.String
		re.Size = &size
	}
	if r.speed.Valid {
		speed := int(r.speed.Int32)
		re.Speed = &speed
	}
	return re
}

func exportRaces(races []Race) []raceExport {
	re := make([]raceExport, len(races))
	for i, r := range races {
		re[i] = r.export()
	}
	return re
}

type errorResponse struct {
	Message    string
	StatusCode int
}

//go:embed server_root.txt
var server_root string

// API Root
func root(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "ROOT ")
	io.WriteString(w, r.URL.String())
	io.WriteString(w, "\n")
	io.WriteString(w, server_root)
}

// List Races
func list_races(w http.ResponseWriter, r *http.Request) {
	races, err := read_races(_db)
	var b []byte
	if err == nil {
		if len(races) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		data := struct {
			Races []raceExport `json:"races"`
		}{
			Races: exportRaces(races),
		}
		b, err = json.Marshal(data)
	}
	if err == nil {
		w.WriteHeader(http.StatusOK)
	}
	if err != nil {
		e := errorResponse{Message: err.Error(), StatusCode: http.StatusInternalServerError}
		w.WriteHeader(e.StatusCode)
		b, err = json.Marshal(&e)
		if err != nil {
			io.WriteString(w, "Something went very wrong")
			return
		}
		w.Write(b)
		return
	}
	w.Write(b)
}

// List Languages
func list_languages(w http.ResponseWriter, r *http.Request) {
	langs, err := read_languages(_db)
	var b []byte
	if err == nil {
		if len(langs) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		data := struct {
			Langs []string `json:"languages"`
		}{
			Langs: langs,
		}
		b, err = json.Marshal(data)
	}
	if err == nil {
		w.WriteHeader(http.StatusOK)
	}
	if err != nil {
		e := errorResponse{Message: err.Error(), StatusCode: http.StatusInternalServerError}
		w.WriteHeader(e.StatusCode)
		b, err = json.Marshal(&e)
		if err != nil {
			io.WriteString(w, "Something went very wrong")
			return
		}
		w.Write(b)
		return
	}
	w.Write(b)
}

// Show Race with id
func show_race(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")
	if len(path) != 3 || path[1] != "race" {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, "Path not defined")
		return
	}
	id, err := strconv.Atoi(path[2])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, "No resource with this id")
		return
	}
	race, ok, err := read_race(_db, id)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var b []byte
	if err == nil {
		data := race.export()
		b, err = json.Marshal(data)
	}
	if err == nil {
		w.WriteHeader(http.StatusOK)
	}
	if err != nil {
		e := errorResponse{Message: err.Error(), StatusCode: http.StatusInternalServerError}
		w.WriteHeader(e.StatusCode)
		b, err = json.Marshal(&e)
		if err != nil {
			io.WriteString(w, "Something went very wrong")
			return
		}
		w.Write(b)
		return
	}
	w.Write(b)
}

type addLang struct {
	Name string `json:"name"`
}

// add language
func add_language(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "This is a POST endpoint")
		return
	}
	var data addLang
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil && err != io.EOF {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, err.Error())
		return
	}

	if data.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "Name must not be blank")
		return
	}

	_, ok, err := add_lang(_db, data.Name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error())
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNotModified)
		io.WriteString(w, "Already exists")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// add race
func add_race(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "This is a POST endpoint")
		return
	}
	var data raceExport
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil && err != io.EOF {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, err.Error())
		return
	}

	if data.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "Name must not be blank")
		return
	}

	race := Race{name: data.Name, languages: data.Languages}
	if data.Size != nil {
		race.size = sql.NullString{String: *data.Size, Valid: true}
	}
	if data.Speed != nil {
		race.speed = sql.NullInt32{Int32: int32(*data.Speed), Valid: true}
	}

	id, err := write_race(_db, race)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error())
		return
	}
	data.Id = id
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Println("Encode json err", err)
	}
}

// delete all races
func delete_races(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "This is a DELETE endpoint")
		return
	}
	err := truncate_races(_db)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
