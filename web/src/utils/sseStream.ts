/**
 * AutoOps SSE 流式请求工具
 * 使用原生 fetch + ReadableStream 接收 SSE 流，绕过 axios 拦截器
 * 
 * 用法：
 *   const stream = createSSEStream('/task/ansible/1/log/2', {
 *     onMessage: (line) => console.log(line),
 *     onError: (err) => console.error(err),
 *     onClose: () => console.log('closed')
 *   });
 *   // 关闭连接
 *   stream.close();
 */

import { getToken } from "@/utils/auth";

interface SSEHandlers {
    onOpen?: () => void;
    onMessage?: (line: string) => void;
    onError?: (error: any) => void;
    onClose?: () => void;
}

export function createSSEStream(endpoint: string, handlers: SSEHandlers = {}) {
    let aborted = false;
    let reader: ReadableStreamDefaultReader | null = null;

    const API_HOST = import.meta.env.VITE_API_BASE_URL || "";
    const API_PREFIX = "/api/v1";

    // 构建完整 URL
    let fullUrl: string;
    if (endpoint.startsWith("http")) {
        fullUrl = endpoint;
    } else {
        const base = API_HOST.replace(/\/$/, "");
        const path = endpoint.startsWith("/") ? endpoint : "/" + endpoint;
        const withPrefix = path.startsWith(API_PREFIX) ? path : API_PREFIX + path;
        fullUrl = base + withPrefix;
    }

    const token = getToken();

    async function connect() {
        try {
            const headers: Record<string, string> = {
                Accept: "text/event-stream",
            };
            if (token) {
                headers["Authorization"] = "Bearer " + token;
            }

            const response = await fetch(fullUrl, { headers });

            if (!response.ok) {
                throw new Error(`SSE connection failed: ${response.status}`);
            }

            if (handlers.onOpen) {
                handlers.onOpen();
            }

            const body = response.body;
            if (!body) {
                throw new Error("ReadableStream not supported");
            }

            reader = body.getReader();
            const decoder = new TextDecoder();
            let buffer = "";

            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split("\n");
                buffer = lines.pop() || "";

                for (const line of lines) {
                    const trimmed = line.trim();
                    if (trimmed === "") continue;

                    // SSE 格式: "data: xxx"
                    let data = trimmed;
                    if (data.startsWith("data:")) {
                        data = data.substring(5).trim();
                    }

                    if (data && !aborted && handlers.onMessage) {
                        handlers.onMessage(data);
                    }
                }
            }
        } catch (error: any) {
            if (!aborted && handlers.onError) {
                handlers.onError(error);
            }
        } finally {
            if (!aborted && handlers.onClose) {
                handlers.onClose();
            }
        }
    }

    // 启动异步连接
    connect();

    return {
        close() {
            aborted = true;
            if (reader) {
                reader.cancel().catch(() => { });
            }
            if (handlers.onClose) {
                handlers.onClose();
            }
        },
    };
}
