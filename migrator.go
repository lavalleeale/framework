package framework

import "reflect"

type Migration interface {
	Up() string
	Down() string
}

func (f *Framework) Migrate(migrations ...Migration) {
	f.Db.Query(`CREATE TABLE IF NOT EXISTS migrations (
    migration_id VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);`)
	for _, migration := range migrations {
		name := reflect.TypeOf(migration).Name()
		rows, err := f.Db.Query("SELECT migration_id FROM migrations WHERE migration_id = $1", name)
		if err != nil {
			panic(err)
		}
		if !rows.Next() {
			_, err = f.Db.Query(migration.Up())
			if err != nil {
				panic(err)
			}
			_, err = f.Db.Query("INSERT INTO migrations (migration_id) VALUES ($1)", name)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (f *Framework) Rollback(migrations ...Migration) {
	for _, migration := range migrations {
		name := reflect.TypeOf(migration).Name()
		rows, err := f.Db.Query("SELECT migration_id FROM migrations WHERE migration_id = $1", name)
		if err != nil {
			panic(err)
		}
		if rows.Next() {
			_, err = f.Db.Query(migration.Down())
			if err != nil {
				panic(err)
			}
			_, err = f.Db.Query("DELETE FROM migrations WHERE migration_id = $1", name)
			if err != nil {
				panic(err)
			}
		}
	}
}
