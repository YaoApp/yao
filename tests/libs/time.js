function now() {
  return new Date().toISOString().split("T")[0];
}

function lastYear() {
  var d = new Date();
  d.setFullYear(d.getFullYear() - 1);
  return d.toISOString().split("T")[0];
}

function hello(name) {
  return "name:" + name;
}
