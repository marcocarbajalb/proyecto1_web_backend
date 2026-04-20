package db

import (
	"database/sql"
	_ "embed"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Abre la conexión a SQLite y verifica que responde
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// Ejecuta el schema embebido
func Migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
