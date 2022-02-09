package network

import (
	"net"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

// IP 读取IP地址
func IP() map[string]string {
	res := map[string]string{}
	ifaces, err := net.Interfaces()
	if err != nil {
		exception.New("读取网卡失败 %s", 500, err.Error()).Throw()
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			exception.New("读取IP地址失败 %s", 500, err.Error()).Throw()
		}
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			res[i.Name] = ip.String()
		}
	}
	return res
}

// ProcessIP  xiang.network.IP IP地址
func ProcessIP(process *gou.Process) interface{} {
	return IP()
}

// FreePort xiang.network.FreePort 获取可用端口
func FreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		exception.New("获取可用端口失败 %s", 500, err.Error()).Throw()
		return 0
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		exception.New("获取可用端口失败 %s", 500, err.Error()).Throw()
		return 0
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// ProcessFreePort  xiang.network.FreePort 获取可用端口
func ProcessFreePort(process *gou.Process) interface{} {
	return FreePort()
}
