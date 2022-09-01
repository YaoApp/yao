/**
 * Export Models, APIs, Flows, Tables, Tasks, Schedules, etc.
 */

/**
 * Export Apis
 * @param {*} dsl
 * @returns
 */
function Apis() {
  let apis = {};
  apis[`dyform`] = dyformAPI();
  return apis;
}

/**
 * Export Models
 * @param {*} dsl
 * @returns
 */
function Models(name, dsl) {
  let models = {};
  models[`dyform.${name}`] = dyformModel(name);
  return models;
}

/**
 * Export Flows
 * @param {*} dsl
 * @returns
 */
function Flows(name, dsl) {
  return [{}];
}

/**
 * Export Tables
 * @param {*} dsl
 * @returns
 */
function Tables(name, dsl) {
  let tables = {};
  tables[`dyform.${name}`] = dyformTable(name);
  return tables;
}

/**
 * Export Tasks
 * @param {*} dsl
 * @returns
 */
function Tasks(name, dsl) {
  let tasks = {};
  tasks[`dyform.${name}`] = dyformTask(name);
  return tasks;
}

/**
 * Export Schedules
 * @param {*} dsl
 * @returns
 */
function Schedules(name, dsl) {
  let schedules = {};
  schedules[`dyform.${name}`] = dyformSchedule(name);
  return schedules;
}

function dyformSchedule(name) {
  return {
    name: `Schedule ${name}`,
    schedule: "*/1 * * * *",
    process: `widgets.dyform.Save`,
    args: [name],
  };
}

function dyformTask(name) {
  return {
    name: `Task ${name}`,
    worker_nums: 2,
    attempts: 3,
    attempt_after: 200,
    timeout: 2,
    size: 1000,
    process: `widgets.dyform.Save`,
    args: [name],
  };
}

function dyformAPI() {
  return {
    name: `dyform API`,
    version: "1.0.0",
    description: "dyform APIs",
    group: "dyform",
    guard: "-",
    paths: [
      {
        path: "/:name/search",
        method: "GET",
        process: `widgets.dyform.Save`,
        in: ["$param.name", ":query"],
        out: { status: 200, type: "application/json" },
      },
    ],
  };
}

function dyformTable(name) {
  return {
    name: `dyform_${name}`,
    version: "1.0.0",
    decription: "云服务库",
    bind: { model: `dyform.${name}` },
    apis: {},
    columns: {},
    filters: {},
    list: {
      primary: "id",
      layout: { columns: [{ name: "ID", width: 6 }], filters: [] },
      actions: {},
    },
    edit: {
      primary: "id",
      layout: { fieldset: [{ columns: [{ name: "ID", width: 6 }] }] },
      actions: { cancel: {} },
    },
    insert: {},
    view: {},
  };
}

function dyformModel(name) {
  return {
    table: { name: `dyform_${name}` },
    columns: [
      { label: "DYFORM ID", name: "id", type: "ID" },
      { label: "SN", name: "sn", type: "string", length: 20, unique: true },
      { label: "NAME", name: "name", type: "string", length: 200, index: true },
      { label: "SOURCE", name: "source", type: "JSON", nullable: true },
      {
        label: "TITLE",
        name: "title",
        type: "string",
        length: 200,
        index: true,
      },
    ],
    indexes: [],
  };
}
