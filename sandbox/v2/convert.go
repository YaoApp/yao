package sandbox

import (
	sqpb "github.com/yaoapp/yao/tai/systemquery/pb"
)

// convertPorts converts proto PortInfo slice to sandbox PortInfo pointer slice.
func convertPorts(pbPorts []*sqpb.PortInfo) []*PortInfo {
	if pbPorts == nil {
		return nil
	}
	result := make([]*PortInfo, 0, len(pbPorts))
	for _, p := range pbPorts {
		result = append(result, &PortInfo{
			Port:     int(p.Port),
			Protocol: p.Protocol,
			Process:  p.Process,
			PID:      int(p.Pid),
			State:    p.State,
			Address:  p.Address,
			Command:  p.Command,
		})
	}
	return result
}

// convertProcesses converts proto ProcessInfo slice to sandbox ProcessInfo pointer slice.
func convertProcesses(pbProcs []*sqpb.ProcessInfo) []*ProcessInfo {
	if pbProcs == nil {
		return nil
	}
	result := make([]*ProcessInfo, 0, len(pbProcs))
	for _, p := range pbProcs {
		result = append(result, &ProcessInfo{
			PID:        int(p.Pid),
			PPID:       int(p.Ppid),
			User:       p.User,
			Command:    p.Command,
			State:      p.State,
			CPUPercent: p.CpuPercent,
			MemPercent: p.MemPercent,
			RSSBytes:   p.RssBytes,
			VSZBytes:   p.VszBytes,
			StartTime:  p.StartTime,
			CPUTimeMs:  p.CpuTimeMs,
			Threads:    int(p.Threads),
			OpenFiles:  int(p.OpenFiles),
		})
	}
	return result
}

// convertLoad converts proto SystemLoad to sandbox SystemLoad.
// Returns nil if input is nil.
func convertLoad(pbLoad *sqpb.SystemLoad) *SystemLoad {
	if pbLoad == nil {
		return nil
	}
	return &SystemLoad{
		Load1:        pbLoad.Load1,
		Load5:        pbLoad.Load5,
		Load15:       pbLoad.Load15,
		MemTotal:     pbLoad.MemTotal,
		MemUsed:      pbLoad.MemUsed,
		MemAvailable: pbLoad.MemAvailable,
		SwapTotal:    pbLoad.SwapTotal,
		SwapUsed:     pbLoad.SwapUsed,
		CPUCount:     int(pbLoad.CpuCount),
		CPUUsage:     pbLoad.CpuUsage,
		UptimeSec:    pbLoad.UptimeSec,
	}
}
