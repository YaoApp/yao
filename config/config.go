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
	"gopkg.in/natefinch/lumberjack.v2"
)

// Conf 配置参数
var Conf Config

// LogOutput 日志输出
var LogOutput io.WriteCloser // 日志文件

// DSLExtensions the dsl file Extensions
var DSLExtensions = []string{"*.yao", "*.json", "*.jsonc"}

func init() {
	Init()
}

// Init setting
func Init() {

	filename, _ := filepath.Abs(filepath.Join(".", ".env"))
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		Conf = Load()
		if Conf.Mode == "production" {
			Production()
		} else if Conf.Mode == "development" {
			Development()
		}
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
	if cfg.Studio.Secret == "" {
		v, err := crypto.Hash(crypto.HashTypes["SHA256"], uuid.New().String())
		if err != nil {
			exception.New("Can't gengrate studio secret %s", 500, err.Error()).Throw()
		}
		cfg.Studio.Secret = strings.ToUpper(v)
		cfg.Studio.Auto = true
	}

	// DataRoot
	if cfg.DataRoot == "" {
		cfg.DataRoot = filepath.Join(cfg.Root, "data")
		if !filepath.IsAbs(cfg.DataRoot) {
			cfg.DataRoot, _ = filepath.Abs(cfg.DataRoot)
		}
	}

	return cfg
}

// Production 设定为生产环境
func Production() {
	os.Setenv("YAO_MODE", "production")
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
	os.Setenv("YAO_MODE", "development")
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
		return
	}

	logpath := filepath.Dir(logfile)

	// Check if the log path exists
	if _, err := os.Stat(logpath); errors.Is(err, os.ErrNotExist) {
		LogOutput, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
		log.SetOutput(LogOutput)
		gin.DefaultWriter = io.MultiWriter(LogOutput)
		return
	}

	LogOutput = &lumberjack.Logger{
		Filename:   logfile,
		MaxSize:    Conf.LogMaxSize, // megabytes
		MaxBackups: Conf.LogMaxBackups,
		MaxAge:     Conf.LogMaxAage, //days
		LocalTime:  Conf.LogLocalTime,
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
