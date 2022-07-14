function onData(data, recvLen) {
  console.log(`Data: ${data} ${recvLen}`);
  log.Trace("onData: %v %v", data, recvLen);
}

function onError(err) {
  console.log(`Error: ${err} `);
}

function onClosed(data, err) {
  console.log(`Closed: ${data} ${err} `);
}

function onConnected(option) {
  console.log("onConnected", option);
}

// Convert a hex string to a byte array
function hexToBytes(hex) {
  for (var bytes = [], c = 0; c < hex.length; c += 2) {
    bytes.push(parseInt(hex.substr(c, 2), 16));
  }
  return bytes;
}

// Convert a byte array to a hex string
function bytesToHex(bytes) {
  for (var hex = [], i = 0; i < bytes.length; i++) {
    var current = bytes[i] < 0 ? bytes[i] + 256 : bytes[i];
    hex.push((current >>> 4).toString(16));
    hex.push((current & 0xf).toString(16));
  }
  return hex.join("");
}
