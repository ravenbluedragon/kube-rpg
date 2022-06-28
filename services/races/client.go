package main

import (
	"context"
	"database/sql"
	_ "embed"
	"log"

	"github.com/machinebox/graphql"
)

type raceData struct {
	Name      string
	Languages []struct {
		Name string
	}
	Size  string
	Speed int32
}

func (rd raceData) convert() Race {
	languages := make([]string, len(rd.Languages))
	for i, l := range rd.Languages {
		languages[i] = l.Name
	}
	return Race{
		id:        0,
		name:      rd.Name,
		languages: languages,
		size:      sql.NullString{String: rd.Size, Valid: true},
		speed:     sql.NullInt32{Int32: rd.Speed, Valid: true},
	}
}

//go:embed race.gql
var race_gql string

func collect_from_api(target string) ([]Race, error) {
	client := graphql.NewClient(target)
	req := graphql.NewRequest(race_gql)

	ctx := context.Background()
	var resp struct{ Races []raceData }

	if err := client.Run(ctx, req, &resp); err != nil {
		log.Println("Failed to retrieve results")
		return nil, err
	}

	races := resp.Races
	log.Printf("Retrieved %d rows of race data", len(races))
	data := make([]Race, len(races))
	for i, rd := range races {
		data[i] = rd.convert()
	}

	return data, nil
}
