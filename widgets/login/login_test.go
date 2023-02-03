package login

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	i18n.Load(config.Conf)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(Logins))

	assert.Equal(t, "admin", Logins["admin"].ID)
	assert.Equal(t, "::Admin Login", Logins["admin"].Name)
	assert.Equal(t, "yao.login.Admin", Logins["admin"].Action.Process)
	assert.Equal(t, []interface{}{":payload"}, Logins["admin"].Action.Args)
	assert.Equal(t, "yao.utils.Captcha", Logins["admin"].Layout.Captcha)
	assert.Equal(t, "/assets/images/login/cover.svg", Logins["admin"].Layout.Cover)
	assert.Equal(t, "/x/Chart/dashboard", Logins["admin"].Layout.Entry)
	assert.Equal(t, "https://yaoapps.com", Logins["admin"].Layout.Site)
	assert.Equal(t, "::Make Your Dream With Yao App Engine", Logins["admin"].Layout.Slogan)

	assert.Equal(t, "user", Logins["user"].ID)
	assert.Equal(t, "::User Login", Logins["user"].Name)
	assert.Equal(t, "scripts.user.Login", Logins["user"].Action.Process)
	assert.Equal(t, []interface{}{":payload"}, Logins["user"].Action.Args)
	assert.Equal(t, "yao.utils.Captcha", Logins["user"].Layout.Captcha)
	assert.Equal(t, "/assets/images/login/cover.svg", Logins["user"].Layout.Cover)
	assert.Equal(t, "/x/Table/pet", Logins["user"].Layout.Entry)
	assert.Equal(t, "https://yaoapps.com/doc", Logins["user"].Layout.Site)
	assert.Equal(t, "::Make Your Dream With Yao App Engine", Logins["user"].Layout.Slogan)
}

func TestLoadHK(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// runtime.Load(config.Conf)
	i18n.Load(config.Conf)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(Logins))
	assert.Equal(t, "admin", Logins["admin"].ID)
	assert.Equal(t, "::Admin Login", Logins["admin"].Name)
	assert.Equal(t, "yao.login.Admin", Logins["admin"].Action.Process)
	assert.Equal(t, []interface{}{":payload"}, Logins["admin"].Action.Args)
	assert.Equal(t, "yao.utils.Captcha", Logins["admin"].Layout.Captcha)
	assert.Equal(t, "/assets/images/login/cover.svg", Logins["admin"].Layout.Cover)
	assert.Equal(t, "/x/Chart/dashboard", Logins["admin"].Layout.Entry)
	assert.Equal(t, "https://yaoapps.com", Logins["admin"].Layout.Site)
	assert.Equal(t, "::Make Your Dream With Yao App Engine", Logins["admin"].Layout.Slogan)

	assert.Equal(t, "user", Logins["user"].ID)
	assert.Equal(t, "::User Login", Logins["user"].Name)
	assert.Equal(t, "scripts.user.Login", Logins["user"].Action.Process)
	assert.Equal(t, []interface{}{":payload"}, Logins["user"].Action.Args)
	assert.Equal(t, "yao.utils.Captcha", Logins["user"].Layout.Captcha)
	assert.Equal(t, "/assets/images/login/cover.svg", Logins["user"].Layout.Cover)
	assert.Equal(t, "/x/Table/pet", Logins["user"].Layout.Entry)
	assert.Equal(t, "https://yaoapps.com/doc", Logins["user"].Layout.Site)
	assert.Equal(t, "::Make Your Dream With Yao App Engine", Logins["user"].Layout.Slogan)

	adminV, err := i18n.Trans("zh-hk", []string{"login.admin"}, Logins["admin"])
	admin := adminV.(*DSL)
	assert.Equal(t, "admin", admin.ID)
	assert.Equal(t, "管理員登錄", admin.Name)
	assert.Equal(t, "yao.login.Admin", admin.Action.Process)
	assert.Equal(t, []interface{}{":payload"}, admin.Action.Args)
	assert.Equal(t, "yao.utils.Captcha", admin.Layout.Captcha)
	assert.Equal(t, "/assets/images/login/cover.svg", admin.Layout.Cover)
	assert.Equal(t, "/x/Chart/dashboard", admin.Layout.Entry)
	assert.Equal(t, "https://yaoapps.com", admin.Layout.Site)
	assert.Equal(t, "和 Yao App Engine 一起，為夢想而努力", admin.Layout.Slogan)

	userV, err := i18n.Trans("zh-hk", []string{"login.user"}, Logins["user"])
	user := userV.(*DSL)
	assert.Equal(t, "user", user.ID)
	assert.Equal(t, "用戶登錄", user.Name)
	assert.Equal(t, "scripts.user.Login", user.Action.Process)
	assert.Equal(t, []interface{}{":payload"}, user.Action.Args)
	assert.Equal(t, "yao.utils.Captcha", user.Layout.Captcha)
	assert.Equal(t, "/assets/images/login/cover.svg", user.Layout.Cover)
	assert.Equal(t, "/x/Table/pet", user.Layout.Entry)
	assert.Equal(t, "https://yaoapps.com/doc", user.Layout.Site)
	assert.Equal(t, "和 Yao App Engine 一起，為夢想而努力", user.Layout.Slogan)
}

func TestLoadCN(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	os.Setenv("YAO_LANG", "zh-cn")
	i18n.Load(config.Conf)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(Logins))

	adminV, err := i18n.Trans("zh-cn", []string{"login.admin"}, Logins["admin"])
	admin := adminV.(*DSL)

	assert.Equal(t, "admin", admin.ID)
	assert.Equal(t, "管理员登录", admin.Name)
	assert.Equal(t, "yao.login.Admin", admin.Action.Process)
	assert.Equal(t, []interface{}{":payload"}, admin.Action.Args)
	assert.Equal(t, "yao.utils.Captcha", admin.Layout.Captcha)
	assert.Equal(t, "/assets/images/login/cover.svg", admin.Layout.Cover)
	assert.Equal(t, "/x/Chart/dashboard", admin.Layout.Entry)
	assert.Equal(t, "https://yaoapps.com", admin.Layout.Site)
	assert.Equal(t, "和 Yao App Engine 一起，为梦想而努力", admin.Layout.Slogan)

	userV, err := i18n.Trans("zh-cn", []string{"login.user"}, Logins["user"])
	user := userV.(*DSL)

	assert.Equal(t, "user", user.ID)
	assert.Equal(t, "用户登录", user.Name)
	assert.Equal(t, "scripts.user.Login", user.Action.Process)
	assert.Equal(t, []interface{}{":payload"}, user.Action.Args)
	assert.Equal(t, "yao.utils.Captcha", user.Layout.Captcha)
	assert.Equal(t, "/assets/images/login/cover.svg", user.Layout.Cover)
	assert.Equal(t, "/x/Table/pet", user.Layout.Entry)
	assert.Equal(t, "https://yaoapps.com/doc", user.Layout.Site)
	assert.Equal(t, "和 Yao App Engine 一起，为梦想而努力", user.Layout.Slogan)
}

func TestExport(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	err = Export()
	if err != nil {
		t.Fatal(err)
	}

	api, has := api.APIs["widgets.login"]
	assert.True(t, has)
	assert.Equal(t, 4, len(api.HTTP.Paths))
}
