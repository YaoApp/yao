package inbox

import (
	inboxsvc "github.com/yaoapp/yao/agent/inbox"
	inboxtools "github.com/yaoapp/yao/tools/inbox"
)

func init() {
	inboxtools.FnList = inboxsvc.List
	inboxtools.FnRead = inboxsvc.Read
}
