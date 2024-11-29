package setup

import (
	"fmt"
	"net"
	"os"

	"github.com/fatih/color"
	"github.com/yaoapp/yao/config"
)

// Endpoints get endpoints
func Endpoints(cfg config.Config) ([]Endpoint, error) {
	networks, err := getNetworks()
	if err != nil {
		return nil, err
	}

	var endpoints []Endpoint
	for _, network := range networks {
		port := fmt.Sprintf(":%d", cfg.Port)
		if port == ":80" {
			port = ""
		}
		endpoint := Endpoint{
			URL:       fmt.Sprintf("http://%s%s", network.IPv4, port),
			Interface: network.Interface,
		}
		endpoints = append(endpoints, endpoint)
	}

	return endpoints, nil
}

func printError(message string, args ...interface{}) {
	fmt.Println(color.RedString(message, args...))
	os.Exit(1)
}

func printInfo(message string, args ...interface{}) {
	fmt.Println(color.GreenString(message, args...))
}

func getNetworks() ([]Network, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var networks []Network
	for _, iface := range interfaces {
		// 跳过 loopback 接口（如 lo0）
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 获取每个接口的地址信息
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		// 过滤只获取 IPv4 地址
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				// 将网卡名称和 IPv4 地址存储到 Network 结构体中
				network := Network{
					IPv4:      ipnet.IP.String(),
					Interface: iface.Name,
				}
				// 添加到结果切片
				networks = append(networks, network)
			}
		}
	}
	return networks, nil
}

// Network network
type Network struct {
	IPv4      string
	Interface string
}

// Endpoint endpoint
type Endpoint struct {
	URL       string
	Interface string
}
