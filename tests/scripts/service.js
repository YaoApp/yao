function BeforeSearch(param, page, pagesize) {
  return [param, page, pagesize];
}

function AfterSearch(data) {
  var newData = data || {};
  var items = data.data || [];
  for (var i in items) {
    items[i]["city"] = `ID: ${items[i]["id"]} 城市: ${items[i]["city"]} `;
    var res = Process("models.service.Find", items[i]["id"], {});
    // console.log(
    //   `id:${items[i]["id"]} ${res["name"]} page: ${newData["page"]} pagesize: ${newData["pagesize"]} pagecnt: ${newData["pagecnt"]}  next:${newData["next"]}`
    // );
  }
  newData["data"] = items;
  return newData;
}
