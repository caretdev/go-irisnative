package intersystems

import (
	"database/sql/driver"
	"log"

	"github.com/caretdev/go-irisnative/src/connection"
)

type rows struct {
	cn *conn
	rs *connection.ResultSet
}

type noRows struct{}

var emptyRows noRows

var _ driver.Result = noRows{}

func (noRows) LastInsertId() (int64, error) {
	return 0, errNoLastInsertID
}

func (noRows) RowsAffected() (int64, error) {
	return 0, errNoRowsAffected
}


func (r *rows) Close() error {
	log.Println("RowsClose")
	return nil
}

func (r *rows) Columns() []string {
	log.Println("Columns")
	columns := r.rs.Columns()
	colNames := make([]string, len(columns))
	for k, c := range columns {
		colNames[k] = c.Name()
	}
	return colNames
}

func (r *rows) Next(dest []driver.Value) (err error) {
	// log.Println("Next")
	row, err := r.rs.Next()
	if err != nil {
		return err
	}
	for i := range dest {
		dest[i] = row[i]
	}
	return nil
}

