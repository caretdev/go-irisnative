package connection

import (
	"database/sql/driver"
	"errors"
	"reflect"
	"strings"
	"time"
)

var (
	errNoRowsAffected = errors.New("no RowsAffected available after the empty statement")
	errNoLastInsertID = errors.New("no LastInsertId available after the empty statement")
)

type Result struct {
	cn       *Connection
	affected int64
}

func (r Result) LastInsertId() (lastId int64, err error) {
	var rs *ResultSet
	rs, err = r.cn.DirectQuery("SELECT LAST_IDENTITY()")
	if err != nil {
		return
	}
	row, err := rs.Next()
	if err != nil {
		return
	}
	lastId = int64(row[0].(int))
	return
}

func (r Result) RowsAffected() (int64, error) {
	return r.affected, nil
}

type Rows struct {
	cn *Connection
	rs *ResultSet
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

func (r *Rows) Close() error {
	return nil
}

func (r *Rows) Columns() []string {
	if r.rs == nil {
		return []string{}
	}
	columns := r.rs.Columns()
	colNames := make([]string, len(columns))
	for k, c := range columns {
		colname := c.Name()
		// tricking IRIS
		colname = strings.ReplaceAll(colname, "﹒", ".")
		colNames[k] = colname
	}
	// fmt.Printf("Columns: %#v\n", colNames)
	return colNames
}

func (r *Rows) ColumnTypeDatabaseTypeName(index int) string {
	column, ok := r.column(index)
	if !ok {
		return ""
	}
	return databaseTypeName(SQLTYPE(column.column_type))
}

func (r *Rows) ColumnTypeScanType(index int) reflect.Type {
	column, ok := r.column(index)
	if !ok {
		return reflect.TypeOf("")
	}
	return scanType(SQLTYPE(column.column_type))
}

func (r *Rows) ColumnTypeNullable(index int) (nullable, ok bool) {
	column, ok := r.column(index)
	if !ok {
		return true, false
	}
	switch column.nullable {
	case 0:
		return false, true
	case 1:
		return true, true
	default:
		return true, false
	}
}

func (r *Rows) ColumnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	column, ok := r.column(index)
	if !ok {
		return 0, 0, false
	}
	switch SQLTYPE(column.column_type) {
	case NUMERIC, DECIMAL, FLOAT, REAL, DOUBLE:
		return int64(column.precision), int64(column.scale), true
	default:
		return 0, 0, false
	}
}

func (r *Rows) ColumnTypeLength(index int) (length int64, ok bool) {
	column, ok := r.column(index)
	if !ok || column.precision <= 0 {
		return 0, false
	}
	switch SQLTYPE(column.column_type) {
	case CHAR, VARCHAR, LONGVARCHAR, WCHAR, WVARCHAR, WLONGVARCHAR, GUID, BINARY, VARBINARY, LONGVARBINARY:
		return int64(column.precision), true
	default:
		return 0, false
	}
}

func (r *Rows) column(index int) (Column, bool) {
	if r == nil || r.rs == nil || index < 0 || index >= len(r.rs.columns) {
		return Column{}, false
	}
	return r.rs.columns[index], true
}

func databaseTypeName(colType SQLTYPE) string {
	switch colType {
	case GUID:
		return "GUID"
	case WLONGVARCHAR:
		return "WLONGVARCHAR"
	case WVARCHAR:
		return "WVARCHAR"
	case WCHAR:
		return "WCHAR"
	case BIT:
		return "BIT"
	case TINYINT:
		return "TINYINT"
	case BIGINT:
		return "BIGINT"
	case LONGVARBINARY:
		return "LONGVARBINARY"
	case VARBINARY:
		return "VARBINARY"
	case BINARY:
		return "BINARY"
	case LONGVARCHAR:
		return "LONGVARCHAR"
	case CHAR:
		return "CHAR"
	case NUMERIC:
		return "NUMERIC"
	case DECIMAL:
		return "DECIMAL"
	case INTEGER:
		return "INTEGER"
	case SMALLINT:
		return "SMALLINT"
	case FLOAT:
		return "FLOAT"
	case REAL:
		return "REAL"
	case DOUBLE:
		return "DOUBLE"
	case DATE, TYPE_DATE:
		return "DATE"
	case TIME, TYPE_TIME:
		return "TIME"
	case TIMESTAMP, TYPE_TIMESTAMP, TIMESTAMP_POSIX:
		return "TIMESTAMP"
	default:
		return ""
	}
}

func scanType(colType SQLTYPE) reflect.Type {
	switch colType {
	case BIT:
		return reflect.TypeOf(false)
	case TINYINT, SMALLINT, INTEGER, BIGINT:
		return reflect.TypeOf(int64(0))
	case NUMERIC, DECIMAL, FLOAT, REAL, DOUBLE:
		return reflect.TypeOf(float64(0))
	case TIMESTAMP, TYPE_TIMESTAMP, TIMESTAMP_POSIX:
		return reflect.TypeOf(time.Time{})
	case BINARY, VARBINARY, LONGVARBINARY:
		return reflect.TypeOf([]byte{})
	default:
		return reflect.TypeOf("")
	}
}

func (r *Rows) Next(dest []driver.Value) (err error) {
	row, err := r.rs.Next()
	if err != nil {
		return err
	}
	for i := range dest {
		dest[i] = row[i]
	}
	// fmt.Printf("RowsNext: %#v\n", dest)
	return nil
}
