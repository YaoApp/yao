"""Minimal SSE server on port 9801 using only stdlib.
Sends a 'hello' event every second, up to 5 events then closes."""
import http.server
import time

class SSEHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "keep-alive")
        self.end_headers()

        for i in range(5):
            msg = f"data: hello-{i}\n\n"
            try:
                self.wfile.write(msg.encode())
                self.wfile.flush()
            except BrokenPipeError:
                return
            time.sleep(0.2)

    def log_message(self, format, *args):
        pass

if __name__ == "__main__":
    server = http.server.HTTPServer(("0.0.0.0", 9801), SSEHandler)
    server.serve_forever()
