package intersystems

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log"
	"net"
	"unicode"

	"github.com/caretdev/go-irisnative/src/connection"

	_ "io"
	_ "math"
	_ "reflect"
	_ "strconv"
	_ "strings"
	_ "time"
	_ "unsafe"
)

var (
	ErrCouldNotDetectUsername = errors.New("intersystems: Could not detect default username. Please provide one explicitly")
	errNoRowsAffected         = errors.New("no RowsAffected available after the empty statement")
	errNoLastInsertID         = errors.New("no LastInsertId available after the empty statement")
)

var (
	_ driver.Driver = Driver{}
)

type values map[string]string

// Driver implements database/sql/driver.Driver.
type Driver struct{}

func (d Driver) Open(name string) (driver.Conn, error) {
	return Open(name)
}

func init() {
	sql.Register("intersystems", &Driver{})
}

func Open(dsn string) (_ driver.Conn, err error) {
	log.Println("Open")
	c, err := NewConnector(dsn)
	if err != nil {
		return nil, err
	}
	return c.open(context.Background())
}

type conn struct {
	c connection.Connection
}

func (c *Connector) open(ctx context.Context) (cn *conn, err error) {
	log.Println("Begin")
	o := make(values)
	for k, v := range c.opts {
		o[k] = v
	}
	host := o["host"]
	addr := net.JoinHostPort(host, o["port"])
	namespace := o["namespace"]
	login := o["user"]
	password := o["password"]

	cn = &conn{}

	cn.c, err = connection.Connect(addr, namespace, login, password)
	if err != nil {
		return nil, err
	}
	return cn, nil
}

func (cn *conn) Begin() (_ driver.Tx, err error) {
	log.Println("Begin")
	return cn, nil
}

func (cn *conn) Close() (err error) {
	log.Println("ConnClose")
	cn.c.Disconnect()
	return nil
}

func (cn *conn) Prepare(q string) (_ driver.Stmt, err error) {
	log.Println("Prepare")
	st := &stmt{cn: cn, name: ""}
	return st, nil
}

func (cn *conn) Commit() (err error) {
	log.Println("Commit")
	return nil
}

func (cn *conn) Rollback() (err error) {
	log.Println("Rollback")
	return nil
}

func (cn *conn) Exec(query string, args []driver.Value) (res driver.Result, err error) {
	log.Println("ConnExec: " + query)
	res = emptyRows
	return res, nil
}

func (cn *conn) Query(query string, args []driver.Value) (driver.Rows, error) {
	log.Println("ConnQuery: ", query)
	log.Println("ConnQueryArgs: ", args)

	parameters := make([]interface{}, len(args))
	for i, a := range args {
		parameters[i] = a
	}
	rs, _ := cn.c.DirectQuery(query, parameters...)

	return &rows{
		cn: cn,
		rs: rs,
	}, nil
}

func parseOpts(name string, o values) error {
	s := newScanner(name)

	for {
		var (
			keyRunes, valRunes []rune
			r                  rune
			ok                 bool
		)

		if r, ok = s.SkipSpaces(); !ok {
			break
		}

		// Scan the key
		for !unicode.IsSpace(r) && r != '=' {
			keyRunes = append(keyRunes, r)
			if r, ok = s.Next(); !ok {
				break
			}
		}

		// Skip any whitespace if we're not at the = yet
		if r != '=' {
			r, ok = s.SkipSpaces()
		}

		// The current character should be =
		if r != '=' || !ok {
			return fmt.Errorf(`missing "=" after %q in connection info string"`, string(keyRunes))
		}

		// Skip any whitespace after the =
		if r, ok = s.SkipSpaces(); !ok {
			// If we reach the end here, the last value is just an empty string as per libpq.
			o[string(keyRunes)] = ""
			break
		}

		if r != '\'' {
			for !unicode.IsSpace(r) {
				if r == '\\' {
					if r, ok = s.Next(); !ok {
						return fmt.Errorf(`missing character after backslash`)
					}
				}
				valRunes = append(valRunes, r)

				if r, ok = s.Next(); !ok {
					break
				}
			}
		} else {
		quote:
			for {
				if r, ok = s.Next(); !ok {
					return fmt.Errorf(`unterminated quoted string literal in connection string`)
				}
				switch r {
				case '\'':
					break quote
				case '\\':
					r, _ = s.Next()
					fallthrough
				default:
					valRunes = append(valRunes, r)
				}
			}
		}

		o[string(keyRunes)] = string(valRunes)
	}

	return nil
}
