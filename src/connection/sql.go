package connection

import (
	"io"
	"log"

	"github.com/caretdev/go-irisnative/src/list"
)

type StatementFeature struct {
	featureOption   int
	msgCount        int
	maxRowItemCount int
}

type Column struct {
	name              string
	column_type       int
	precision         int
	scale             int
	nullable          int
	slot_position     int
	label             string
	table_name        string
	schema            string
	catalog           string
	is_auto_increment bool
	is_case_sensitive bool
	is_currency       bool
	is_read_only      bool
	is_row_id         bool
}

func (c Column) Name() string {
	return c.name
}

type ResultSet struct {
	c       *Connection
	columns []Column
	sf      StatementFeature
	count   int
	data    []byte
	offset  uint
}

func (rs ResultSet) Columns() []Column {
	return rs.columns
}

func statementFeature(msg *Message) StatementFeature {
	featureOption := 0
	msgCount := 0
	maxRowItemCount := 0
	msg.Get(&featureOption)
	if featureOption == 2 {
		msg.Get(&msgCount)
	}
	if featureOption == 1 || featureOption == 2 {
		msg.Get(&maxRowItemCount)
	}
	return StatementFeature{
		featureOption,
		msgCount,
		maxRowItemCount,
	}
}

type Value interface{}

// type ResultSetRow struct{}

func (rs *ResultSet) fetchMoreData() bool {
	msg := NewMessage(FETCH_DATA)
	_, err := rs.c.conn.Write(msg.Dump(rs.c.count()))
	if err != nil {
		panic(err)
	}
	msg, err = ReadMessage(rs.c.conn)
	if err != nil {
		panic(err)
	}

	rs.data = msg.data
	rs.offset = 0
	return len(msg.data) > 0
}

func (rs *ResultSet) Next() ([]Value, error) {
	if rs.offset >= uint(len(rs.data)) && !rs.fetchMoreData() {
		return nil, io.EOF
	}
	row := make([]Value, rs.count)
	data := rs.data
	count := rs.count
	var offset uint = rs.offset
	if rs.sf.featureOption == 1 {
		li := list.GetListItem(data, &rs.offset)
		li.Get(&data)
		offset = 0
		count = rs.sf.maxRowItemCount
	}
	vals := make([]list.ListItem, count)
	for i := 0; i < count; i++ {
		li := list.GetListItem(data, &offset)
		vals[i] = li
	}
	if rs.sf.featureOption != 1 {
		rs.offset = offset
	}
	for i, c := range rs.columns {
		li := vals[c.slot_position]
		switch c.column_type {
		case 12:
			var value string
			li.Get(&value)
			row[i] = value
		case -6, 4, 5:
			var value int
			li.Get(&value)
			row[i] = value
		case -7:
			var value bool
			li.Get(&value)
			row[i] = value
		case 2:
			var value float32
			li.Get(&value)
			row[i] = value
		case 8:
			var value float64
			li.Get(&value)
			row[i] = value
		default:
			var value string
			li.Get(&value)
			row[i] = value
		}
	}
	return row, nil
}

func getColumns(msg *Message, statementFeature StatementFeature) []Column {
	cnt := 0
	msg.Get(&cnt)
	columns := make([]Column, cnt)
	for i := 0; i < cnt; i++ {
		column := Column{}
		msg.Get(&column.name)
		msg.Get(&column.column_type)
		switch column.column_type {
		case 9:
			column.column_type = 91
		case 10:
			column.column_type = 92
		case 11:
			column.column_type = 93
		}
		msg.Get(&column.precision)
		msg.Get(&column.scale)
		msg.Get(&column.nullable)
		msg.Get(&column.label)
		msg.Get(&column.table_name)
		msg.Get(&column.schema)
		msg.Get(&column.catalog)
		additional := ""
		msg.Get(&additional)
		if statementFeature.featureOption&0x01 == 1 {
			msg.Get(&column.slot_position)
			column.slot_position -= 1
		} else {
			column.slot_position = i
		}
		column.is_auto_increment = additional[0] == 0x01
		column.is_case_sensitive = additional[1] == 0x01
		column.is_currency = additional[2] == 0x01
		column.is_read_only = additional[3] == 0x01
		if len(additional) >= 12 {
			column.is_row_id = additional[11] == 0x01
		}
		columns[i] = column
	}
	return columns
}

func parameterInfo(msg *Message) {
	cnt := 0
	msg.Get(&cnt)
	flag := 0
	msg.Get(&flag)
}

func writeParameters(msg *Message, args ...interface{}) {
	msg.Set(len(args))
	for range args {
		msg.Set(99)
		msg.Set(4)
	}

	msg.Set(1) // parameterSets
	msg.Set(len(args))
	for _, arg := range args {
		msg.Set(arg)
	}
}

func (c *Connection) DirectQuery(sqlText string, args ...interface{}) (*ResultSet, error) {
	log.Print("DirectQuery", sqlText)
	msg := NewMessage(DIRECT_QUERY)
	msg.SetSQLText(sqlText)
	writeParameters(&msg, args...)
	msg.Set(0) // Query timeout
	msg.Set(0) // Max rows

	_, err := c.conn.Write(msg.Dump(c.count()))
	if err != nil {
		return nil, err
	}
	msg, err = ReadMessage(c.conn)
	if err != nil {
		return nil, err
	}
	log.Printf("%#v", msg)
	statementFeature := statementFeature(&msg)
	columns := getColumns(&msg, statementFeature)
	parameterInfo((&msg))
	rs := &ResultSet{
		c:       c,
		sf:      statementFeature,
		columns: columns,
		count:   len(columns),
	}

	msg, err = ReadMessage(c.conn)
	if err != nil {
		return nil, err
	}

	msg.GetRaw(&rs.data)

	return rs, nil
}
