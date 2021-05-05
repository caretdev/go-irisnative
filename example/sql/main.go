package main

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/caretdev/go-irisnative"
)

func main() {
	// var addr = "localhost:1972"
	// var namespace = "%SYS"
	// var login = "_SYSTEM"
	// var password = "SYS"

	db, err := sql.Open("intersystems", "iris://_SYSTEM:SYS@localhost:1972/USER")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// rows, err := db.QueryContext(ctx, `SELECT date($horolog) AS "one", 2 AS "two", 3 as "three"`)
	rows, err := db.QueryContext(ctx, `SELECT name FROM %Dictionary.ClassDefinition order by name`)
	// rows, err := db.QueryContext(ctx, `SELECT count(*) FROM %Dictionary.ClassDefinition`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	columns, _ := rows.Columns()
	cnt := len(columns)

	rawResult := make([][]byte, cnt)
	result := make([]string, cnt)

	dest := make([]interface{}, cnt)
	for i := range rawResult {
		dest[i] = &rawResult[i]
	}
	counter := 0
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			panic(err)
		}
		for i, raw := range rawResult {
			if raw == nil {
				result[i] = "NULL"
			} else {
				result[i] = string(raw)
			}
		}
		counter++
		log.Printf("%3d: %s", counter, result[0])
	}

}
