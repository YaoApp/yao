package tunnel

import (
	"bytes"
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

	hijacker, ok := c.Writer.(http.Hijacker)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hijack not supported"})
		return
	}
	browserConn, bufrw, err := hijacker.Hijack()
	if err != nil {
		logger.Error("hijack failed", "err", err)
		return
	}
	defer browserConn.Close()

	fwd, err := h.RequestForward(taiID, targetPort)
	if err != nil {
		logger.Error("request forward failed",
			"tai_id", taiID, "port", targetPort, "err", err)
		browserConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	rewrittenReq := rewriteRequest(c.Request, taiID)

	var reqBuf bytes.Buffer
	rewrittenReq.Write(&reqBuf)
	if bufrw.Reader.Buffered() > 0 {
		buffered, _ := bufrw.Peek(bufrw.Reader.Buffered())
		reqBuf.Write(buffered)
	}
	if err := fwd.Send(&taipb.ForwardData{Data: reqBuf.Bytes()}); err != nil {
		logger.Error("send initial request", "err", err)
		return
	}

	streamConn := newForwardConn(fwd)
	bridgeTCP(
		&netConnAdapter{ReadWriteCloser: browserConn},
		streamConn,
	)
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

// rewriteRequest clones the request and strips everything up to and including
// /tai/:taiID from the path, handling any baseURL prefix (e.g. /v1/tai/abc/proxy/x → /proxy/x).
func rewriteRequest(orig *http.Request, taiID string) *http.Request {
	r := orig.Clone(orig.Context())

	marker := "/tai/" + taiID
	if idx := strings.Index(r.URL.Path, marker); idx >= 0 {
		r.URL.Path = r.URL.Path[idx+len(marker):]
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
	}

	r.RequestURI = r.URL.RequestURI()
	return r
}

// netConnAdapter wraps an io.ReadWriteCloser as needed by bridgeTCP.
type netConnAdapter struct {
	io.ReadWriteCloser
}
