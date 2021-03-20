package connection

import (
	"errors"
	"net"
)

const VERSION_PROTOCOL uint16 = 57

type Connection struct {
	conn         *net.TCPConn
	messageCount uint32
	unicode      bool
	locale       string
	version      uint16
	info         string
}

func Connect(addr string, namespace, login, password string) (connection Connection, err error) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return
	}

	connection = Connection{
		conn: conn,
	}

	if err = connection.handshake(); err != nil {
		return
	}

	if err = connection.connect(namespace, login, password); err != nil {
		return
	}

	return
}

func (c *Connection) Disconnect() {
	msg := NewMessage(DISCONNECT)
	msg.Set("")
	c.conn.Write(msg.Dump(c.count()))
}

func (c *Connection) count() uint32 {
	count := c.messageCount
	c.messageCount += 1
	return count
}

func (c *Connection) handshake() (err error) {
	var message = NewMessage(HANDSHAKE)
	message.AddRaw(VERSION_PROTOCOL)

	_, err = c.conn.Write(message.Dump(c.count()))
	if err != nil {
		return
	}

	msg, err := ReadMessage(c.conn)
	if err != nil {
		return
	}

	var version uint16
	msg.GetRaw(&version)
	c.version = version

	var unicode uint16
	msg.GetRaw(&unicode)
	c.unicode = unicode == 1

	var locale string
	msg.Get(&locale)
	c.locale = locale
	return
}

func encode(value string) []byte {
	in := []byte(value)
	length := len(in)
	out := make([]byte, length)
	for i := range in {
		length--
		temp := ((int(in[i])^0xa7)&0xff + length) & 0xff
		out[length] = byte(temp<<5 | temp>>3)
	}
	return out
}

func (c *Connection) connect(namespace, login, password string) (err error) {
	msg := NewMessage(CONNECT)
	msg.Set(namespace)
	msg.Set(encode(login))
	msg.Set(encode(password))
	msg.Set("go")            // machine user name
	msg.Set("go-machine")    // machine name
	msg.Set("libirisnative") // application name
	msg.Set("")
	msg.Set("|||||")
	msg.Set("")
	msg.Set(1)
	msg.Set(0)
	msg.Set(1)

	_, err = c.conn.Write(msg.Dump(c.count()))
	if err != nil {
		return
	}

	msg, err = ReadMessage(c.conn)
	if err != nil {
		return
	}
	if status := msg.GetStatus(); status == 417 {
		var errorMsg string
		msg.Get(&errorMsg)
		err = errors.New(errorMsg)
		return
	}

	var info string
	msg.Get(&info)
	c.info = info
	return
}
