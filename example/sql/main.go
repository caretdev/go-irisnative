package main

import (
	"context"
	"database/sql"
	"fmt"
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

	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	row := db.QueryRow("select 123 as \"demo﹒val\"")
	fmt.Printf("Row: %#v\n", row)

	// db.Exec(`truncate table companies`)
	// // rows, err := db.QueryContext(ctx, `SELECT date($horolog) AS "one", 2 AS "two", 3 as "three"`)
	// db.ExecContext(ctx, `insert into companies (name) values (?)`, "test1", "test2")
	// rows, err := db.QueryContext(ctx, `SELECT * FROM companies`)
	// if err != nil {
	// 	panic(err)
	// }
	// defer rows.Close()
	// columns, _ := rows.Columns()
	// // cnt := len(columns)
	// fmt.Println("Columns: ", columns)

}
