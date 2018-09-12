package model

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/jinzhu/gorm"

	// for mysql usage
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var (
	user    = ""
	pass    = ""
	host    = ""
	port    = ""
	dbName  = ""
	options = ""
)

var instance *Engine
var once sync.Once

// Engine - base engine connector to data store.
type Engine struct{ *gorm.DB }

// GetEngine - return engine instance.
func GetEngine() *Engine {
	once.Do(func() {
		user = os.Getenv("DB_USER")
		pass = os.Getenv("DB_PASS")
		host = os.Getenv("DB_HOST")
		port = os.Getenv("DB_PORT")

		dbName = "wl_football_bot"

		options = strings.Join(
			[]string{
				"charset=utf8",
				"parseTime=True",
				"loc=Local",
			},
			"&",
		)

		db, _ := gorm.Open(
			"mysql",
			fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", user, pass, net.JoinHostPort(host, port), dbName, options),
		)

		db.AutoMigrate(&Vote{})

		instance = &Engine{db}
	})

	return instance
}
