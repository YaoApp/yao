package share

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"

	"github.com/yaoapp/gou/session"
)

var sessionDB *session.BuntDB

// SessionStart start session
func SessionStart() error {
	if config.Conf.Session.Store == "file" {
		return SessionFile()
	} else if config.Conf.Session.Store == "redis" {
		return SessionRedis()
	}
	return fmt.Errorf("Session Store config error %s (file|redis)", config.Conf.Session.Store)
}

// SessionStop stop session
func SessionStop() {
	if sessionDB != nil {
		sessionDB.Close()
	}
}

// SessionRedis Connect redis server
func SessionRedis() error {
	args := []string{}
	if config.Conf.Session.Port == "" {
		config.Conf.Session.Port = "6379"
	}

	if config.Conf.Session.DB == "" {
		config.Conf.Session.DB = "1"
	}

	args = append(args, config.Conf.Session.Port, config.Conf.Session.DB, config.Conf.Session.Password)
	rdb, err := session.NewRedis(config.Conf.Session.Host, args...)
	if err != nil {
		return err
	}

	session.Register("redis", rdb)
	session.Name = "redis"
	log.Trace("Session Store:REDIS HOST:%s PORT:%s DB:%s", config.Conf.Session.Host, config.Conf.Session.Port, config.Conf.Session.DB)
	return nil
}

// SessionFile Start session file
func SessionFile() error {
	file := config.Conf.Session.File
	if file == "" {
		file = filepath.Join(config.Conf.Root, "data", ".session.db")
	}

	file, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	burndb, err := session.NewBuntDB(file)
	if err != nil {
		return fmt.Errorf("Session Store File %s Error: %s. Try to remove the file then restart", file, err.Error())
	}

	session.Register("file", burndb)
	session.Name = "file"
	sessionDB = burndb
	log.Trace("Session Store: File %s", file)
	return nil
}
