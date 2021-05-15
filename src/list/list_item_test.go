package list

import (
	"reflect"
	"testing"
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
