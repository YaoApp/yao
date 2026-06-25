package inbox

import (
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/share"
)

func tableMail() string {
	if m, err := model.Get("__yao.agent.mail"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_mail"
}

func tableChat() string {
	if m, err := model.Get("__yao.agent.chat"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_chat"
}

func tableBoardColumn() string {
	if m, err := model.Get("__yao.agent.board_column"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_board_column"
}

func tableBoard() string {
	if m, err := model.Get("__yao.agent.board"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_board"
}

func tableTask() string {
	if m, err := model.Get("__yao.agent.task"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_task"
}
