function Create(
  ctx: Context,
  messages: Message[],
  options: CreateOptions
): CreateResponse | null {
  log.Info("[V2 Oneshot CLI] Create hook called");

  if (ctx.sandbox) {
    log.Info("[V2 Oneshot CLI] Sandbox available, workdir:", ctx.sandbox.workdir);
    ctx.sandbox.WriteFile("v2-marker.txt", "oneshot-cli-test");
  } else {
    log.Warn("[V2 Oneshot CLI] ctx.sandbox not available");
  }

  return { messages };
}

function Next(
  ctx: Context,
  payload: any,
  options: NextOptions
): NextResponse | null {
  log.Info("[V2 Oneshot CLI] Next hook called");
  return null;
}
