package cmd

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: L("Logout from remote Yao server"),
	Long:  L("Revoke token and remove stored credentials"),
	Run: func(cmd *cobra.Command, args []string) {
		cred, err := LoadCredential()
		if err != nil {
			color.Red("  %s %s\n", L("Failed to read credentials:"), err)
			os.Exit(1)
		}
		if cred == nil {
			color.Yellow("  %s\n", L("Not logged in"))
			return
		}

		// Best-effort token revocation via discovery
		if cred.AccessToken != "" && cred.Server != "" {
			if ep, err := discoverEndpoints(cred.Server); err == nil && ep.RevocationEndpoint != "" {
				revokeToken(ep.RevocationEndpoint, cred.AccessToken)
			}
		}

		if err := RemoveCredential(); err != nil {
			color.Red("  %s %s\n", L("Failed to remove credentials:"), err)
			os.Exit(1)
		}

		color.Green("  ✓ %s\n", L("Logged out"))
		if cred.Server != "" {
			color.White("    %s %s\n", L("Server:"), cred.Server)
		}
	},
}

func revokeToken(endpoint, token string) {
	data := url.Values{"token": {token}}
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	http.DefaultClient.Do(req)
}

func init() {
	// Add i18n entries
	langs["Login to remote Yao server"] = "登录远程 Yao 服务器"
	langs["Login to remote Yao server using device authorization flow"] = "使用设备授权流程登录远程 Yao 服务器"
	langs["Remote Yao server URL"] = "远程 Yao 服务器地址"
	langs["Logout from remote Yao server"] = "登出远程 Yao 服务器"
	langs["Revoke token and remove stored credentials"] = "撤销令牌并移除存储的凭证"
	langs["Missing --server flag"] = "缺少 --server 参数"
	langs["Open:"] = "打开:"
	langs["Or visit:"] = "或访问:"
	langs["Enter code:"] = "输入设备码:"
	langs["Waiting for authorization..."] = "等待授权..."
	langs["Login failed:"] = "登录失败:"
	langs["Login successful"] = "登录成功"
	langs["Server:"] = "服务器:"
	langs["Scope:"] = "授权范围:"
	langs["Failed to read credentials:"] = "读取凭证失败:"
	langs["Not logged in"] = "未登录"
	langs["Failed to remove credentials:"] = "移除凭证失败:"
	langs["Logged out"] = "已登出"
	langs["Path to credentials file"] = "凭证文件路径"
	langs["Failed to load credentials:"] = "加载凭证失败:"
	langs["Server discovery failed:"] = "服务发现失败:"
}
