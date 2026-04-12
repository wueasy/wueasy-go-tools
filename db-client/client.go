package dbClient

import (
	"database/sql"
	"errors"
	"time"

	postgresClient "github.com/wueasy/wueasy-go-tools/db-client/postgres-client"

	mysqlClient "github.com/wueasy/wueasy-go-tools/db-client/mysql-client"

	nacosConfig "github.com/wueasy/wueasy-go-tools/config"

	"github.com/jmoiron/sqlx"
)

func Init(config nacosConfig.DbConfig) (*sqlx.DB, error) {

	driverName := "mysql"
	if config.DriverName != "" {
		driverName = config.DriverName
	}

	// 如果开启 SQL 日志，注册包装 driver 并使用它
	actualDriverName := driverName
	if config.ShowSql {
		actualDriverName = logDriverName(driverName)
		if !registeredLogDrivers[actualDriverName] {
			// 获取原始 driver 实例来包装
			db, err := sql.Open(driverName, "")
			if err == nil {
				origDriver := db.Driver()
				db.Close()
				sql.Register(actualDriverName, &logDriver{Driver: origDriver})
				registeredLogDrivers[actualDriverName] = true
			}
		}
	}

	var connStr string

	// 账号和密码单独设置，拼接到连接字符串
	connStr = config.Uri
	if config.Username != "" && config.Password != "" {
		if driverName == "mysql" {
			connStr = config.Username + ":" + config.Password + "@" + config.Uri
		} else if driverName == "postgres" {
			connStr = "postgres://" + config.Username + ":" + config.Password + "@" + config.Uri
		}
	}

	var db *sqlx.DB
	var err error

	if config.ShowSql {
		// 直接用 sqlx.NewDb + sql.Open 使用包装后的 driver
		sqlDb, openErr := sql.Open(actualDriverName, connStr)
		if openErr != nil {
			return nil, openErr
		}
		db = sqlx.NewDb(sqlDb, driverName) // driverName 用于方言判断
	} else {
		if driverName == "mysql" {
			db, err = mysqlClient.InitMysql(driverName, connStr)
		} else if driverName == "postgres" {
			db, err = postgresClient.InitPostgres(driverName, connStr)
		} else {
			return nil, errors.New("不支持的数据库类型")
		}
		if err != nil {
			return nil, err
		}
	}

	// 设置连接池参数
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)

	// 验证连接
	err = db.Ping()
	return db, err
}
