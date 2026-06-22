package board

import (
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/share"
)

func tableBoard() string {
	if m, err := model.Get("__yao.agent.board"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_board"
}

func tableBoardColumn() string {
	if m, err := model.Get("__yao.agent.board_column"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_board_column"
}

func tableTask() string {
	if m, err := model.Get("__yao.agent.task"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_task"
}
