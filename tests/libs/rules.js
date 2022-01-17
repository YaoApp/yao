// 校验数值
function order_sn(value, row) {
  value = value || null;
  row = row || {};
  if (value == null) {
    return false;
  }
  return row;
}

function FmtUser(value, row) {
  row[9] = "自动添加备注 @From " + value;
  return row;
}

function FmtGoods(value, row) {
  return row;
}

function mobile(value, row) {
  return row;
}

function ImportData(columns, data) {
  return 1;
}
