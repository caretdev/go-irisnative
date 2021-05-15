package intersystems

import (
	"database/sql/driver"
	"log"
)

type stmt struct {
	cn   *conn
	name string
}

func (st *stmt) Query(v []driver.Value) (r driver.Rows, err error) {
	log.Println("StmtQuery")
	return nil, nil
}

func (st *stmt) Exec(v []driver.Value) (res driver.Result, err error) {
	log.Println("StmtExec")
	res = emptyRows
	return res, nil
}

func (st *stmt) Close() (err error) {
	log.Println("StmtClose")
	return nil
}

func (st *stmt) NumInput() int {
	return -1
}
