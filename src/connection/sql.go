package connection

import (
	"database/sql/driver"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/caretdev/go-irisnative/src/list"
)

const timeLaylout = "2006-01-02 15:04:05.000000000"
const timeLayloutShort = "2006-01-02 15:04:05"

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

type SQLTYPE int16

const (
	GUID            SQLTYPE = -11
	WLONGVARCHAR    SQLTYPE = -10
	WVARCHAR        SQLTYPE = -9
	WCHAR           SQLTYPE = -8
	BIT             SQLTYPE = -7
	TINYINT         SQLTYPE = -6
	BIGINT          SQLTYPE = -5
	LONGVARBINARY   SQLTYPE = -4
	VARBINARY       SQLTYPE = -3
	BINARY          SQLTYPE = -2
	LONGVARCHAR     SQLTYPE = -1
	CHAR            SQLTYPE = 1
	NUMERIC         SQLTYPE = 2
	DECIMAL         SQLTYPE = 3
	INTEGER         SQLTYPE = 4
	SMALLINT        SQLTYPE = 5
	FLOAT           SQLTYPE = 6
	REAL            SQLTYPE = 7
	DOUBLE          SQLTYPE = 8
	DATE            SQLTYPE = 9
	TIME            SQLTYPE = 10
	TIMESTAMP       SQLTYPE = 11
	VARCHAR         SQLTYPE = 12
	TYPE_DATE       SQLTYPE = 91
	TYPE_TIME       SQLTYPE = 92
	TYPE_TIMESTAMP  SQLTYPE = 93
	DATE_HOROLOG    SQLTYPE = 1091
	TIME_HOROLOG    SQLTYPE = 1092
	TIMESTAMP_POSIX SQLTYPE = 1093
)

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
	sqlCode int16
}

type SQLError struct {
	SQLCode int16
	Message string
}

func (e *SQLError) Error() string {
	return fmt.Sprintf("Error Code: %d, Message: %s", e.SQLCode, e.Message)
}

// func SQLError(code int) error {
// 	return &SQLError{SQLCode: code}
// }

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

func fromODBC(coltype SQLTYPE, li list.ListItem) (result interface{}, err error) {
	result = nil
	if li.IsNull() || li.IsEmpty() {
		return
	}
	switch coltype {
	case VARCHAR:
		if li.DataLength() == 0 {
			return
		}
		var value string
		li.Get(&value)
		if value == "\x00" {
			value = ""
		}
		result = value
	case INTEGER, TINYINT, SMALLINT:
		var value int
		li.Get(&value)
		result = value
	case BIGINT:
		var value int64
		li.Get(&value)
		result = value
	case BIT:
		var value bool
		li.Get(&value)
		result = value
	case FLOAT:
		var value float32
		li.Get(&value)
		result = value
	case DOUBLE:
		var value float64
		li.Get(&value)
		result = value
	case TIMESTAMP_POSIX:
		if li.DataLength() == 0 {
			return
		}
		if li.Type() == list.LISTITEM_STRING {
			var strval string
			li.Get(&strval)
			result, err = time.Parse(timeLaylout, strval)
			return
		}
		var value int64
		li.Get(&value)
		if value > 0 {
			value ^= 0x1000000000000000
		} else {
			value |= 0x6000000000000000
		}
		seconds := value / 1000000
		nano := value % 1000000 * 1000
		result = time.Unix(seconds, nano).In(time.Local)
	case VARBINARY:
		// var value []uint8
		var value string
		li.Get(&value)
		fmt.Printf("VARBINARY: %#v\n", value)
	case TYPE_TIMESTAMP:
		var strval string
		li.Get(&strval)
		result, err = time.Parse(timeLayloutShort, strval)
	default:
		var value string
		li.Get(&value)
		fmt.Printf("fromODBC: invalid type: %v - %#v - %#v", coltype, li, value)
		result = value
	}
	return
}

