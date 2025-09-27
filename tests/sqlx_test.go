package tests_test

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"
	"time"

	"database/sql"
	"database/sql/driver"

	_ "github.com/caretdev/go-irisnative"
	. "github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

var _, _ Ext = &DB{}, &Tx{}
var _, _ ColScanner = &Row{}, &Rows{}

// var _ Queryer = &qStmt{}
// var _ Execer = &qStmt{}

var irisdb *DB

func init() {
	var err error
	irisdb, err = Connect("intersystems", connectionString)
	if err != nil {
		panic(err)
	}
}

type Schema struct {
	create string
	drop   string
}

var defaultSchema = Schema{
	create: `
CREATE TABLE person (
	first_name VARCHAR(65535),
	last_name VARCHAR(65535),
	email VARCHAR(65535),
	added_at timestamp default CURRENT_TIMESTAMP
);

CREATE TABLE place (
	country VARCHAR(65535),
	city VARCHAR(65535) NULL,
	telcode integer
);

CREATE TABLE capplace (
	"COUNTRY" VARCHAR(65535),
	"CITY" VARCHAR(65535) NULL,
	"TELCODE" integer
);

CREATE TABLE nullperson (
    first_name VARCHAR(65535) NULL,
    last_name VARCHAR(65535) NULL,
    email VARCHAR(65535) NULL
);

CREATE TABLE employees (
	name VARCHAR(65535),
	id integer,
	boss_id integer
);

`,
	drop: `
drop table person;
drop table place;
drop table capplace;
drop table nullperson;
drop table employees;
`,
}

func (s Schema) IRIS() (string, string, string) {
	return s.create, s.drop, `CURRENT_TIMESTAMP`
}

func loadDefaultFixture(db *DB, t *testing.T) {
	tx := db.MustBegin()
	tx.MustExec(tx.Rebind("INSERT INTO person (first_name, last_name, email) VALUES (?, ?, ?)"), "Jason", "Moiron", "jmoiron@jmoiron.net")
	tx.MustExec(tx.Rebind("INSERT INTO person (first_name, last_name, email) VALUES (?, ?, ?)"), "John", "Doe", "johndoeDNE@gmail.net")
	tx.MustExec(tx.Rebind("INSERT INTO place (country, city, telcode) VALUES (?, ?, ?)"), "United States", "New York", "1")
	tx.MustExec(tx.Rebind("INSERT INTO place (country, telcode) VALUES (?, ?)"), "Hong Kong", "852")
	tx.MustExec(tx.Rebind("INSERT INTO place (country, telcode) VALUES (?, ?)"), "Singapore", "65")
	if db.DriverName() == "mysql" {
		tx.MustExec(tx.Rebind("INSERT INTO capplace (`COUNTRY`, `TELCODE`) VALUES (?, ?)"), "Sarf Efrica", "27")
	} else {
		tx.MustExec(tx.Rebind("INSERT INTO capplace (\"COUNTRY\", \"TELCODE\") VALUES (?, ?)"), "Sarf Efrica", "27")
	}
	tx.MustExec(tx.Rebind("INSERT INTO employees (name, id) VALUES (?, ?)"), "Peter", "4444")
	tx.MustExec(tx.Rebind("INSERT INTO employees (name, id, boss_id) VALUES (?, ?, ?)"), "Joe", "1", "4444")
	tx.MustExec(tx.Rebind("INSERT INTO employees (name, id, boss_id) VALUES (?, ?, ?)"), "Martin", "2", "4444")
	tx.Commit()
}

func MultiExec(e Execer, query string) {
	stmts := strings.Split(query, ";\n")
	if len(strings.Trim(stmts[len(stmts)-1], " \n\t\r")) == 0 {
		stmts = stmts[:len(stmts)-1]
	}
	for _, s := range stmts {
		// fmt.Printf("MultiExec...\n%s\n", s)
		_, err := e.Exec(s)
		if err != nil {
			fmt.Println(err, s)
		}
	}
}

