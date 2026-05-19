function Next(payload: { data: any; response: string }) {
  if (payload.response && payload.response.includes("need more information")) {
    return { action: "need_input", message: "Please provide more details." };
  }
  return null;
}
