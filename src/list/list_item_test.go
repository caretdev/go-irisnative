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
		var listItem, offset = GetListItem(v.dump, 0)
		if tempDump := listItem.Dump(); !reflect.DeepEqual(v.dump, tempDump) {
			t.Error("Not equal", v.dump, tempDump, listItem)
		}
		if offset != uint(len(v.dump)) {
			t.Error("Wrong offset: ", v)
		}
	}
}
