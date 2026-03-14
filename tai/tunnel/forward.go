package tunnel

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/tai/tunnel/taipb"
	"github.com/yaoapp/yao/tai/types"
)

// HandleForward handles HTTP/VNC/any TCP-level forwarding through the gRPC tunnel.
// Route: ANY /tai/:taiID/proxy/*path  and  GET /tai/:taiID/vnc/*path
//
// It hijacks the browser's raw TCP connection, asks Tai to open a Forward stream
// to the resolved target port, rewrites the request path, and then performs
// bidirectional byte-level bridging. No protocol parsing beyond HTTP hijack.
func (h *TunnelHandler) HandleForward(c *gin.Context) {
	logger := h.logger
	reg := h.reg

	taiID := c.Param("taiID")

	node, ok := reg.Get(taiID)
	if !ok || node.Status != "online" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "tai node not available"})
		return
	}

	targetPort := resolveTargetPort(c, node)
	if targetPort == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot resolve target port"})
		return
	}

	rewrittenReq := rewriteRequest(c.Request, taiID)
	logger.Debug("[forward] "+node.Mode+" → tai:"+fmt.Sprintf("%d", targetPort),
		"tai_id", taiID,
		"addr", node.Addr,
		"path", rewrittenReq.URL.Path,
	)

	hijacker, ok := c.Writer.(http.Hijacker)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hijack not supported"})
		return
	}
	browserConn, bufrw, err := hijacker.Hijack()
	if err != nil {
		logger.Error("[forward] hijack failed", "tai_id", taiID, "err", err)
		return
	}
	defer browserConn.Close()

	fwd, err := h.RequestForward(taiID, targetPort)
	if err != nil {
		logger.Error("[forward] stream failed",
			"tai_id", taiID, "port", targetPort, "err", err)
		browserConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	var reqBuf bytes.Buffer
	rewrittenReq.Write(&reqBuf)
	if bufrw.Reader.Buffered() > 0 {
		buffered, _ := bufrw.Peek(bufrw.Reader.Buffered())
		reqBuf.Write(buffered)
	}
	if err := fwd.Send(&taipb.ForwardData{Data: reqBuf.Bytes()}); err != nil {
		logger.Error("[forward] send failed", "tai_id", taiID, "err", err)
		return
	}

	streamConn := newForwardConn(fwd)
	bridgeTCP(
		&netConnAdapter{ReadWriteCloser: browserConn},
		streamConn,
	)
	logger.Debug("[forward] closed", "tai_id", taiID)
}

// HandleForwardLazy is a gin.HandlerFunc that resolves the global TunnelHandler
// at call time (not registration time), so routes can be registered before the
// gRPC server starts.
func HandleForwardLazy(c *gin.Context) {
	h := GlobalHandler()
	if h == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "tunnel handler not initialized"})
		return
	}
	h.HandleForward(c)
}

// resolveTargetPort determines the Tai-side port from the route pattern.
func resolveTargetPort(c *gin.Context, node *types.NodeMeta) int {
	path := c.Request.URL.Path

	if strings.Contains(path, "/vnc/") {
		if node.Ports.VNC != 0 {
			return node.Ports.VNC
		}
		return 16080
	}
	if strings.Contains(path, "/proxy/") {
		if node.Ports.HTTP != 0 {
			return node.Ports.HTTP
		}
		return 8099
	}
	return 0
}

// rewriteRequest clones the request and strips the Yao-side route prefix,
// leaving only what the Tai-side handler expects.
//
// The Tai httpproxy expects /{containerID}:{port}/..., so the /proxy prefix
// is stripped. The Tai VNC router expects /vnc/{containerID}/ws, so the /vnc
// prefix is kept.
//
//	/v1/tai/abc/proxy/cid:8080/foo → /cid:8080/foo
//	/v1/tai/abc/vnc/cid/ws        → /vnc/cid/ws
func rewriteRequest(orig *http.Request, taiID string) *http.Request {
	r := orig.Clone(orig.Context())

	marker := "/tai/" + taiID
	if idx := strings.Index(r.URL.Path, marker); idx >= 0 {
		rest := r.URL.Path[idx+len(marker):]
		rest = strings.TrimPrefix(rest, "/proxy")
		if rest == "" {
			rest = "/"
		}
		r.URL.Path = rest
	}

	r.RequestURI = r.URL.RequestURI()
	return r
}

// netConnAdapter wraps an io.ReadWriteCloser as needed by bridgeTCP.
type netConnAdapter struct {
	io.ReadWriteCloser
}
