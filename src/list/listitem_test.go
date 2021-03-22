package list

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestItem struct {
	value interface{}
	dump  []byte
}

var tests []TestItem = []TestItem{
	{
		nil,
		[]byte{0x01},
	},
	{
		0,
		[]byte{0x02, 0x04},
	},
	{
		144,
		[]byte{0x03, 0x04, 0x90},
	},
	{
		256,
		[]byte{0x04, 0x04, 0x00, 0x01},
	},
	{
		-1,
		[]byte{0x02, 0x05},
	},
	{
		-256,
		[]byte{0x03, 0x05, 0x00},
	},
	{
		-257,
		[]byte{0x04, 0x05, 0xff, 0xfe},
	},
	{
		"abc",
		[]byte{0x05, 0x01, 0x61, 0x62, 0x63},
	},
	{
		"",
		[]byte{0x02, 0x01},
	},
	{
		[]byte{0x61, 0x62, 0x63},
		[]byte{0x05, 0x01, 0x61, 0x62, 0x63},
	},
}

func TestListItem(t *testing.T) {
	for _, v := range tests {
		var offset uint = 0
		var listItem = GetListItem(v.dump, &offset)
		if tempDump := listItem.Dump(); !reflect.DeepEqual(v.dump, tempDump) {
			t.Error("Not equal", v.dump, tempDump, listItem)
		}
		if offset != uint(len(v.dump)) {
			t.Error("Wrong offset: ", v)
		}
	}
}

var longString = map[int][]byte{
	253: {0xff, 0x01},
	254: {0x00, 0xff, 0x00, 0x01},
	255: {0x00, 0x00, 0x01, 0x01},
	256: {0x00, 0x01, 0x01, 0x01},
	512: {0x00, 0x01, 0x02, 0x01},
}

func TestLongListItem(t *testing.T) {
	for l, v := range longString {
		temp := make([]byte, l)
		for i := range temp {
			temp[i] = 255
		}
		li := NewListItem(temp)
		tempDump := li.Dump()[:len(v)]
		if !reflect.DeepEqual(v, tempDump) {
			t.Error("Not equal", l, tempDump, v)
		}
	}
}

func TestIntListItem(t *testing.T) {
	var li ListItem
	li = NewListItem(int(1))
	assert.Equal(t, li.Dump(), []byte{0x3, 0x4, 0x1})
	li = NewListItem(int8(1))
	assert.Equal(t, li.Dump(), []byte{0x3, 0x4, 0x1})
	li = NewListItem(int16(1))
	assert.Equal(t, li.Dump(), []byte{0x3, 0x4, 0x1})
	li = NewListItem(int32(1))
	assert.Equal(t, li.Dump(), []byte{0x3, 0x4, 0x1})
	li = NewListItem(int64(1))
	assert.Equal(t, li.Dump(), []byte{0x3, 0x4, 0x1})

	li = NewListItem(uint(1))
	assert.Equal(t, li.Dump(), []byte{0x3, 0x4, 0x1})
	li = NewListItem(uint8(1))
	assert.Equal(t, li.Dump(), []byte{0x3, 0x4, 0x1})
	li = NewListItem(uint16(1))
	assert.Equal(t, li.Dump(), []byte{0x3, 0x4, 0x1})
	li = NewListItem(uint32(1))
	assert.Equal(t, li.Dump(), []byte{0x3, 0x4, 0x1})
	li = NewListItem(uint64(1))
	assert.Equal(t, li.Dump(), []byte{0x3, 0x4, 0x1})
}

func TestFloatListItem(t *testing.T) {
	var li ListItem
	var v float64

	li = NewListItem(float32(0.01))
	assert.Equal(t, []byte{0x4, 0x6, 0xfe, 0x1}, li.Dump())
	v, _ = li.asFloat64()
	assert.Equal(t, float64(0.01), v)

	li = NewListItem(float64(0.01))
	assert.Equal(t, []byte{0x4, 0x6, 0xfe, 0x1}, li.Dump())
	v, _ = li.asFloat64()
	assert.Equal(t, float64(0.01), v)

	li = NewListItem(float64(1.234))
	assert.Equal(t, []byte{0x5, 0x6, 0xfd, 0xd2, 0x4}, li.Dump())
	v, _ = li.asFloat64()
	assert.Equal(t, float64(1.234), v)

	li = NewListItem(float64(-12.345))
	assert.Equal(t, []byte{0x5, 0x7, 0xfd, 0xc7, 0xcf}, li.Dump())
	v, _ = li.asFloat64()
	assert.Equal(t, float64(-12.345), v)

	li = NewListItem(float64(100))
	assert.Equal(t, []byte{0x4, 0x6, 0x2, 0x1}, li.Dump())
	v, _ = li.asFloat64()
	assert.Equal(t, float64(100), v)

	li = NewListItem(float64(-100))
	assert.Equal(t, []byte{0x3, 0x7, 0x2}, li.Dump())
	v, _ = li.asFloat64()
	assert.Equal(t, float64(-100), v)
}

func TestBoolListItem(t *testing.T) {
	var li ListItem
	li = NewListItem(false)
	assert.Equal(t, []byte{0x3, 0x4, 0x0}, li.Dump())
	li = NewListItem(true)
	assert.Equal(t, []byte{0x3, 0x4, 0x1}, li.Dump())
}

func TestEmptyListItem(t *testing.T) {
	var li ListItem
	li = NewListItem(nil)
	assert.Equal(t, []byte{0x1}, li.Dump())
	// assert.Equal(t, []byte{0x2, 0x1}, li.Dump())
	li = NewListItem("")
	assert.Equal(t, []byte{0x2, 0x1}, li.Dump())
}

func TestUnicode(t *testing.T) {
	var li ListItem
	var v string
	var val string = "тест"
	li = NewListItem(val)
	assert.Equal(t, []byte{0x0a, 0x02, 0x42, 0x04, 0x35, 0x04, 0x41, 0x04, 0x42, 0x04}, li.Dump())
	v, _ = li.asString()
	assert.Equal(t, v, val)

	val = "test"
	li = NewListItem(val)
	assert.Equal(t, []byte{0x6, 0x1, 0x74, 0x65, 0x73, 0x74}, li.Dump())
	v, _ = li.asString()
	assert.Equal(t, v, val)

	val = "testтест"
	li = NewListItem(val)
	assert.Equal(t, []byte{0x12, 0x02, 0x74, 0x00, 0x65, 0x00, 0x73, 0x00, 0x74, 0x00, 0x42, 0x04, 0x35, 0x04, 0x41, 0x04, 0x42, 0x04}, li.Dump())
	v, _ = li.asString()
	assert.Equal(t, v, val)

	// val = "💻"
	// li = NewListItem(val)
	// assert.Equal(t, []byte{0x06, 0x02, 0x3d, 0xd8, 0xbb, 0xdc}, li.Dump())
	// v, _ = li.asString()
	// assert.Equal(t, v, val)
}
