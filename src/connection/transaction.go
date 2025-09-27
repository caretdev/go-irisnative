package connection

type tx struct {
	c *Connection
}

func (t *tx) Commit() error {
	if t.c == nil || !t.c.TX {
		panic("database/sql/driver: misuse of driver: extra Commit")
	}

	t.c.TX = false
	err := t.c.Commit()
	t.c = nil

	return err
}

func (t *tx) Rollback() error {
	if t.c == nil || !t.c.TX {
		panic("database/sql/driver: misuse of driver: extra Rollback")
	}

	t.c.TX = false
	err := t.c.Rollback()
	t.c = nil

	return err
}
