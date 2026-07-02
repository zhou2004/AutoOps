/**
 * AutoOps API 兼容层
 * 适配旧版 request 调用方式到新的 http 模块
 * 
 * 关键：将 http 响应包装为 { data: response } 格式，
 * 以匹配旧 project 中 axios response 的 { data: GoJSONBody } 结构
 */
import { http } from "@/utils/http";
import type { RequestMethods } from "@/utils/http/types.d";

interface RequestOptions {
    url: string;
    method: string;
    data?: any;
    params?: any;
    headers?: Record<string, string>;
}

export function request(options: RequestOptions) {
    const { url, method, data, params, headers } = options;
    const config: any = {};
    if (data !== undefined) config.data = data;
    if (params !== undefined) config.params = params;
    if (headers) config.headers = headers;

    // 包装响应为 { data: GoJSONBody } 以兼容旧组件
    // 旧组件模式：const { data: res } = await this.$api.xxx() → res = Go JSON body → res.code / res.data
    return http.request(method.toLowerCase() as RequestMethods, url, config)
        .then((response: any) => ({ data: response }));
}

export default request;
