package setup

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
)

// Check if the app is installed
// true: start setup, false: start app
func Check() bool {

	// check app source
	if !appSourceExists() {
		return true
	}

	// check env file
	root := appRoot()
	envfile := filepath.Join(root, ".env")
	if _, err := os.Stat(envfile); err != nil && os.IsNotExist(err) {
		return true
	}

	return false
}

func appSourceExists() bool {

	appsource := appSource()
	if strings.HasSuffix(appsource, ".yaz") || strings.HasPrefix(appsource, "::binary") {
		return true
	}

	// check app.yao/app.json/app.jsonc
	root := appRoot()
	appfiles := []string{"app.yao", "app.json", "app.jsonc"}
	exist := false
	for _, appfile := range appfiles {
		appfile = filepath.Join(root, appfile)
		if _, err := os.Stat(appfile); err == nil {
			exist = true
			break
		}
	}

	return exist
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

func appSource() string {
	return os.Getenv("YAO_APP_SOURCE")
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

	return false

	// switch cfg.DB.Driver {

	// case "sqlite3":
	// 	if cfg.DB.Primary != nil && len(cfg.DB.Primary) > 0 {

	// 		dbfile, err := filepath.Abs(cfg.DB.Primary[0])
	// 		if err != nil {
	// 			return false
	// 		}

	// 		if _, err := os.Stat(dbfile); err != nil && os.IsNotExist(err) {
	// 			return false
	// 		}

	// 		return false
	// 	}
	// 	break

	// case "mysql":
	// 	if cfg.DB.Primary != nil && len(cfg.DB.Primary) > 0 {
	// 		return true
	// 	}
	// 	break
	// }

	// return false
}