func RunWithSchema(schema Schema, t *testing.T, test func(db *DB, t *testing.T, now string)) {
	runner := func(db *DB, t *testing.T, create, drop, now string) {
		defer func() {
			MultiExec(db, drop)
		}()

		MultiExec(db, create)
		test(db, t, now)
	}

	create, drop, now := schema.IRIS()
	runner(irisdb, t, create, drop, now)
}

func TestSimple(t *testing.T) {
	var schema = Schema{
		create: `drop table if exists kv;
CREATE TABLE kv ( k varchar(50), v integer );`,
		drop: `drop table kv;`,
	}
	RunWithSchema(schema, t, func(db *DB, t *testing.T, now string) {
		var k string = "hi"
		var v int = -20
		_, err := db.Exec(db.Rebind("INSERT INTO kv ( k , v ) VALUES ( ? , ? )"), k, v)
		if err != nil {
			t.Error(err)
		}

		rows, err := db.Queryx("SELECT * FROM kv")
		if err != nil {
			t.Error(err)
		}
		defer rows.Close()
		for rows.Next() {
			// var k string
			// var v int
			err = rows.Scan(&k, &v)
			if err != nil {
				break
			}
		}

	})
}

func TestSelectReset(t *testing.T) {
	RunWithSchema(defaultSchema, t, func(db *DB, t *testing.T, now string) {
		loadDefaultFixture(db, t)

		filledDest := []string{"a", "b", "c"}
		err := db.Select(&filledDest, "SELECT first_name FROM person ORDER BY first_name ASC;")
		if err != nil {
			t.Fatal(err)
		}
		if len(filledDest) != 2 {
			t.Errorf("Expected 2 first names, got %d.", len(filledDest))
		}
		expected := []string{"Jason", "John"}
		for i, got := range filledDest {
			if got != expected[i] {
				t.Errorf("Expected %d result to be %s, but got %s.", i, expected[i], got)
			}
		}

		var emptyDest []string
		err = db.Select(&emptyDest, "SELECT first_name FROM person WHERE first_name = 'Jack';")
		if err != nil {
			t.Fatal(err)
		}
		// Verify that selecting 0 rows into a nil target didn't create a
		// non-nil slice.
		if emptyDest != nil {
			t.Error("Expected emptyDest to be nil")
		}
	})
}

type Message struct {
	Text       string      `db:"string"`
	Properties PropertyMap `db:"properties"` // Stored as JSON in the database
}

type PropertyMap map[string]string

// Implement driver.Valuer and sql.Scanner interfaces on PropertyMap
func (p PropertyMap) Value() (driver.Value, error) {
	if len(p) == 0 {
		return nil, nil
	}
	return json.Marshal(p)
}

func (p PropertyMap) Scan(src interface{}) error {
	v := reflect.ValueOf(src)
	if !v.IsValid() || v.CanAddr() && v.IsNil() {
		return nil
	}
	switch ts := src.(type) {
	case []byte:
		return json.Unmarshal(ts, &p)
	case string:
		return json.Unmarshal([]byte(ts), &p)
	default:
		return fmt.Errorf("Could not not decode type %T -> %T", src, p)
	}
}

func TestEmbeddedMaps(t *testing.T) {
	var schema = Schema{
		create: `
			CREATE TABLE message (
				string varchar(65535),
				properties varchar(65535)
			);`,
		drop: `drop table message;`,
	}

	RunWithSchema(schema, t, func(db *DB, t *testing.T, now string) {
		messages := []Message{
			{"Hello, World", PropertyMap{"one": "1", "two": "2"}},
			{"Thanks, Joy", PropertyMap{"pull": "request"}},
		}
		q1 := `INSERT INTO message (string, properties) VALUES (:string, :properties);`
		for _, m := range messages {
			_, err := db.NamedExec(q1, m)
			if err != nil {
				t.Fatal(err)
			}
		}
		var count int
		err := db.Get(&count, "SELECT count(*) FROM message")
		if err != nil {
			t.Fatal(err)
		}
		if count != len(messages) {
			t.Fatalf("Expected %d messages in DB, found %d", len(messages), count)
		}

		var m Message
		err = db.Get(&m, "SELECT * FROM message LIMIT 1;")
		if err != nil {
			t.Fatal(err)
		}
		if m.Properties == nil {
			t.Fatal("Expected m.Properties to not be nil, but it was.")
		}
	})
}

