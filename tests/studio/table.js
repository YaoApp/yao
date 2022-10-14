/**
 * Create table
 * @param {*} name
 */
function Make(name) {
  var fs = new FS("dsl");
  var model = GetModel(name);
  var table = JSON.stringify({
    name: `${model.name} Admin`,
    actions: { bind: { model: name } },
  });
  fs.WriteFile(`/tables/auto/${name}.tab.json`, table);
}

/**
 * Get Model
 * @param {*} name
 * @returns
 */
function Model(name) {
  var fs = new FS("dsl");
  var file = `/models/${name}.mod.json`;
  var data = fs.ReadFile(file);
  return JSON.parse(data);
}

/**
 * for unit tests
 * @returns
 */
function Ping() {
  return "PONG";
}

/**
 * for unit tests
 * @param  {...any} args
 * @returns
 */
function UnitTest(...args) {
  if (args.length > 0 && args[0] == "throw-test") {
    throw new Exception("I'm a teapot", 418);
  }
  return args;
}
