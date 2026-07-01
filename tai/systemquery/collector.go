package systemquery

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/yaoapp/yao/tai/systemquery/pb"
)

// Collector defines the interface for gathering system information.
type Collector interface {
	ListPorts(ctx context.Context) ([]*pb.PortInfo, error)
	ListProcesses(ctx context.Context, skipCPU bool) ([]*pb.ProcessInfo, *pb.SystemLoad, error)
}

// LocalCollector gathers system information from the local machine using gopsutil.
type LocalCollector struct{}

// NewLocalCollector creates a new LocalCollector.
func NewLocalCollector() *LocalCollector {
	return &LocalCollector{}
}

// ListPorts returns listening TCP/UDP ports on the local machine.
func (c *LocalCollector) ListPorts(ctx context.Context) ([]*pb.PortInfo, error) {
	conns, err := net.ConnectionsWithContext(ctx, "all")
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}

	var ports []*pb.PortInfo
	seen := make(map[string]bool)

	for _, conn := range conns {
		if conn.Status != "LISTEN" {
			continue
		}

		key := fmt.Sprintf("%s:%d:%s", conn.Laddr.IP, conn.Laddr.Port, protocolType(conn.Type))
		if seen[key] {
			continue
		}
		seen[key] = true

		info := &pb.PortInfo{
			Port:     int32(conn.Laddr.Port),
			Protocol: protocolType(conn.Type),
			Pid:      int32(conn.Pid),
			State:    conn.Status,
			Address:  conn.Laddr.IP,
		}

		if conn.Pid > 0 {
			if p, err := process.NewProcessWithContext(ctx, conn.Pid); err == nil {
				if name, err := p.NameWithContext(ctx); err == nil {
					info.Process = name
				}
				if cmdline, err := p.CmdlineWithContext(ctx); err == nil {
					info.Command = cmdline
				}
			}
		}

		ports = append(ports, info)
	}

	return ports, nil
}

// ListProcesses returns running processes and system load.
func (c *LocalCollector) ListProcesses(ctx context.Context, skipCPU bool) ([]*pb.ProcessInfo, *pb.SystemLoad, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list processes: %w", err)
	}

	var infos []*pb.ProcessInfo
	for _, p := range procs {
		info := &pb.ProcessInfo{
			Pid: int32(p.Pid),
		}

		if ppid, err := p.PpidWithContext(ctx); err == nil {
			info.Ppid = int32(ppid)
		}
		if username, err := p.UsernameWithContext(ctx); err == nil {
			info.User = username
		}
		if cmdline, err := p.CmdlineWithContext(ctx); err == nil {
			info.Command = cmdline
		}
		if status, err := p.StatusWithContext(ctx); err == nil && len(status) > 0 {
			info.State = status[0]
		}
		if !skipCPU {
			if cpuPct, err := p.CPUPercentWithContext(ctx); err == nil {
				info.CpuPercent = float32(cpuPct)
			}
		}
		if memPct, err := p.MemoryPercentWithContext(ctx); err == nil {
			info.MemPercent = memPct
		}
		if memInfo, err := p.MemoryInfoWithContext(ctx); err == nil && memInfo != nil {
			info.RssBytes = int64(memInfo.RSS)
			info.VszBytes = int64(memInfo.VMS)
		}
		if createTime, err := p.CreateTimeWithContext(ctx); err == nil {
			info.StartTime = createTime / 1000
		}
		if times, err := p.TimesWithContext(ctx); err == nil && times != nil {
			info.CpuTimeMs = int64((times.User + times.System) * 1000)
		}
		if threads, err := p.NumThreadsWithContext(ctx); err == nil {
			info.Threads = threads
		}
		if fds, err := p.NumFDsWithContext(ctx); err == nil {
			info.OpenFiles = fds
		}

		infos = append(infos, info)
	}

	sysLoad, err := c.collectLoad(ctx, skipCPU)
	if err != nil {
		return infos, nil, fmt.Errorf("collect load: %w", err)
	}

	return infos, sysLoad, nil
}

func (c *LocalCollector) collectLoad(ctx context.Context, skipCPU bool) (*pb.SystemLoad, error) {
	sysLoad := &pb.SystemLoad{}

	if avg, err := load.AvgWithContext(ctx); err == nil && avg != nil {
		sysLoad.Load1 = float32(avg.Load1)
		sysLoad.Load5 = float32(avg.Load5)
		sysLoad.Load15 = float32(avg.Load15)
	}

	if !skipCPU {
		if cpuPcts, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(cpuPcts) > 0 {
			sysLoad.CpuUsage = float32(cpuPcts[0])
		}
	}

	if counts, err := cpu.CountsWithContext(ctx, true); err == nil {
		sysLoad.CpuCount = int32(counts)
	}

	if vmem, err := mem.VirtualMemoryWithContext(ctx); err == nil && vmem != nil {
		sysLoad.MemTotal = int64(vmem.Total)
		sysLoad.MemUsed = int64(vmem.Used)
		sysLoad.MemAvailable = int64(vmem.Available)
	}

	if swap, err := mem.SwapMemoryWithContext(ctx); err == nil && swap != nil {
		sysLoad.SwapTotal = int64(swap.Total)
		sysLoad.SwapUsed = int64(swap.Used)
	}

	if uptime, err := host.UptimeWithContext(ctx); err == nil {
		sysLoad.UptimeSec = int64(uptime)
	}

	return sysLoad, nil
}

func protocolType(connType uint32) string {
	switch connType {
	case 1:
		return "tcp"
	case 2:
		return "udp"
	default:
		return fmt.Sprintf("unknown(%d)", connType)
	}
}