func TestMissingNames(t *testing.T) {
	RunWithSchema(defaultSchema, t, func(db *DB, t *testing.T, now string) {
		loadDefaultFixture(db, t)
		type PersonPlus struct {
			FirstName string `db:"first_name"`
			LastName  string `db:"last_name"`
			Email     string
			// AddedAt time.Time `db:"added_at"`
		}

		// test Select first
		pps := []PersonPlus{}
		// pps lacks added_at destination
		err := db.Select(&pps, "SELECT * FROM person")
		if err == nil {
			t.Error("Expected missing name from Select to fail, but it did not.")
		}

		// test Get
		pp := PersonPlus{}
		err = db.Get(&pp, "SELECT * FROM person LIMIT 1")
		if err == nil {
			t.Error("Expected missing name Get to fail, but it did not.")
		}

		// test naked StructScan
		pps = []PersonPlus{}
		rows, err := db.Query("SELECT * FROM person LIMIT 1")
		if err != nil {
			t.Fatal(err)
		}
		rows.Next()
		err = StructScan(rows, &pps)
		if err == nil {
			t.Error("Expected missing name in StructScan to fail, but it did not.")
		}
		rows.Close()

		// now try various things with unsafe set.
		db = db.Unsafe()
		pps = []PersonPlus{}
		err = db.Select(&pps, "SELECT * FROM person")
		if err != nil {
			t.Error(err)
		}

		// test Get
		pp = PersonPlus{}
		err = db.Get(&pp, "SELECT * FROM person LIMIT 1")
		if err != nil {
			t.Error(err)
		}

		// test naked StructScan
		pps = []PersonPlus{}
		rowsx, err := db.Queryx("SELECT * FROM person LIMIT 1")
		if err != nil {
			t.Fatal(err)
		}
		rowsx.Next()
		err = StructScan(rowsx, &pps)
		if err != nil {
			t.Error(err)
		}
		rowsx.Close()

		// // test Named stmt
		// if !isUnsafe(db) {
		// 	t.Error("Expected db to be unsafe, but it isn't")
		// }
		// nstmt, err := db.PrepareNamed(`SELECT * FROM person WHERE first_name != :name`)
		// if err != nil {
		// 	t.Fatal(err)
		// }
		// // its internal stmt should be marked unsafe
		// if !nstmt.Stmt.unsafe {
		// 	t.Error("expected NamedStmt to be unsafe but its underlying stmt did not inherit safety")
		// }
		// pps = []PersonPlus{}
		// err = nstmt.Select(&pps, map[string]interface{}{"name": "Jason"})
		// if err != nil {
		// 	t.Fatal(err)
		// }
		// if len(pps) != 1 {
		// 	t.Errorf("Expected 1 person back, got %d", len(pps))
		// }

		// // test it with a safe db
		// db.unsafe = false
		// if isUnsafe(db) {
		// 	t.Error("expected db to be safe but it isn't")
		// }
		// nstmt, err = db.PrepareNamed(`SELECT * FROM person WHERE first_name != :name`)
		// if err != nil {
		// 	t.Fatal(err)
		// }
		// // it should be safe
		// if isUnsafe(nstmt) {
		// 	t.Error("NamedStmt did not inherit safety")
		// }
		// nstmt.Unsafe()
		// if !isUnsafe(nstmt) {
		// 	t.Error("expected newly unsafed NamedStmt to be unsafe")
		// }
		// pps = []PersonPlus{}
		// err = nstmt.Select(&pps, map[string]interface{}{"name": "Jason"})
		// if err != nil {
		// 	t.Fatal(err)
		// }
		// if len(pps) != 1 {
		// 	t.Errorf("Expected 1 person back, got %d", len(pps))
		// }

	})
}

