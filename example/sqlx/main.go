package main

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "github.com/caretdev/go-irisnative" // driver
	"github.com/jmoiron/sqlx"
)

type Person struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

func create(ctx context.Context, db *sqlx.DB) {
	drop(ctx, db)
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS demo_person (
		id INT PRIMARY KEY,
		name VARCHAR(80) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		panic(err)
	}
}

func drop(ctx context.Context, db *sqlx.DB) {
	_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS demo_person`)
	if err != nil {
		panic(err)
	}
}

func main() {
	ctx := context.Background()
	dsn := "iris://_SYSTEM:SYS@localhost:1972/USER"
	db := sqlx.MustConnect("iris", dsn)
	defer db.Close()

	create(ctx, db)
	defer drop(ctx, db)

	// Struct-based insert with NamedExec
	p := Person{ID: 3, Name: "Carol"}
	_, err := db.NamedExecContext(ctx,
		`INSERT INTO demo_person(id, name) VALUES(:id, :name)`, p,
	)
	if err != nil {
		log.Fatal("named insert:", err)
	}

	// Select into slice of structs
	var people []Person
	if err := db.SelectContext(ctx, &people, `SELECT id, name, created_at FROM demo_person ORDER BY id`); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("people: %#v\n", people)

	// Get a single struct
	var one Person
	if err := db.GetContext(ctx, &one, `SELECT id, name, created_at FROM demo_person WHERE id=?`, people[0].ID); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("one: %+v\n", one)

	// Named query with IN (sqlx.In)
	ids := []int{1, 2, 3}
	q, args, err := sqlx.In(`SELECT id, name FROM demo_person WHERE id IN (?)`, ids)
	if err != nil {
		log.Fatal(err)
	}
	q = db.Rebind(q) // ensure driver-specific bindvars
	rows, err := db.QueryxContext(ctx, q, args...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, name)
	}
}
