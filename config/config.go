package config

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v6"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/crypto"
)

// Conf 配置参数
var Conf Config

// LogOutput 日志输出
var LogOutput *os.File // 日志文件

// DSLExtensions the dsl file Extensions
var DSLExtensions = []string{"*.yao", "*.json", "*.jsonc"}

func init() {
	filename, _ := filepath.Abs(filepath.Join(".", ".env"))
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		Conf = Load()
		return
	}

	Conf = LoadFrom(filename)
	if Conf.Mode == "production" {
		Production()
	} else if Conf.Mode == "development" {
		Development()
	}
}

// LoadFrom 从配置项中加载
func LoadFrom(envfile string) Config {

	file, err := filepath.Abs(envfile)
	if err != nil {
		cfg := Load()
		ReloadLog()
		return cfg
	}

	// load from env
	godotenv.Overload(file)
	cfg := Load()
	ReloadLog()
	return cfg
}

// Load the config
func Load() Config {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		exception.New("Can't read config %s", 500, err.Error()).Throw()
	}

	// Root path
	cfg.Root, _ = filepath.Abs(cfg.Root)

	// App Root
	if cfg.AppSource == "" {
		cfg.AppSource = cfg.Root
	}

	// Studio Secret
	if cfg.Studio.Secret == nil {
		v, err := crypto.Hash(crypto.HashTypes["SHA256"], uuid.New().String())
		if err != nil {
			exception.New("Can't gengrate studio secret %s", 500, err.Error()).Throw()
		}
		cfg.Studio.Secret = []byte(strings.ToUpper(v))
		cfg.Studio.Auto = true
	}

	// DataRoot
	if cfg.DataRoot == "" {
		cfg.DataRoot = filepath.Join(cfg.Root, "data")
		if !filepath.IsAbs(cfg.DataRoot) {
			cfg.DataRoot, _ = filepath.Abs(cfg.DataRoot)
		}

		if _, err := os.Stat(cfg.DataRoot); errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(cfg.DataRoot, os.ModePerm); err != nil {
				exception.New("Can't create data root %s", 500, err.Error()).Throw()
			}
		}
	}

	return cfg
}

// Production 设定为生产环境
func Production() {
	Conf.Mode = "production"
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(log.TEXT)
	if Conf.LogMode == "JSON" {
		log.SetFormatter(log.JSON)
	}
	gin.SetMode(gin.ReleaseMode)
	ReloadLog()
}

// Development 设定为开发环境
func Development() {
	Conf.Mode = "development"
	log.SetLevel(log.TraceLevel)
	log.SetFormatter(log.TEXT)
	if Conf.LogMode == "JSON" {
		log.SetFormatter(log.JSON)
	}
	gin.SetMode(gin.DebugMode)
	ReloadLog()
}

// ReloadLog 重新打开日志
func ReloadLog() {
	CloseLog()
	OpenLog()
}

// OpenLog 打开日志
func OpenLog() {

	if Conf.Log == "" {
		Conf.Log = filepath.Join(Conf.Root, "logs", "application.log")
	}

	if !filepath.IsAbs(Conf.Log) {
		Conf.Log = filepath.Join(Conf.Root, Conf.Log)
	}

	logfile, err := filepath.Abs(Conf.Log)
	if err != nil {
		log.With(log.F{"file": logfile}).Error(err.Error())
		return
	}

	logpath := filepath.Dir(logfile)
	if _, err := os.Stat(logpath); os.IsNotExist(err) {
		if err := os.MkdirAll(logpath, os.ModePerm); err != nil {
			log.With(log.F{"file": logfile}).Error(err.Error())
			return
		}
	}
	LogOutput, err = os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.With(log.F{"file": logfile}).Error(err.Error())
		return
	}

	log.SetOutput(LogOutput)
	gin.DefaultWriter = io.MultiWriter(LogOutput)
}

// CloseLog 关闭日志
func CloseLog() {
	if LogOutput != nil {
		err := LogOutput.Close()
		if err != nil {
			log.Error(err.Error())
			return
		}
	}
}
