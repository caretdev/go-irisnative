package connection

import (
	"fmt"
	"net"

	"github.com/caretdev/go-irisnative/src/list"
)

type Message struct {
	header MessageHeader
	data   []byte
	offset uint
}

func NewMessage(messageType MessageType) Message {
	return Message{
		NewMessageHeader(messageType),
		[]byte{},
		0,
	}
}

func ReadMessage(conn *net.TCPConn) (msg Message, err error) {
	buffer := make([]byte, 14)

	_, err = conn.Read(buffer)
	if err != nil {
		return
	}

	var header [14]byte
	copy(header[:], buffer[:14])
	var msgHeader = MessageHeader{header}

	data := make([]byte, msgHeader.GetLength())
	_, err = conn.Read(data)
	if err != nil {
		return
	}

	msg = Message{msgHeader, data, 0}

	return
}

func (m *Message) AddRaw(value interface{}) {
	switch v := value.(type) {
	case uint16:
		m.data = append(m.data, byte(v&0xff))
		m.data = append(m.data, byte(v>>8&0xff))
		m.offset += 2
	case []byte:
		m.data = append(m.data, v...)
		m.offset += uint(len(v))
	}
}

func (m *Message) GetRaw(value interface{}) error {
	switch v := value.(type) {
	case *uint16:
		*v = uint16(m.data[m.offset]) | (uint16(m.data[m.offset+1]) << 8)
		m.offset += 2
	case *bool:
		*v = (uint16(m.data[m.offset]) | (uint16(m.data[m.offset+1]) << 8)) == 1
		m.offset += 2
	default:
		return fmt.Errorf("unknown type: %T", v)
	}
	return nil
}

func (m *Message) Set(value interface{}) error {
	listItem := list.NewListItem(value)
	m.AddRaw(listItem.Dump())
	return nil
}

func (m *Message) GetStatus() uint16 {
	return m.header.GetStatus()
}

func (m *Message) Get(value interface{}) error {
	listItem, offset := list.GetListItem(m.data, m.offset)
	m.offset = offset
	listItem.Get(value)
	return nil
}

type AnyType struct {
	listItem list.ListItem
}

func (v *AnyType) Int() int {
	var value int
	v.listItem.Get(&value)
	return value
}

func (m *Message) GetAny() AnyType {
	listItem, offset := list.GetListItem(m.data, m.offset)
	m.offset = offset
	return AnyType{listItem}
}

func (m *Message) Dump(count uint32) []byte {
	m.header.SetCount(count)
	m.header.SetLength(uint32(len(m.data)))

	return append(m.header.header[:], m.data...)
}
