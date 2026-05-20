function Create(
  ctx: Context,
  messages: Message[],
  options: CreateOptions
): CreateResponse | null {
  return { messages };
}

function Next(
  ctx: Context,
  payload: any,
  options: NextOptions
): NextResponse | null {
  const results: Record<string, any> = {
    has_computer: !!ctx.computer,
    has_workspace: !!ctx.workspace,
  };

  if (ctx.computer) {
    results.computer_id = ctx.computer.id || "";

    try {
      const info = ctx.computer.Info();
      results.computer_info = {
        kind: info.kind,
        node_id: info.node_id,
        status: info.status,
        os: info.system?.os || "",
      };
    } catch (e) {
      results.computer_info_error = String(e);
    }

    try {
      const execResult = ctx.computer.Exec(["echo", "jsapi-v2-test"]);
      results.exec_stdout = (execResult.stdout || "").trim();
      results.exec_exit_code = execResult.exit_code;
    } catch (e) {
      results.exec_error = String(e);
    }
  }

  if (ctx.workspace) {
    try {
      ctx.workspace.WriteFile("jsapi-test.txt", "hello from jsapi v2");
      const content = ctx.workspace.ReadFile("jsapi-test.txt");
      results.write_read_ok = content === "hello from jsapi v2";
      results.read_content = content;
    } catch (e) {
      results.write_read_error = String(e);
    }

    try {
      ctx.workspace.MkdirAll("sub/dir");
      ctx.workspace.WriteFile("sub/dir/nested.txt", "nested content");
      const exists = ctx.workspace.Exists("sub/dir/nested.txt");
      results.mkdir_exists_ok = exists;
    } catch (e) {
      results.mkdir_exists_error = String(e);
    }

    try {
      const entries = ctx.workspace.ReadDir(".");
      results.readdir_count = entries.length;
      results.readdir_names = entries.map((e: any) => e.name);
    } catch (e) {
      results.readdir_error = String(e);
    }

    try {
      const stat = ctx.workspace.Stat("jsapi-test.txt");
      results.stat_ok = stat.name === "jsapi-test.txt" && !stat.is_dir && stat.size > 0;
      results.stat = { name: stat.name, size: stat.size, is_dir: stat.is_dir };
    } catch (e) {
      results.stat_error = String(e);
    }

    try {
      ctx.workspace.Copy("jsapi-test.txt", "jsapi-test-copy.txt");
      const copied = ctx.workspace.ReadFile("jsapi-test-copy.txt");
      results.copy_ok = copied === "hello from jsapi v2";
    } catch (e) {
      results.copy_error = String(e);
    }

    try {
      ctx.workspace.Rename("jsapi-test-copy.txt", "jsapi-test-renamed.txt");
      const renamed = ctx.workspace.Exists("jsapi-test-renamed.txt");
      const oldGone = !ctx.workspace.Exists("jsapi-test-copy.txt");
      results.rename_ok = renamed && oldGone;
    } catch (e) {
      results.rename_error = String(e);
    }

    try {
      ctx.workspace.Remove("jsapi-test-renamed.txt");
      const gone = !ctx.workspace.Exists("jsapi-test-renamed.txt");
      results.remove_ok = gone;
    } catch (e) {
      results.remove_error = String(e);
    }
  }

  return { data: results };
}
