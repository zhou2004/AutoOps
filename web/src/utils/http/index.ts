import Axios, {
  type AxiosInstance,
  type AxiosRequestConfig,
  type CustomParamsSerializer
} from "axios";
import type {
  PureHttpError,
  RequestMethods,
  PureHttpResponse,
  PureHttpRequestConfig
} from "./types.d";
import { stringify } from "qs";
import { getToken, formatToken, removeToken } from "@/utils/auth";
import { ElMessage } from "element-plus";
import router from "@/router";

// AutoOps API 基础路径（从环境变量读取）
const API_HOST = import.meta.env.VITE_API_BASE_URL || "";           // 例如 http://172.22.107.76:8000
const API_PREFIX = "/api/v1";                                       // Go 后端统一前缀

// 是否正在跳转登录页的标志
let isRedirectingToLogin = false;

// 相关配置
const defaultConfig: AxiosRequestConfig = {
  timeout: 15000,
  headers: {
    Accept: "application/json, text/plain, */*",
    "Content-Type": "application/json"
  },
  paramsSerializer: {
    serialize: stringify as unknown as CustomParamsSerializer
  }
};

class PureHttp {
  constructor() {
    this.httpInterceptorsRequest();
    this.httpInterceptorsResponse();
  }

  private static initConfig: PureHttpRequestConfig = {};
  private static axiosInstance: AxiosInstance = Axios.create(defaultConfig);

  /** 请求拦截 */
  private httpInterceptorsRequest(): void {
    PureHttp.axiosInstance.interceptors.request.use(
      async (config: PureHttpRequestConfig): Promise<any> => {
        if (typeof config.beforeRequestCallback === "function") {
          config.beforeRequestCallback(config);
          return config;
        }
        if (PureHttp.initConfig.beforeRequestCallback) {
          PureHttp.initConfig.beforeRequestCallback(config);
          return config;
        }
        // 拼接 API 基础路径（与旧项目 request.js 逻辑一致）
        // 1. 确保 /api/v1 前缀
        // 2. 如果 API_HOST 是绝对路径，设置 baseURL
        if (config.url && !config.url.startsWith("http")) {
          // 添加 /api/v1 前缀（如果 URL 还没有）
          if (!config.url.startsWith(API_PREFIX)) {
            config.url = API_PREFIX + (config.url.startsWith("/") ? config.url : "/" + config.url);
          }
          // 设置 baseURL（axios 自动拼接主机名）
          if (API_HOST) {
            config.baseURL = API_HOST.replace(/\/$/, "");
          }
        }
        // 白名单接口不需要 token
        const whiteList = ["/login", "/captcha"];
        const isWhiteListed = whiteList.some(url => config.url?.endsWith(url));
        if (!isWhiteListed) {
          const token = getToken();
          if (token) {
            config.headers["Authorization"] = formatToken(token);
          }
        }
        return config;
      },
      error => Promise.reject(error)
    );
  }

  /** 响应拦截 */
  private httpInterceptorsResponse(): void {
    const instance = PureHttp.axiosInstance;
    instance.interceptors.response.use(
      (response: PureHttpResponse) => {
        const $config = response.config;
        if (typeof $config.beforeResponseCallback === "function") {
          $config.beforeResponseCallback(response);
          return response.data;
        }
        if (PureHttp.initConfig.beforeResponseCallback) {
          PureHttp.initConfig.beforeResponseCallback(response);
          return response.data;
        }
        // 检查业务状态码
        const resData = response.data;
        if (resData && typeof resData === 'object') {
          const { code, message } = resData;
          if (code === 401 || code === 406) {
            if (!isRedirectingToLogin) {
              isRedirectingToLogin = true;
              removeToken();
              ElMessage({ message: message || 'Token已过期，请重新登录', type: 'warning', duration: 2000 });
              router.push("/login");
              setTimeout(() => { isRedirectingToLogin = false; }, 1000);
            }
            return new Promise(() => { });
          }
        }
        return response.data;
      },
      (error: PureHttpError) => {
        if (error.response) {
          const status = error.response.status;
          if (status === 401) {
            if (!isRedirectingToLogin) {
              isRedirectingToLogin = true;
              removeToken();
              ElMessage({ message: '登录已过期，请重新登录', type: 'warning', duration: 2000 });
              router.push("/login");
              setTimeout(() => { isRedirectingToLogin = false; }, 1000);
            }
            return new Promise(() => { });
          }
          if (!isRedirectingToLogin) {
            const msg = error.response.data || `请求失败(${status})`;
            ElMessage.error(msg);
          }
        } else if (!isRedirectingToLogin) {
          ElMessage.error(error.message || '网络连接失败');
        }
        return Promise.reject(error);
      }
    );
  }

  public request<T>(
    method: RequestMethods,
    url: string,
    param?: AxiosRequestConfig,
    axiosConfig?: PureHttpRequestConfig
  ): Promise<T> {
    const config = { method, url, ...param, ...axiosConfig } as PureHttpRequestConfig;
    return new Promise((resolve, reject) => {
      PureHttp.axiosInstance
        .request(config)
        .then((response: undefined) => resolve(response))
        .catch(error => reject(error));
    });
  }

  public post<T, P>(url: string, params?: AxiosRequestConfig<P>, config?: PureHttpRequestConfig): Promise<T> {
    return this.request<T>("post", url, params, config);
  }

  public get<T, P>(url: string, params?: AxiosRequestConfig<P>, config?: PureHttpRequestConfig): Promise<T> {
    return this.request<T>("get", url, params, config);
  }
}

export const http = new PureHttp();
