package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/caretdev/go-irisnative"
)

func main() {
	dsn := "iris://_SYSTEM:SYS@localhost:1972/USER"
	db, err := sql.Open("iris", dsn)
	if err != nil { log.Fatal(err) }
	defer db.Close()

	// Connection pool tuning
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = db.ExecContext(ctx, `DROP TABLE IF EXISTS demo_person`)
	if err != nil { log.Fatal("drop table:", err) }

	// 1) Create a table (id INT PRIMARY KEY, name VARCHAR(80))
	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS demo_person (
		id INT PRIMARY KEY,
		name VARCHAR(80) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil { log.Fatal("create table:", err) }

	// 2) Insert with placeholders
	res, err := db.ExecContext(ctx, `INSERT INTO demo_person(id, name) VALUES(?, ?)`, 1, "Alice")
	if err != nil { log.Fatal("insert:", err) }
	if n, _ := res.RowsAffected(); n > 0 { fmt.Println("inserted:", n) }

	// 3) Query rows
	rows, err := db.QueryContext(ctx, `SELECT id, name, created_at FROM demo_person ORDER BY id`)
	if err != nil { log.Fatal("query:", err) }
	defer rows.Close()

	for rows.Next() {
		var (
			id int
			name string
			createdAt time.Time
		)
		if err := rows.Scan(&id, &name, &createdAt); err != nil { log.Fatal(err) }
		fmt.Printf("row: id=%d name=%s created_at=%s\n", id, name, createdAt.Format(time.RFC3339))
	}
	if err := rows.Err(); err != nil { log.Fatal(err) }

	// 4) Prepared statement
	stmt, err := db.PrepareContext(ctx, `UPDATE demo_person SET name=? WHERE id=?`)
	if err != nil { log.Fatal("prepare:", err) }
	defer stmt.Close()
	if _, err := stmt.ExecContext(ctx, "Alice Updated", 1); err != nil { log.Fatal("update:", err) }

	// 5) Transaction example
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil { log.Fatal("begin tx:", err) }
	if _, err := tx.ExecContext(ctx, `INSERT INTO demo_person(id, name) VALUES(?, ?)`, 2, "Bob"); err != nil {
		tx.Rollback()
		log.Fatal("tx insert:", err)
	}
	if err := tx.Commit(); err != nil { log.Fatal("commit:", err) }
}
