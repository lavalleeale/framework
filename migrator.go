package framework

import (
	"reflect"

	"github.com/jmoiron/sqlx"
)

type Migration interface {
	Up(*sqlx.Tx) error
	Down(*sqlx.Tx) error
}

func (f *Framework) Migrate(migrations ...Migration) {
	f.Db.Query(`CREATE TABLE IF NOT EXISTS migrations (
    migration_id VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);`)
	tx := f.Db.MustBegin()
	for _, migration := range migrations {
		name := reflect.TypeOf(migration).Name()
		rows, err := tx.Query("SELECT migration_id FROM migrations WHERE migration_id = $1", name)
		if err != nil {
			tx.Rollback()
			panic(err)
		}
		defer rows.Close()
		if !rows.Next() {
			err = migration.Up(tx)
			if err != nil {
				tx.Rollback()
				panic(err)
			}
			_, err = tx.Exec("INSERT INTO migrations (migration_id) VALUES ($1)", name)
			if err != nil {
				tx.Rollback()
				panic(err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

func (f *Framework) Rollback(migrations ...Migration) {
	tx := f.Db.MustBegin()
	for _, migration := range migrations {
		name := reflect.TypeOf(migration).Name()
		rows, err := tx.Query("SELECT migration_id FROM migrations WHERE migration_id = $1", name)
		if err != nil {
			tx.Rollback()
			panic(err)
		}
		defer rows.Close()
		if rows.Next() {
			err = migration.Down(tx)
			if err != nil {
				tx.Rollback()
				panic(err)
			}
			_, err = tx.Exec("DELETE FROM migrations WHERE migration_id = $1", name)
			if err != nil {
				tx.Rollback()
				panic(err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		panic(err)
	}
}
