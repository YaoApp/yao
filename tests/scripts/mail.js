var id = 1024;

/**
 * Generate job id
 * @returns
 */
function NextID() {
  id = id + 1;
  console.log(`NextID: ${id}`);
  return id;
}

function Send(id, mail, flag) {
  for (var i = 1; i <= 3; i++) {
    Process("xiang.system.Sleep", 200);
    Process("tasks.mail.Progress", id, i, 3, "unit-test");
  }
  if (flag) {
    console.log(`flag: ${flag}`);
    Process("xiang.system.Sleep", 2000);
  }
  console.log(
    `Send: ${JSON.stringify({ foo: "bar", mail: mail, flag: flag || "-" })}`
  );
  return { foo: "bar", mail: mail, flag: flag || "-" };
}

/**
 * OnAdd add event
 * @param {*} id
 */
function OnAdd(id) {
  console.log(`OnAdd: #${id}`);
}

/**
 * OnProgress
 * @param {*} id
 * @param {*} current
 * @param {*} total
 * @param {*} message
 */
function OnProgress(id, current, total, message) {
  console.log(`OnProgress: #${id} ${message} ${current}/${total} `);
}

function OnSuccess(id, res) {
  console.log(`OnSuccess: #${id} ${JSON.stringify(res)}`);
}

function OnError(id, err) {
  console.log(`OnError: #${id} ${err}`);
}
