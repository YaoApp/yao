package tunnel

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/tai/tunnel/taipb"
	"github.com/yaoapp/yao/tai/types"
)

const defaultVNCPort = 5900

// forwardRoute holds the structured routing information extracted from the
// incoming request URL. It is passed to RequestForward so that Yao can
// populate the TunnelControl proto fields and Tai can route directly without
// parsing the first packet.
type forwardRoute struct {
	channelType   string // "proxy" | "vnc"
	containerID   string // target container or "__host__"
	containerPort int    // container-internal port (vnc default 5900)
	subpath       string // rewritten request path for the container
}

// HandleForward handles HTTP/VNC/any TCP-level forwarding through the gRPC tunnel.
// Route: ANY /tai/:taiID/proxy/*path  and  GET /tai/:taiID/vnc/*path
//
// It hijacks the browser's raw TCP connection, asks Tai to open a Forward stream
// with explicit routing information, rewrites the request path, and then performs
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

	route, err := resolveRoute(c, node)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rewrittenReq := rewriteRequest(c.Request, taiID, route)
	logger.Debug("[forward] "+node.Mode+" → tai",
		"tai_id", taiID,
		"type", route.channelType,
		"container", route.containerID,
		"container_port", route.containerPort,
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

	fwd, err := h.RequestForward(taiID, route)
	if err != nil {
		logger.Error("[forward] stream failed",
			"tai_id", taiID, "type", route.channelType, "err", err)
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

// resolveRoute extracts structured routing info from the request URL path.
//
// For proxy requests (/tai/:taiID/proxy/{containerID}:{port}/{subpath}):
//
//	channelType = "proxy", containerPort from URL, subpath = remaining path.
//
// For VNC requests (/tai/:taiID/vnc/{containerID}/ws):
//
//	channelType = "vnc", containerPort = 5900, subpath = /vnc/{containerID}/ws.
func resolveRoute(c *gin.Context, node *types.NodeMeta) (*forwardRoute, error) {
	path := c.Request.URL.Path
	taiID := c.Param("taiID")

	marker := "/tai/" + taiID
	idx := strings.Index(path, marker)
	if idx < 0 {
		return nil, fmt.Errorf("cannot locate /tai/%s in path", taiID)
	}
	rest := path[idx+len(marker):]

	if strings.HasPrefix(rest, "/vnc/") {
		// /vnc/{containerID}/ws → containerID, port=5900
		tail := strings.TrimPrefix(rest, "/vnc/")
		containerID := tail
		if slashIdx := strings.IndexByte(tail, '/'); slashIdx >= 0 {
			containerID = tail[:slashIdx]
		}
		if containerID == "" {
			return nil, fmt.Errorf("missing container ID in VNC path: %s", path)
		}
		return &forwardRoute{
			channelType:   "vnc",
			containerID:   containerID,
			containerPort: defaultVNCPort,
			subpath:       rest, // keep /vnc/{containerID}/ws
		}, nil
	}

	if strings.HasPrefix(rest, "/proxy/") {
		// /proxy/{containerID}:{port}/{subpath}
		proxyPath := strings.TrimPrefix(rest, "/proxy")
		// proxyPath = /{containerID}:{port}/{subpath}
		proxyPath = strings.TrimPrefix(proxyPath, "/")
		if proxyPath == "" {
			return nil, fmt.Errorf("empty proxy path")
		}

		slash := strings.IndexByte(proxyPath, '/')
		var head, subpath string
		if slash == -1 {
			head = proxyPath
			subpath = "/"
		} else {
			head = proxyPath[:slash]
			subpath = proxyPath[slash:]
		}

		colon := strings.LastIndexByte(head, ':')
		if colon < 0 {
			return nil, fmt.Errorf("missing port in proxy path: %s", path)
		}
		containerID := head[:colon]
		portStr := head[colon+1:]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q in proxy path: %w", portStr, err)
		}
		return &forwardRoute{
			channelType:   "proxy",
			containerID:   containerID,
			containerPort: port,
			subpath:       subpath,
		}, nil
	}

	return nil, fmt.Errorf("unknown route pattern: %s", rest)
}

// rewriteRequest clones the request and sets the path to the route's subpath.
//
// For proxy: the path becomes the subpath (e.g. /foo/bar).
// For VNC: the path keeps /vnc/{containerID}/ws as-is.
func rewriteRequest(orig *http.Request, taiID string, route *forwardRoute) *http.Request {
	r := orig.Clone(orig.Context())
	r.URL.Path = route.subpath
	r.RequestURI = r.URL.RequestURI()
	return r
}

// netConnAdapter wraps an io.ReadWriteCloser as needed by bridgeTCP.
type netConnAdapter struct {
	io.ReadWriteCloser
}
