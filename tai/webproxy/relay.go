package webproxy

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// relayTransport implements http.RoundTripper using the tai relay CONNECT protocol.
type relayTransport struct {
	containerID string
	targetPort  int
	relayAddr   string // address of the relay service (host:port)
}

func (t *relayTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	conn, err := dialRelay(t.relayAddr, t.targetPort)
	if err != nil {
		return nil, err
	}

	// Write the request to the relay tunnel
	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, err
	}

	// Read the response
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Attach conn to response body so it gets closed when body is read
	resp.Body = &connCloser{resp.Body, conn}
	return resp, nil
}

// dialRelay establishes a TCP connection through the relay CONNECT protocol.
func dialRelay(relayAddr string, targetPort int) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", relayAddr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("relay dial failed: %w", err)
	}

	// Send CONNECT command
	connectLine := fmt.Sprintf("CONNECT 127.0.0.1:%d\n", targetPort)
	if _, err := conn.Write([]byte(connectLine)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("relay CONNECT write failed: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("relay response read failed: %w", err)
	}

	if strings.TrimSpace(line) != "OK" {
		conn.Close()
		return nil, fmt.Errorf("relay rejected: %s", strings.TrimSpace(line))
	}

	// If there's buffered data from the reader, wrap the connection
	if reader.Buffered() > 0 {
		return &bufferedConn{Conn: conn, reader: reader}, nil
	}

	return conn, nil
}

// connCloser wraps a ReadCloser and an additional Closer to close when done.
type connCloser struct {
	body interface{ Read([]byte) (int, error) }
	conn net.Conn
}

func (c *connCloser) Read(p []byte) (int, error) {
	return c.body.Read(p)
}

func (c *connCloser) Close() error {
	return c.conn.Close()
}

// bufferedConn wraps a net.Conn with a buffered reader for leftover data.
type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (bc *bufferedConn) Read(p []byte) (int, error) {
	return bc.reader.Read(p)
}
