package mysqlClient

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func InitMysql(driverName string, connStr string) (*sqlx.DB, error) {

	// 连接到数据库
	db, err := sqlx.Connect(driverName, connStr)
	return db, err
}
