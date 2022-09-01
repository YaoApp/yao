function Hello() {
  var ws = new WebSocket("ws://127.0.0.1:5093/websocket/chat", "p0");
  var response = ws.push("Hello World");
  return response;
}

const host = "127.0.0.1:5099";

/**
 * WebSocket Client EventMode(daemon)
 */
function Event() {
  var url = `ws://${host}/websocket/chat`;
  var ws = new WebSocket(url, "yao-chat-01");

  // Connection opened
  ws.on("open", function (event) {
    ws.send("Hello Server!", event.data);
  });

  // Listen for messages
  ws.on("message", function (event) {
    console.log("Message from server ", event.data);
    ws.close(200, "Bye");
  });

  // Listen for error
  ws.on("error", (event) => {
    console.log("Message from server ", event);
  });

  // Listen for close
  ws.on("close", (event) => {
    console.log("The connection has been closed successfully");
  });
}

/**
 * WebSocket Client PushMode
 */
function Push() {
  var url = `ws://${host}/websocket/chat`;
  var ws = new WebSocket(url, "yao-chat-01");
  var message = ws.push("Hello Server!");
  console.log(message);
}

/**
 * WebSocket Client JWT Auth
 */
function Token() {
  var token = "xxx";
  var url = `ws://${host}/websocket/chat?token=${token}`;
  var ws = new WebSocket(url, "yao-chat-01");
  var message = ws.push("Hello Server!");
  console.log(message);
}

/**
 * WebSocket Client Basic Auth
 */
function Basic() {
  // Basic Auth username:password
  var user = "test";
  var password = "WsTest123**";
  var url = `ws://${user}:${password}@${host}/websocket/chat?token=${token}`;
  var ws = new WebSocket(url, "yao-chat-01");
  var message = ws.push("Hello Server!");
  console.log(message);
}

function onData(data, recvLen) {
  console.log(`Data: ${data} ${recvLen}`);
  log.Trace("onData: %v %v", data, recvLen);
  if (data[0] == "1") {
    Process("websocket.Close", "message");
  }
}

function onError(err) {
  console.log(`Error: ${err} `);
}

function onClosed(data, err) {
  console.log(`Closed: ${data} ${err} `);
}

function onConnected(option) {
  console.log("onConnected", option);
  Process("websocket.Write", "message", "1|Hello World");
}
