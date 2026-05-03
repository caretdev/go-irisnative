package list

import (
	"math"
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

	// Test float32 encoding (compact float, type 08)
	li = NewListItem(float32(0.01))
	dump := li.Dump()
	assert.Equal(t, byte(0x08), dump[1]) // Should be type 08 (compact float)
	v, _ = li.asFloat64()
	assert.InDelta(t, float64(0.01), v, 0.0001)

	// Test float64 encoding (IEEE double, type 09)
	li = NewListItem(float64(0.01))
	dump = li.Dump()
	assert.Equal(t, byte(0x09), dump[1]) // Should be type 09 (IEEE double)
	v, _ = li.asFloat64()
	assert.InDelta(t, float64(0.01), v, 0.0001)

	// Test round-trip for various float values
	testValues := []float64{1.234, -12.345, 100, -100, 0.5, -0.5, 1.5, -1.5}
	for _, testVal := range testValues {
		li = NewListItem(testVal)
		dump = li.Dump()
		assert.Equal(t, byte(0x09), dump[1]) // Should be type 09 (IEEE double)
		v, _ = li.asFloat64()
		assert.InDelta(t, testVal, v, 0.0001)
	}
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
	assert.Equal(t, byte(0x02), li.Dump()[1]) // Should be type 02 (Unicode)
	v, _ = li.asString()
	assert.Equal(t, v, val)

	val = "test"
	li = NewListItem(val)
	assert.Equal(t, byte(0x01), li.Dump()[1]) // Should be type 01 (ASCII string)
	v, _ = li.asString()
	assert.Equal(t, v, val)

	val = "testтест"
	li = NewListItem(val)
	assert.Equal(t, byte(0x02), li.Dump()[1]) // Should be type 02 (Unicode)
	v, _ = li.asString()
	assert.Equal(t, v, val)

	// Test emoji (surrogate pairs)
	val = "💻"
	li = NewListItem(val)
	assert.Equal(t, byte(0x02), li.Dump()[1]) // Should be type 02 (Unicode)
	v, _ = li.asString()
	assert.Equal(t, v, val)
}

func TestSpecialIEEEValues(t *testing.T) {
	var li ListItem
	var v float64

	// Test positive infinity
	li = NewListItem(math.Inf(1))
	dump := li.Dump()
	assert.Equal(t, byte(0x08), dump[1]) // Should be type 08 (compact float)
	v, _ = li.asFloat64()
	assert.True(t, math.IsInf(v, 1))

	// Test negative infinity
	li = NewListItem(math.Inf(-1))
	dump = li.Dump()
	assert.Equal(t, byte(0x08), dump[1]) // Should be type 08 (compact float)
	v, _ = li.asFloat64()
	assert.True(t, math.IsInf(v, -1))

	// Test NaN
	li = NewListItem(math.NaN())
	dump = li.Dump()
	assert.Equal(t, byte(0x09), dump[1]) // Should be type 09 (IEEE double)
	v, _ = li.asFloat64()
	assert.True(t, math.IsNaN(v))
}

