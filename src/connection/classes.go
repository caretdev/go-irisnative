package connection

func (c *Connection) ServerVersion() (result string, err error) {
	err = c.ClasMethod("%SYSTEM.Version", "GetVersion", &result)
	return
}

func (c *Connection) ClasMethod(class, method string, result interface{}, args ...interface{}) (err error) {
	msg := NewMessage(CLASSMETHOD_VALUE)
	msg.Set(class)
	msg.Set(method)
	msg.Set(len(args))
	for _, arg := range args {
		msg.Set(arg)
	}

	_, err = c.conn.Write(msg.Dump(c.count()))
	if err != nil {
		return
	}
	msg, err = ReadMessage(c.conn)
	if err != nil {
		return
	}

	msg.Get(result)

	return
}

func (c *Connection) ClasMethodVoid(class, method string, args ...interface{}) (err error) {
	msg := NewMessage(CLASSMETHOD_VOID)
	msg.Set(class)
	msg.Set(method)
	msg.Set(len(args))
	for _, arg := range args {
		msg.Set(arg)
	}

	_, err = c.conn.Write(msg.Dump(c.count()))
	if err != nil {
		return
	}
	msg, err = ReadMessage(c.conn)
	if err != nil {
		return
	}
	return
}
