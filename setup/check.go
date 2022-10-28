package setup

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
)

// Check setup programe
func Check() bool {

	root := appRoot()

	appfile := filepath.Join(root, "app.json")
	if _, err := os.Stat(appfile); err != nil && os.IsNotExist(err) {
		return true
	}

	envfile := filepath.Join(root, ".env")
	if _, err := os.Stat(envfile); err != nil && os.IsNotExist(err) {
		cfg, err := getConfig()
		if err != nil || !hasInstalled(cfg) {
			return true
		}
	}

	return false
}

// ValidateHosting host ports
func ValidateHosting(option map[string]string) error {
	if option["YAO_PORT"] == "" {
		return fmt.Errorf("监听端口必须填写")
	}

	if option["YAO_STUDIO_PORT"] == option["YAO_PORT"] {
		return fmt.Errorf("监听端口和 Studio 端口不能相同")
	}

	if option["YAO_PORT"] != SetupPort {
		conn, _ := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", option["YAO_PORT"]), time.Second)
		if conn != nil {
			defer conn.Close()
			return fmt.Errorf("监听端口 %s 已被占用", option["YAO_PORT"])
		}
	}

	if option["YAO_STUDIO_PORT"] != SetupPort {
		conn, _ := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", option["YAO_STUDIO_PORT"]), time.Second)
		if conn != nil {
			defer conn.Close()
			return fmt.Errorf("Studio 端口 %s 已被占用", option["YAO_STUDIO_PORT"])
		}
	}

	return nil
}

// ValidateDB db connection
func ValidateDB(option map[string]string) error {

	driver, dsn, err := getDSN(option)
	if err != nil {
		return fmt.Errorf("连接失败 %s", err.Error())
	}

	m, err := capsule.Add("validate", driver, dsn)
	if err != nil {
		return fmt.Errorf("连接失败 %s", err.Error())
	}

	conn, err := m.Primary()
	if err != nil {
		return fmt.Errorf("连接失败 %s", err.Error())
	}

	err = conn.Ping(2 * time.Second)
	if err != nil {
		return fmt.Errorf("连接失败 %s", err.Error())
	}

	return nil
}

func appRoot() string {

	root := os.Getenv("YAO_ROOT")
	if root == "" {
		path, err := os.Getwd()
		if err != nil {
			printError("无法获取应用目录: %s", err)
			Stop()
		}
		root = path
	}

	root, err := filepath.Abs(root)
	if err != nil {
		printError("无法获取应用目录: %s", err)
		Stop()
	}

	return root
}

func getConfig() (config.Config, error) {
	root := appRoot()
	envfile := filepath.Join(root, ".env")
	cfg := config.LoadFrom(envfile)
	return cfg, nil
}

func hasInstalled(cfg config.Config) bool {

	switch cfg.DB.Driver {

	case "sqlite3":
		if cfg.DB.Primary != nil && len(cfg.DB.Primary) > 0 {

			dbfile, err := filepath.Abs(cfg.DB.Primary[0])
			if err != nil {
				return false
			}

			if _, err := os.Stat(dbfile); err != nil && os.IsNotExist(err) {
				return false
			}

			return true
		}
		break

	case "mysql":
		if cfg.DB.Primary != nil && len(cfg.DB.Primary) > 0 {
			return true
		}
		break
	}

	return false
}
