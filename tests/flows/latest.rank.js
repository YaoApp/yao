// 结果集处理脚本
function main(args, manus, data) {
  // 解析厂商ID
  var manu_ids = [];
  for (var i in manus) {
    var id = manus[i].id;
    if (id) {
      manu_ids.push(id);
    }
  }

  // 处理返回结果
  var res = {
    manus: manus,
    manu_ids: manu_ids,
  };
  return res;
}
