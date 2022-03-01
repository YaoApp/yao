function HexToStringString() {
  return Process("xiang.helper.HexToString", "ab");
}

function HexToStringBytes(data) {
  return Process("xiang.helper.HexToString", data);
}

function Buffer() {
  var buffer = new Uint8Array(2);
  buffer[0] = 0x0;
  buffer[1] = 0x1;
  return BufferToString(buffer);
}

function BufferToString(buffer) {
  return String.fromCharCode.apply(null, new Uint8Array(buffer));
}
