package tai

import (
	"errors"
	"net"

	hepb "github.com/yaoapp/yao/tai/hostexec/pb"
	"github.com/yaoapp/yao/tai/proxy"
	"github.com/yaoapp/yao/tai/runtime"
	"github.com/yaoapp/yao/tai/types"
	"github.com/yaoapp/yao/tai/vnc"
	"github.com/yaoapp/yao/tai/volume"
	"google.golang.org/grpc"
)

// ConnResources holds bare connection resources for a Tai node.
// Returned by Dial* functions. Caller (usually registry) is responsible
// for calling Close() when the node disconnects or resources are replaced.
type ConnResources struct {
	GRPCConn *grpc.ClientConn
	Runtime  runtime.Runtime
	Image    runtime.Image
	HostExec hepb.HostExecClient
	Volume   volume.Volume
	Proxy    proxy.Proxy
	VNC      vnc.VNC
	Caps     types.Capabilities
	System   types.SystemInfo
	Ports    types.Ports
	Version  string
	DataDir  string // host-side data dir (local mode only)

	// Tunnel mode: local listeners that bridge to Tai via WS.
	Listeners []net.Listener
}

// Close releases all held resources. Safe to call with nil fields.
func (r *ConnResources) Close() error {
	if r == nil {
		return nil
	}
	var errs []error
	if r.Runtime != nil {
		if err := r.Runtime.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.Volume != nil {
		if err := r.Volume.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	for _, ln := range r.Listeners {
		ln.Close()
	}
	if r.GRPCConn != nil {
		if err := r.GRPCConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
