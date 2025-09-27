package tests_test

import (
	"database/sql"
	"testing"

	_ "github.com/caretdev/go-irisnative"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// var connectionString string

// func TestMain(m *testing.M) {
// 	// ctx := context.Background()
// 	// ctr, err := iris.RunContainer(ctx)
// 	// if err != nil {
// 	// 	log.Println("Failed to start container:", err)
// 	// 	os.Exit(1)
// 	// }
// 	// connectionString = ctr.MustConnectionString(ctx)
// 	// log.Println("Container started successfully", connectionString)

// 	// exitCode := m.Run()
// 	// ctr.Terminate(ctx)
// 	// os.Exit(exitCode)
// }

// func openDbWrapper[T require.TestingT](t T, dsn string) *sql.DB {
// 	db, err := sql.Open(`intersystems`, dsn)
// 	require.NoError(t, err)
// 	require.NoError(t, db.Ping())
// 	return db
// }

// func closeDbWrapper[T require.TestingT](t T, db *sql.DB) {
// 	if db == nil {
// 		return
// 	}
// 	require.NoError(t, db.Close())
// }

func TestOpen(t *testing.T) {
	t.Run("with config", func(t *testing.T) {
		db := openDbWrapper(t, connectionString)
		defer closeDbWrapper(t, db)

		var (
			namespace string
			username  string
		)
		res := db.QueryRow(`SELECT $namespace, $username`)
		require.NoError(t, res.Scan(&namespace, &username))
		require.Equal(t, "TEST", namespace)
		require.Equal(t, "testuser", username)
	})
}

func TestInsert(t *testing.T) {
	t.Run("with config", func(t *testing.T) {
		var err error
		db := openDbWrapper(t, connectionString)
		defer closeDbWrapper(t, db)

		_, err = db.Exec("create table Sample.Person (ID identity, Name varchar(100))")
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO Sample.Person (Name) VALUES ('Test')")
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO Sample.Person (Name) VALUES (?)", "Test")
		require.NoError(t, err)
		var rows *sql.Rows
		rows, err = db.Query("select * from Sample.Person")
		require.NoError(t, err)
		defer rows.Close()

		var id int
		var name string
		assert.Equal(t, true, rows.Next())
		err = rows.Scan(&id, &name)
		require.NoError(t, err)
		assert.Equal(t, []interface{}{1, "Test"}, []interface{}{id, name})

		assert.Equal(t, true, rows.Next())
		err = rows.Scan(&id, &name)
		require.NoError(t, err)
		assert.Equal(t, []interface{}{2, "Test"}, []interface{}{id, name})

		_, err = db.Exec("drop table Sample.Person")
		require.NoError(t, err)
	})
}
