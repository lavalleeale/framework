package framework

import (
	"database/sql"
	"reflect"
)

type Migration interface {
	Up(*sql.DB) error
	Down(*sql.DB) error
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
			err = migration.Up(f.Db)
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
			err = migration.Down(f.Db)
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
