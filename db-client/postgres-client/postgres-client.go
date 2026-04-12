package postgresClient

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func InitPostgres(driverName string, connStr string) (*sqlx.DB, error) {

	// 连接到数据库
	db, err := sqlx.Connect(driverName, connStr)
	return db, err
}
