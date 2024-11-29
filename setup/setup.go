package setup

import (
	"fmt"
	"net"
	"os"

	"github.com/fatih/color"
	"github.com/yaoapp/yao/config"
)

// URLs get admin url
func URLs(cfg config.Config) ([]string, error) {

	ips, err := Ips()
	if err != nil {
		return nil, err
	}

	for i := range ips {
		ips[i] = fmt.Sprintf("http://%s:%d", ips[i], cfg.Port)
	}

	return ips, nil
}

func printError(message string, args ...interface{}) {
	fmt.Println(color.RedString(message, args...))
	os.Exit(1)
}

func printInfo(message string, args ...interface{}) {
	fmt.Println(color.GreenString(message, args...))
}

// Ips get the local ip list
func Ips() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	iplist := []string{"127.0.0.1"}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				iplist = append(iplist, ipnet.IP.String())
			}
		}
	}
	return iplist, nil
}
