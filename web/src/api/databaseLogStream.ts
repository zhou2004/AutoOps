import { tr } from "@/locales";
import { platformApiBase } from "./platformPermissions";
import { getToken, removeToken } from "@/utils/auth";
export interface DatabaseLogEvent {
  event: string;
  data: {
    text?: string;
    path?: string;
    message?: string;
    reason?: string;
    at?: number;
    bounded?: boolean;
  };
}
// Bound UTF-8 bytes as well as rows; CJK and emoji are not one byte per character.
export function appendLogText(current: string, chunk: string): string {
  const limit = 2 * 1024 * 1024;
  const combined = current + chunk;
  const rows = combined.endsWith("\n") ? 10001 : 10000;
  let text = combined.split("\n").slice(-rows).join("\n");
  const bytes = new TextEncoder().encode(text);
  if (bytes.length > limit) {
    let start = bytes.length - limit;
    while ((bytes[start] & 0xc0) === 0x80) start++;
    text = new TextDecoder().decode(bytes.subarray(start));
  }
  return text;
}
export async function databaseLogStream(
  clusterId: number,
  query: string,
  signal: AbortSignal,
  receive: (event: DatabaseLogEvent) => void
) {
  const response = await fetch(
    `${platformApiBase}/database/clusters/${clusterId}/logs/stream?${query}`,
    {
      headers: {
        Authorization: `Bearer ${getToken()}`,
        Accept: "text/event-stream"
      },
      signal
    }
  );
  if (response.status === 401) removeToken();
  if (
    !response.ok ||
    !response.headers.get("content-type")?.includes("text/event-stream")
  ) {
    const result = await response.json();
    if (result.code === 401) removeToken();
    throw new Error(result.message || result.msg || tr("日志连接失败"));
  }
  if (!response.body) throw new Error(tr("浏览器不支持日志流"));
  const reader = response.body.getReader(),
    decoder = new TextDecoder();
  let buffer = "",
    ended = false;
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      if (buffer.length > 2 * 1024 * 1024)
        throw new Error(tr("日志事件超过大小限制"));
      // SSE permits LF or CRLF, and frames/chinese UTF-8 may span network chunks.
      let match: RegExpExecArray | null;
      while ((match = /\r?\n\r?\n/.exec(buffer))) {
        const frame = buffer.slice(0, match.index);
        buffer = buffer.slice(match.index + match[0].length);
        let event = "message";
        const data: string[] = [];
        for (const line of frame.split(/\r?\n/)) {
          if (line.startsWith("event:")) event = line.slice(6).trim();
          if (line.startsWith("data:"))
            data.push(line.slice(5).replace(/^ /, ""));
        }
        if (data.length) {
          if (event === "end" || event === "error") ended = true;
          receive({ event, data: JSON.parse(data.join("\n")) });
        }
      }
    }
    if (!ended && !signal.aborted)
      throw new Error(tr("日志连接已中断，请重新连接"));
  } finally {
    await reader.cancel().catch(() => {});
    reader.releaseLock();
  }
}
