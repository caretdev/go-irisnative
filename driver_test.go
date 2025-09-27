package intersystems

// func TestInsert(t *testing.T) {
// 	t.Run("with config", func(t *testing.T) {
// 		var err error
// 		db := openDbWrapper(t, connectionString)
// 		defer closeDbWrapper(t, db)

// 		var res sql.Result
// 		_, _ = db.Exec("drop table if exists Sample.Person")
// 		res, err = db.Exec("create table Sample.Person (ID int primary key, Name varchar(100))")
// 		fmt.Println(res, err)
// 		res, err = db.Exec("INSERT INTO Sample.Person (ID, Name) VALUES (1, 'Test')")
// 		fmt.Println(res, err)
// 		// require.NoError(t, res.Scan(&namespace, &username))
// 		// require.Equal(t, "USER", namespace)
// 		// require.Equal(t, "_SYSTEM", username)
// 	})
// }