type Person struct {
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string
	AddedAt   time.Time `db:"added_at"`
}

func TestNilReceiver(t *testing.T) {
	RunWithSchema(defaultSchema, t, func(db *DB, t *testing.T, now string) {
		loadDefaultFixture(db, t)
		var p *Person
		err := db.Get(p, "SELECT * FROM person LIMIT 1")
		if err == nil {
			t.Error("Expected error when getting into nil struct ptr.")
		}
		var pp *[]Person
		err = db.Select(pp, "SELECT * FROM person")
		if err == nil {
			t.Error("Expected an error when selecting into nil slice ptr.")
		}
	})
}

func TestNamedQuery(t *testing.T) {
	var schema = Schema{
		create: `
			drop table if exists person;
			drop table if exists jsperson;
			drop table if exists place;
			drop table if exists placeperson;
			CREATE TABLE place (
				id integer PRIMARY KEY,
				name varchar(100) NULL
			);
			CREATE TABLE person (
				first_name varchar(100) NULL,
				last_name varchar(100) NULL,
				email varchar(100) NULL
			);
			CREATE TABLE placeperson (
				first_name varchar(100) NULL,
				last_name varchar(100) NULL,
				email varchar(100) NULL,
				place_id integer NULL
			);
			CREATE TABLE jsperson (
				"FIRST" varchar(100) NULL,
				last_name varchar(100) NULL,
				"EMAIL" varchar(100) NULL
			);`,
		drop: `
			drop table person;
			drop table jsperson;
			drop table place;
			drop table placeperson;
			`,
	}

	RunWithSchema(schema, t, func(db *DB, t *testing.T, now string) {
		type Person struct {
			FirstName sql.NullString `db:"first_name"`
			LastName  sql.NullString `db:"last_name"`
			Email     sql.NullString
		}

		p := Person{
			FirstName: sql.NullString{String: "ben", Valid: true},
			LastName:  sql.NullString{String: "doe", Valid: true},
			Email:     sql.NullString{String: "ben@doe.com", Valid: true},
		}

		q1 := `INSERT INTO person (first_name, last_name, email) VALUES (:first_name, :last_name, :email)`
		_, err := db.NamedExec(q1, p)
		if err != nil {
			log.Fatal(err)
		}

		p2 := &Person{}
		rows, err := db.NamedQuery("SELECT * FROM person WHERE first_name=:first_name", p)
		if err != nil {
			log.Fatal(err)
		}
		for rows.Next() {
			err = rows.StructScan(p2)
			if err != nil {
				t.Error(err)
			}
			if p2.FirstName.String != "ben" {
				t.Error("Expected first name of `ben`, got " + p2.FirstName.String)
			}
			if p2.LastName.String != "doe" {
				t.Error("Expected first name of `doe`, got " + p2.LastName.String)
			}
		}

		// these are tests for #73;  they verify that named queries work if you've
		// changed the db mapper.  This code checks both NamedQuery "ad-hoc" style
		// queries and NamedStmt queries, which use different code paths internally.
		old := (*db).Mapper

		type JSONPerson struct {
			FirstName sql.NullString `json:"FIRST"`
			LastName  sql.NullString `json:"last_name"`
			Email     sql.NullString
		}

		jp := JSONPerson{
			FirstName: sql.NullString{String: "ben", Valid: true},
			LastName:  sql.NullString{String: "smith", Valid: true},
			Email:     sql.NullString{String: "ben@smith.com", Valid: true},
		}

		db.Mapper = reflectx.NewMapperFunc("json", strings.ToUpper)

		// prepare queries for case sensitivity to test our ToUpper function.
		// postgres and sqlite accept "", but mysql uses ``;  since Go's multi-line
		// strings are `` we use "" by default and swap out for MySQL
		pdb := func(s string, db *DB) string {
			if db.DriverName() == "mysql" {
				return strings.Replace(s, `"`, "`", -1)
			}
			return s
		}

		q1 = `INSERT INTO jsperson ("FIRST", last_name, "EMAIL") VALUES (:FIRST, :last_name, :EMAIL)`
		_, err = db.NamedExec(pdb(q1, db), jp)
		if err != nil {
			t.Fatal(err, db.DriverName())
		}

		// Checks that a person pulled out of the db matches the one we put in
		check := func(t *testing.T, rows *Rows) {
			jp = JSONPerson{}
			for rows.Next() {
				err = rows.StructScan(&jp)
				if err != nil {
					t.Error(err)
				}
				if jp.FirstName.String != "ben" {
					t.Errorf("Expected first name of `ben`, got `%s` (%s) ", jp.FirstName.String, db.DriverName())
				}
				if jp.LastName.String != "smith" {
					t.Errorf("Expected LastName of `smith`, got `%s` (%s)", jp.LastName.String, db.DriverName())
				}
				if jp.Email.String != "ben@smith.com" {
					t.Errorf("Expected first name of `doe`, got `%s` (%s)", jp.Email.String, db.DriverName())
				}
			}
		}

		ns, err := db.PrepareNamed(pdb(`
			SELECT * FROM jsperson
			WHERE
				"FIRST"=:FIRST AND
				last_name=:last_name AND
				"EMAIL"=:EMAIL
		`, db))

		if err != nil {
			t.Fatal(err)
		}
		rows, err = ns.Queryx(jp)
		if err != nil {
			t.Fatal(err)
		}

		check(t, rows)

		// Check exactly the same thing, but with db.NamedQuery, which does not go
		// through the PrepareNamed/NamedStmt path.
		rows, err = db.NamedQuery(pdb(`
			SELECT * FROM jsperson
			WHERE
				"FIRST"=:FIRST AND
				last_name=:last_name AND
				"EMAIL"=:EMAIL
		`, db), jp)
		if err != nil {
			t.Fatal(err)
		}

		check(t, rows)

		db.Mapper = old

		// Test nested structs
		type Place struct {
			ID   int
			Name sql.NullString
		}
		type PlacePerson struct {
			FirstName sql.NullString  `db:"first_name"`
			LastName  sql.NullString  `db:"last_name"`
			Email     sql.NullString
			Place     Place
		}

		pl := Place{
			Name: sql.NullString{String: "myplace", Valid: true},
		}

		pp := PlacePerson{
			FirstName: sql.NullString{String: "ben", Valid: true},
			LastName:  sql.NullString{String: "doe", Valid: true},
			Email:     sql.NullString{String: "ben@doe.com", Valid: true},
		}

		q2 := `INSERT INTO place (id, name) VALUES (1, :name)`
		_, err = db.NamedExec(q2, pl)
		if err != nil {
			log.Fatal(err)
		}

		id := 1
		pp.Place.ID = id

		q3 := `INSERT INTO placeperson (first_name, last_name, email, place_id) VALUES (:first_name, :last_name, :email, :place.id)`
		_, err = db.NamedExec(q3, pp)
		if err != nil {
			log.Fatal(err)
		}

		pp2 := &PlacePerson{}
		rows, err = db.Unsafe().NamedQuery(`
			SELECT
				first_name,
				last_name,
				email,
				place.id   as "place﹒id",
				place.name as "place﹒name"
			FROM placeperson
			INNER JOIN place ON place.id = placeperson.place_id
			WHERE
				place.id=:place.id`, pp)
		if err != nil {
			log.Fatal(err)
		}
		for rows.Next() {
			err = rows.StructScan(pp2)

			if err != nil {
				t.Error(err)
			}
			if pp2.FirstName.String != "ben" {
				t.Error("Expected first name of `ben`, got " + pp2.FirstName.String)
			}
			if pp2.LastName.String != "doe" {
				t.Error("Expected first name of `doe`, got " + pp2.LastName.String)
			}
			if pp2.Place.Name.String != "myplace" {
				t.Error("Expected place name of `myplace`, got " + pp2.Place.Name.String)
			}
			if pp2.Place.ID != pp.Place.ID {
				t.Errorf("Expected place name of %v, got %v", pp.Place.ID, pp2.Place.ID)
			}
		}
	})
}
