import { tr } from "@/locales";

const messages: Record<string, string> = {
 "Log scan was incomplete; its cursor will be initialized again to avoid reporting backlog as current traffic": "日志扫描未完成，将重建基线，避免把积压日志误报为当前流量",
 "Failed log cursor cleared; recovery starts a new sampling baseline": "失败日志的游标已清除，恢复后会建立新的采样基线",
  "Access-log ownership check failed": "访问日志归属检查失败",
  "container identity changed during traffic collection": "流量采集期间容器身份发生变化",
  "Dedicated traffic logs selected; byte and route metrics cover only sites with traffic logging enabled, ordinary access logs are excluded to prevent duplicate counting": "已优先使用专用流量日志；字节和流向统计仅覆盖已启用流量日志的站点，普通访问日志不重复计数",
  "Only the first 8 registered access logs were scanned": "本次最多扫描 8 个已登记访问日志",
  "Access-log scan reached 4 MiB limit": "本次访问日志扫描达到 4 MiB 上限",
  "Some access-log records use an unsupported format": "部分访问日志格式无法解析，已从统计中排除",
  "Log cursor initialized; only new requests are counted": "日志采集基线已建立，仅统计之后的新请求",
  "No registered access-log file; container stdout and custom formats require a dedicated supported access log": "没有可采集的已登记访问日志；容器输出或自定义格式需配置受支持的专用访问日志",
  "Shared access log excluded: requests cannot be attributed to one Nginx instance": "已排除多个实例共用的日志，避免将无法归属的请求重复计数",
  "Collection gap exceeded 90 seconds; log cursors were reset instead of attributing old requests to the current interval": "采集中断超过 90 秒，已重建日志基线，避免将历史请求计入当前时段",
  "Access-log collection failed: ": "访问日志采集失败：",
  "Request counter baseline initialized or reset; QPS resumes after the next sample": "请求计数基线已建立或重置，下一次采样后恢复 QPS",
  "Instance discovery failed; traffic metrics are unavailable": "实例发现失败，流量指标暂不可用",
  "stub_status endpoint is not configured": "尚未配置 stub_status 地址",
  "stub_status collection failed": "stub_status 采集失败",
  "History reached the 100000-sample or 32 MiB query limit; older data in this range is omitted": "历史查询达到 100000 个采样或 32 MiB 上限，该范围内更早的数据未计入",
  "Access-log metrics require log permission": "查看日志派生指标需要日志权限",
  "No traffic sample in the last 90 seconds": "最近 90 秒没有有效流量采样",
  "Waiting for the first traffic sample": "等待首次流量采样",
  "Top-list aggregation reached the 5000-key limit": "排行统计达到 5000 个不同值的上限",
  "Missing, unparsed, shared or bounded log data is excluded; figures represent observed traffic only": "统计仅代表已采集的流量；缺失、无法解析、共享或超出扫描上限的日志不计入",
  "Log-derived metrics are limited to instances with log permission": "日志派生指标仅包含具有日志权限的实例",
  "QPS is derived from Nginx counter deltas; byte rates represent logged HTTP bytes, not NIC bandwidth. 5xx responses define the error rate. Log statistics use collection intervals; bounded backlog is disclosed as partial.": "QPS 来自 Nginx 请求计数差值；字节速率为日志记录的 HTTP 字节量，错误率统计 5xx 响应。日志按采集时段汇总，采集不完整时会标明部分覆盖。",
  "log path left its registered scope": "日志路径已离开登记范围",
  "log is not a regular file": "日志目标不是普通文件",
  "log rotated during sampling; retry on next collection": "采样期间日志已轮转，将在下次采集重试"
};

/** Translate stable adapter messages while preserving remote diagnostics. */
export function trafficMessage(value: string): string {
  let result = value || "";
  for (const [english, chinese] of Object.entries(messages)) {
    result = result.split(english).join(tr(chinese));
  }
  return result;
}
