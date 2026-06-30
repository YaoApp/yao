package task

import (
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/share"
)

func tableTask() string {
	if m, err := model.Get("__yao.agent.task"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_task"
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

func tableMail() string {
	if m, err := model.Get("__yao.agent.mail"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_mail"
}

func tableExecution() string {
	if m, err := model.Get("__yao.agent.execution"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_execution"
}

func tableMessage() string {
	if m, err := model.Get("__yao.agent.message"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_message"
}

func tableScheduleLog() string {
	if m, err := model.Get("__yao.agent.schedule_log"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_schedule_log"
}

func tableAssistant() string {
	if m, err := model.Get("__yao.agent.assistant"); err == nil && m.MetaData.Table.Name != "" {
		return m.MetaData.Table.Name
	}
	return share.App.Prefix + "agent_assistant"
}
