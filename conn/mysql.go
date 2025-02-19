package conn

import (
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var mysqlClient *sqlx.DB

func ShengcaiMysql(user, pass, host, dbName string) {
	if ENV == "local-docker" {
		host = HostDockerInternal
	}
	dsn := (&mysql.Config{
		User:                 user,
		Passwd:               pass,
		Net:                  "tcp",
		Addr:                 host + ":3306",
		DBName:               dbName,
		Collation:            "utf8mb4_unicode_ci",
		AllowNativePasswords: true,
	}).FormatDSN()

	mysqlClient = sqlx.MustOpen("mysql", dsn)
}

func NewMysql(user, pass, host, dbName string) {
	if ENV == "local" {
		host = "127.0.0.1"
	} else if ENV == "local-docker" {
		host = HostDockerInternal
	}
	dsn := (&mysql.Config{
		User:                 user,
		Passwd:               pass,
		Net:                  "tcp",
		Addr:                 host + ":3306",
		DBName:               dbName,
		Collation:            "utf8mb4_unicode_ci",
		AllowNativePasswords: true,
	}).FormatDSN()

	mysqlClient = sqlx.MustOpen("mysql", dsn)
}

func GetDB() *sqlx.DB {
	return mysqlClient
}
