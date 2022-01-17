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
  last = columns.length - 1;
  ignore = 0;
  failure = 0;
  if (data.length > 1) {
    failure = 1;
  }
  for (var i in data) {
    if (data[i][last] == false) {
      ignore = ignore + 1;
    }
  }
  return [failure, ignore];
}
