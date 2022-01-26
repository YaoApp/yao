function main(args, out, res) {
  // 100000012021128798321101 15 2d 02 f2 96 72 1e 52 b9 cd
  var idstr = args[0] || "";
  var len = idstr.length;

  if (len != 36) {
    return;
  }
  id = BigInt("0x" + idstr.substring(8, len - 8).toUpperCase()).toString(10);
  console.log(id);
}
