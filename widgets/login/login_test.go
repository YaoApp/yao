package login

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/lang"
)

func TestLoad(t *testing.T) {

	os.Unsetenv("YAO_LANG")
	lang.Load(config.Conf)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(Logins))

	assert.Equal(t, "admin", Logins["admin"].ID)
	assert.Equal(t, "Admin Login", Logins["admin"].Name)
	assert.Equal(t, "yao.login.Admin", Logins["admin"].Action.Process)
	assert.Equal(t, []string{":payload"}, Logins["admin"].Action.Args)
	assert.Equal(t, "yao.utils.Captcha", Logins["admin"].Layout.Captcha)
	assert.Equal(t, "/images/admin-cover.png", Logins["admin"].Layout.Cover)
	assert.Equal(t, "/x/Chart/dashboard", Logins["admin"].Layout.Entry)
	assert.Equal(t, "https://yaoapps.com", Logins["admin"].Layout.Site)
	assert.Equal(t, "Make Your Dream With Yao App Engine", Logins["admin"].Layout.Slogan)

	assert.Equal(t, "user", Logins["user"].ID)
	assert.Equal(t, "User Login", Logins["user"].Name)
	assert.Equal(t, "scripts.user.Login", Logins["user"].Action.Process)
	assert.Equal(t, []string{":payload"}, Logins["user"].Action.Args)
	assert.Equal(t, "yao.utils.Captcha", Logins["user"].Layout.Captcha)
	assert.Equal(t, "/images/user-cover.png", Logins["user"].Layout.Cover)
	assert.Equal(t, "/x/Table/pet", Logins["user"].Layout.Entry)
	assert.Equal(t, "https://yaoapps.com/docs", Logins["user"].Layout.Site)
	assert.Equal(t, "Make Your Dream With Yao App Engine", Logins["user"].Layout.Slogan)
}

func TestLoadHK(t *testing.T) {

	os.Setenv("YAO_LANG", "zh-hk")
	lang.Load(config.Conf)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(Logins))

	assert.Equal(t, "admin", Logins["admin"].ID)
	assert.Equal(t, "管理員登錄", Logins["admin"].Name)
	assert.Equal(t, "yao.login.Admin", Logins["admin"].Action.Process)
	assert.Equal(t, []string{":payload"}, Logins["admin"].Action.Args)
	assert.Equal(t, "yao.utils.Captcha", Logins["admin"].Layout.Captcha)
	assert.Equal(t, "/images/admin-cover.png", Logins["admin"].Layout.Cover)
	assert.Equal(t, "/x/Chart/dashboard", Logins["admin"].Layout.Entry)
	assert.Equal(t, "https://yaoapps.com", Logins["admin"].Layout.Site)
	assert.Equal(t, "夢想讓我們與眾不同", Logins["admin"].Layout.Slogan)

	assert.Equal(t, "user", Logins["user"].ID)
	assert.Equal(t, "用戶登錄", Logins["user"].Name)
	assert.Equal(t, "scripts.user.Login", Logins["user"].Action.Process)
	assert.Equal(t, []string{":payload"}, Logins["user"].Action.Args)
	assert.Equal(t, "yao.utils.Captcha", Logins["user"].Layout.Captcha)
	assert.Equal(t, "/images/user-cover.png", Logins["user"].Layout.Cover)
	assert.Equal(t, "/x/Table/pet", Logins["user"].Layout.Entry)
	assert.Equal(t, "https://yaoapps.com/docs", Logins["user"].Layout.Site)
	assert.Equal(t, "夢想讓我們與眾不同", Logins["user"].Layout.Slogan)
}

func TestLoadCN(t *testing.T) {

	os.Setenv("YAO_LANG", "zh-cn")
	lang.Load(config.Conf)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(Logins))

	assert.Equal(t, "admin", Logins["admin"].ID)
	assert.Equal(t, "管理员登录", Logins["admin"].Name)
	assert.Equal(t, "yao.login.Admin", Logins["admin"].Action.Process)
	assert.Equal(t, []string{":payload"}, Logins["admin"].Action.Args)
	assert.Equal(t, "yao.utils.Captcha", Logins["admin"].Layout.Captcha)
	assert.Equal(t, "/images/admin-cover.png", Logins["admin"].Layout.Cover)
	assert.Equal(t, "/x/Chart/dashboard", Logins["admin"].Layout.Entry)
	assert.Equal(t, "https://yaoapps.com", Logins["admin"].Layout.Site)
	assert.Equal(t, "梦想让我们与众不同", Logins["admin"].Layout.Slogan)

	assert.Equal(t, "user", Logins["user"].ID)
	assert.Equal(t, "用户登录", Logins["user"].Name)
	assert.Equal(t, "scripts.user.Login", Logins["user"].Action.Process)
	assert.Equal(t, []string{":payload"}, Logins["user"].Action.Args)
	assert.Equal(t, "yao.utils.Captcha", Logins["user"].Layout.Captcha)
	assert.Equal(t, "/images/user-cover.png", Logins["user"].Layout.Cover)
	assert.Equal(t, "/x/Table/pet", Logins["user"].Layout.Entry)
	assert.Equal(t, "https://yaoapps.com/docs", Logins["user"].Layout.Site)
	assert.Equal(t, "梦想让我们与众不同", Logins["user"].Layout.Slogan)
}

func TestExport(t *testing.T) {

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	err = Export()
	if err != nil {
		t.Fatal(err)
	}

	api, has := gou.APIs["widgets.login"]
	assert.True(t, has)
	assert.Equal(t, 4, len(api.HTTP.Paths))
}
