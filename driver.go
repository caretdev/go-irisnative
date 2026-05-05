package intersystems

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"unicode"

	"github.com/caretdev/go-irisnative/src/connection"
)

var (
	ErrCouldNotDetectUsername = errors.New("intersystems: Could not detect default username. Please provide one explicitly")
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
	sql.Register("iris", &Driver{})
}

func Open(dsn string) (_ driver.Conn, err error) {
	c, err := NewConnector(dsn)
	if err != nil {
		return nil, err
	}
	return c.open(context.Background())
}

type conn struct {
	c  connection.Connection
	tx bool
}

func (c *Connector) open(ctx context.Context) (cn *conn, err error) {
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

	// Set maxRowsPerFetch if specified in DSN
	if maxRows, ok := o["max_rows"]; ok {
		var rows int
		_, err = fmt.Sscanf(maxRows, "%d", &rows)
		if err == nil && rows > 0 {
			cn.c.SetMaxRowsPerFetch(rows)
		}
	}

	// Set queryTimeout if specified in DSN
	if timeout, ok := o["query_timeout"]; ok {
		var to int
		_, err = fmt.Sscanf(timeout, "%d", &to)
		if err == nil && to >= 0 {
			cn.c.SetQueryTimeout(to)
		}
	}

	return cn, nil
}

func (cn *conn) Begin() (driver.Tx, error) {
	return cn.c.BeginTx(driver.TxOptions{})
}

func (cn *conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return cn.c.BeginTx(opts)
}

func (cn *conn) Close() (err error) {
	cn.c.Disconnect()
	return nil
}

func (cn *conn) Prepare(q string) (st driver.Stmt, err error) {
	return cn.c.Prepare(q)
}

func (cn *conn) Commit() error {
	if !cn.tx {
		return errors.New("transaction already closed")
	}
	cn.tx = false
	cn.c.Commit()
	return nil
}

func (cn *conn) Rollback() error {
	if !cn.tx {
		return errors.New("transaction already closed")
	}
	cn.tx = false
	cn.c.Rollback()
	return nil
}

func (cn *conn) Exec(query string, args []driver.NamedValue) (res driver.Result, err error) {
	parameters := make([]interface{}, len(args))
	for i, a := range args {
		parameters[i] = a
	}
	_, err = cn.c.DirectUpdate(query, parameters...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (cn *conn) Query(query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	parameters := make([]interface{}, len(args))
	for i, a := range args {
		parameters[i] = a
	}
	// var rs *connection.ResultSet
	_, err = cn.c.Query(query, parameters...)
	if err != nil {
		return nil, err
	}
	// rows = &connection.Rows{
	// 	cn: cn.c,
	// 	rs: rs,
	// }
	return
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
