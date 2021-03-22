package connection

import (
	"testing"
	"time"

	"github.com/caretdev/go-irisnative/src/list"
	"github.com/stretchr/testify/assert"
)

func TestToODBC(t *testing.T) {
	assert.Equal(t, 0, toODBC(false))
	assert.Equal(t, 1, toODBC(true))
	assert.Equal(t, "test", toODBC("test"))
	assert.Equal(t, "", toODBC(nil))
	assert.Equal(t, "\x00", toODBC(""))
	assert.Equal(t, int(100), toODBC(int(100)))
	assert.Equal(t, "2025-12-25 10:20:30.123456789", toODBC(time.Date(2025, time.December, 25, 10, 20, 30, 123456789, time.UTC)))
	assert.Equal(t, "2025-09-16 21:07:58.043329000", toODBC(time.Date(2025, time.September, 16, 21, 7, 58, 43329000, time.UTC)))
}

func mustFromODBC(coltype SQLTYPE, li list.ListItem) (result interface{}) {
	var err error
	result, err = fromODBC(coltype, li)
	if err != nil {
		panic("Error in mustFromODBC")
	}
	return
}

func TestFromODBC(t *testing.T) {
	assert.Equal(t, "", mustFromODBC(VARCHAR, list.NewListItem("\x00")))
	assert.Equal(t, nil, mustFromODBC(VARCHAR, list.NewListItem(nil)))
	assert.Equal(t, nil, mustFromODBC(VARCHAR, list.NewListItem("")))
	assert.Equal(t, false, mustFromODBC(BIT, list.NewListItem(false)))
	assert.Equal(t, true, mustFromODBC(BIT, list.NewListItem(true)))
	assert.Equal(t, "test", mustFromODBC(VARCHAR, list.NewListItem("test")))
	assert.Equal(t,
		time.Date(2025, time.September, 16, 10, 20, 30, 123456000, time.UTC).Local(),
		mustFromODBC(TIMESTAMP_POSIX, list.NewListItem(1154679522636970432)),
	)
	assert.Equal(t,
		time.Date(2025, time.September, 17, 21, 7, 6, 967698000, time.UTC).Local(),
		mustFromODBC(TIMESTAMP_POSIX, list.NewListItem(1154679647833814674)),
	)
	assert.Equal(t,
		time.Date(1025, time.September, 16, 10, 20, 30, 0, time.UTC).Local(),
		mustFromODBC(TIMESTAMP_POSIX, list.NewListItem(-6947328004811081856)),
	)
	assert.Equal(t,
		"2025-09-16 10:20:30.100000000",
		toODBC(mustFromODBC(TIMESTAMP_POSIX, list.NewListItem(1154679522636946976))),
	)
	assert.Equal(t,
		"2025-09-16 10:20:30.120000000",
		toODBC(mustFromODBC(TIMESTAMP_POSIX, list.NewListItem(1154679522636966976))),
	)
	assert.Equal(t,
		"1025-09-16 10:20:30.000000000",
		toODBC(mustFromODBC(TIMESTAMP_POSIX, list.NewListItem(-6947328004811081856))),
	)
	assert.Equal(t,
		"2025-09-16 10:20:30.010000000",
		toODBC(mustFromODBC(TIMESTAMP_POSIX, list.NewListItem(1154679522636856976))),
	)
	assert.Equal(t,
		nil,
		mustFromODBC(TIMESTAMP_POSIX, list.NewListItem(nil)),
	)
	assert.Equal(t,
		nil,
		mustFromODBC(TIMESTAMP_POSIX, list.NewListItem("")),
	)
}