func (rs *ResultSet) Next() ([]Value, error) {
	if rs == nil || (rs.sqlCode != 0 && rs.sqlCode != 100) {
		return nil, io.EOF
	}
	if rs.offset >= uint(len(rs.data)) && (rs.sqlCode == 100 || !rs.fetchMoreData()) {
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
	var err error
	for i, c := range rs.columns {
		li := vals[c.slot_position]
		row[i], err = fromODBC(SQLTYPE(c.column_type), li)
		if err != nil {
			return nil, err
		}
		// fmt.Printf("col: %s: %d; %#v - %#v\n", c.name, c.column_type, row[i], li)
	}
	// fmt.Printf("row: %#v\n", row)
	return row, nil
}

func (c *Connection) getErrorInfo(sqlCode int16) string {
	msg := NewMessage(GET_SERVER_ERROR)
	msg.Set(sqlCode)
	_, err := c.conn.Write(msg.Dump(c.count()))
	if err != nil {
		panic(err)
	}
	msg, err = ReadMessage(c.conn)
	if err != nil {
		panic(err)
	}
	var sqlMessage string
	msg.Get(&sqlMessage)
	return sqlMessage
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

func toODBC(value interface{}) interface{} {
	var val interface{}
	switch v := value.(type) {
	case *string:
		val = *v
	case string:
		val = v
		if v == "" {
			val = "\x00"
		}
	case nil:
		val = ""
	case bool:
		if v {
			val = 1
		} else {
			val = 0
		}
	case time.Time:
		val = v.UTC().Format(timeLaylout)
	case int, int8, int16, int32, int64:
		val = v
	case float32, float64:
		val = v
	case []uint8:
		val = v
	default:
		fmt.Printf("unsupported type: %T\n", v)
		val = fmt.Sprintf("%v", v)
	}
	return val
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
		msg.Set(toODBC(arg))
	}
}

func (c *Connection) Query(sqlText string, args ...interface{}) (rs *ResultSet, err error) {
	queries := strings.Split(sqlText, ";\n")
	if len(queries) == 2 {
		sqlText = queries[0]
		_, err = c.DirectUpdate(sqlText, args...)
		if err != nil {
			return
		}

		sqlText = queries[1]
		args = []interface{}{}
	}
	rs, err = c.DirectQuery(sqlText, args...)
	if err != nil {
		return
	}
	return
}

func (c *Connection) DirectQuery(sqlText string, args ...interface{}) (*ResultSet, error) {
	sqlText, _, args = FormatQuery(sqlText, args...)
	// fmt.Printf("DirectQuery: %s; %#v\n", sqlText, args)

	var statementId = c.statementId()
	msg := NewMessage(DIRECT_QUERY)
	msg.header.SetStatementId(statementId)
	msg.SetSQLText(sqlText)
	writeParameters(&msg, args...)
	msg.Set(10) // Query timeout
	msg.Set(0)  // Max rows

	_, err := c.conn.Write(msg.Dump(c.count()))
	if err != nil {
		return nil, err
	}
	msg, err = ReadMessage(c.conn)
	if err != nil {
		return nil, err
	}
	sqlCode := int16(msg.GetStatus())
	if sqlCode != 0 && sqlCode != 100 {
		return nil, &SQLError{SQLCode: sqlCode, Message: c.getErrorInfo(sqlCode)}
	}
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
	rs.sqlCode = int16(msg.GetStatus())
	if err != nil {
		return nil, err
	}

	msg.GetRaw(&rs.data)

	return rs, nil
}

func (m Message) debug() string {
	var sb strings.Builder
	for i, b := range m.data {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(strconv.Itoa(int(b)))
	}
	return fmt.Sprintf("$char(%s)", sb.String())
}

func FormatQuery(sqlText string, args ...interface{}) (string, int, []interface{}) {
	var count int
	for i := range args {
		count++
		sqlText = strings.Replace(sqlText, "?", fmt.Sprintf(" :%%qpar(%d) ", i+1), 1)
		if !strings.Contains(sqlText, "?") {
			break
		}
	}
	return sqlText, count, args
}

func (c *Connection) Exec(sqlText string, args ...interface{}) (res *Result, err error) {
	queries := strings.Split(sqlText, ";\n")
	var onConflict = ""
	if len(queries) == 2 {
		sqlText = queries[0]
		onConflict = strings.Split(queries[1], "-- ")[1]
		if strings.Contains(onConflict, "ON CONFLICT UPDATE") {
			// fmt.Printf("------\n%s\n%#v\n------\n", sqlText, args)
			sqlText = strings.Replace(sqlText, "INSERT INTO", "INSERT OR UPDATE", 1)
			onConflict = ""
		}
	}
	res, err = c.DirectUpdate(sqlText, args...)
	if err != nil {
		if strings.Contains(onConflict, "ON CONFLICT DO NOTHING") {
			res = &Result{cn: c, affected: 0}
			err = nil
			return
		}
	}
	return
}

func (c *Connection) DirectUpdate(sqlText string, args ...interface{}) (*Result, error) {
	var batchSize int
	sqlText, batchSize, args = FormatQuery(sqlText, args...)
	// fmt.Printf("DirectUpdate: %s; %#v\n", sqlText, args)
	var batches = 1
	if batchSize > 0 {
		batches = len(args) / batchSize
	}
	var addToCache = false
	var statementId = c.statementId()
	var executeMany = false
	var optFastInsert = false
	var rowsAffected int64 = 0
	var identityColumn = false
	var defaults = []interface{}{}
	for i := 1; i <= batches; i++ {
		if i > 1 && executeMany {
			break
		}
		var msg Message
		if !addToCache {
			msg = NewMessage(DIRECT_UPDATE)
			msg.SetSQLText(sqlText)
			msg.Set(batchSize)
			for j := 0; j < batchSize; j++ {
				msg.Set(99)
				msg.Set(1)
			}
			// msg.Set(len(args))
			// for range args {
			// 	msg.Set(99)
			// 	msg.Set(1)
			// }
		} else {
			msg = NewMessage(PREPARED_UPDATE)
		}
		if addToCache && !executeMany && optFastInsert {
			msg.AddRaw([]byte{1, 0, 0, 0})
			msg.Set("")
			msg.Set(0)
			if identityColumn {
				msg.Set(2)
				msg.Set("")
			} else {
				msg.Set(1)
			}
			var batch []interface{} = make([]interface{}, batchSize)
			copy(batch, args)
			args = slices.Delete(args, 0, batchSize)
			var params []byte
			var item list.ListItem
			for _, arg := range batch {
				item = list.NewListItem(toODBC(arg))
				params = append(params, item.Dump()...)
			}
			for _, arg := range defaults {
				item = list.NewListItem(toODBC(arg))
				params = append(params, item.Dump()...)
			}
			msg.Set(params)
		} else {
			msg.Set("")
			msg.Set(0)
			if executeMany {
				msg.Set(batches)
				for k := 0; k < batches; k++ {
					msg.Set(batchSize)
					for j := 0; j < batchSize; j++ {
						var idx = (k * batchSize) + j
						fmt.Printf("arg: %d:%d:%d\n", k, j, idx)
						msg.Set(toODBC(args[idx]))
					}
				}
			} else {
				var batch []interface{} = make([]interface{}, batchSize)
				copy(batch, args)
				args = slices.Delete(args, 0, batchSize)
				msg.Set(1)
				msg.Set(len(batch))
				for _, arg := range batch {
					msg.Set(toODBC(arg))
				}
			}
		}

		msg.header.SetStatementId(statementId)
		_, err := c.conn.Write(msg.Dump(c.count()))
		if err != nil {
			return nil, err
		}
		msg, err = ReadMessage(c.conn)
		if err != nil {
			return nil, err
		}
		sqlCode := int16(msg.GetStatus())
		if sqlCode != 0 && sqlCode != 100 {
			return nil, &SQLError{SQLCode: sqlCode, Message: c.getErrorInfo(sqlCode)}
		}
		if i == 1 {
			if c.IsOptionFastInsert() {
				stmtFeatureOption, _ := c.checkStatementFeature(&msg)
				optFastInsert = stmtFeatureOption&uint(OptionFastInsert) == uint(OptionFastInsert)
			}
			addToCache, identityColumn, defaults = c.getParameterInfo(&msg, optFastInsert)
		}
		var batchRows int64
		msg.Get(&batchRows)
		rowsAffected += batchRows
	}
	result := &Result{cn: c, affected: rowsAffected}
	return result, nil
}

func (c *Connection) checkStatementFeature(msg *Message) (featureOption uint, count uint) {
	count = 0
	var keyCount int
	msg.Get(&featureOption)
	if featureOption == uint(OptionFastSelect) || featureOption == uint(OptionFastInsert) {
		if featureOption == uint(OptionFastInsert) {
			msg.Get(&keyCount)
		}
		msg.Get(&count)
	}
	return
}

func (c *Connection) getParameterInfo(msg *Message, optFastInsert bool) (addToCache bool, identityColumn bool, defaults []interface{}) {
	var paramscnt int
	msg.Get(&paramscnt)
	var tablename string
	for i := 0; i < paramscnt; i++ {
		var (
			paramtype int
			precision int
			scale     int
			nullable  bool
			position  int
			someval1  string
			someval2  string
			colname   string
		)
		msg.Get(&paramtype)
		msg.Get(&precision)
		msg.Get(&scale)
		msg.GetAny()
		if optFastInsert {
			msg.Get(&nullable)
			msg.Get(&position)
			msg.Get(&someval1)
			msg.Get(&someval2)
			if i == 0 {
				msg.Get(&tablename)
			}
			msg.Get(&colname)
		}
	}
	var flag int
	defaults = []interface{}{}
	identityColumn = false
	msg.Get(&flag)
	addToCache = flag&0x1 == 0x1
	if optFastInsert {
		var paramsDefault []byte
		msg.Get(&paramsDefault)
		var offset uint = 0
		var li list.ListItem
		li = list.GetListItem(paramsDefault, &offset)
		identityColumn = li.IsEmpty()
		for {
			if uint(len(paramsDefault)) == offset {
				break
			}
			li = list.GetListItem(paramsDefault, &offset)
			if li.IsNull() {
				continue
			}
			var val string
			li.Get(&val)
			fmt.Printf("li: %#v\n", val)
			defaults = append(defaults, val)
		}
	}
	return
}

type Stmt struct {
	cn          *Connection
	sql         string
	closed      bool
	statementId int32
}

func (c *Connection) Prepare(query string) (*Stmt, error) {
	// msg := NewMessage(PREPARE)
	// msg.SetSQLText(query)
	// msg.Set(0)

	// _, err := c.conn.Write(msg.Dump(c.count()))
	// if err != nil {
	// 	return nil, err
	// }
	// msg, err = ReadMessage(c.conn)
	// if err != nil {
	// 	return nil, err
	// }
	// sqlCode := int16(msg.GetStatus())
	// if sqlCode != 0 && sqlCode != 100 {
	// 	return nil, &SQLError{SQLCode: sqlCode, Message: c.getErrorInfo(sqlCode)}
	// }

	st := &Stmt{cn: c, sql: query}
	return st, nil
}

func (st *Stmt) Exec(args []driver.Value) (res driver.Result, err error) {
	parameters := make([]interface{}, len(args))
	for i, a := range args {
		parameters[i] = a
	}
	res, err = st.cn.Exec(st.sql, parameters...)
	return
}

func (st *Stmt) Query(args []driver.Value) (rows driver.Rows, err error) {
	parameters := make([]interface{}, len(args))
	for i, a := range args {
		parameters[i] = a
	}
	var rs *ResultSet
	rs, err = st.cn.Query(st.sql, parameters...)
	// st.statementId = int32(st.cn.statementId())
	if err != nil {
		return nil, err
	}
	rows = &Rows{
		cn: st.cn,
		rs: rs,
	}
	return
}

func (st *Stmt) Close() (err error) {
	st.closed = true
	return nil
}

func (st *Stmt) NumInput() int {
	return -1
}
