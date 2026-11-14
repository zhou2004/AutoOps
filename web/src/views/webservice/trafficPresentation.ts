import { locale, tr } from "@/locales";
import type { TrafficPoint } from "@/api/webTraffic";

export function trafficNumber(value: number | null | undefined, digits = 1) {
  return value == null || !Number.isFinite(value)
    ? "—"
    : value.toLocaleString(locale.value, { maximumFractionDigits: digits });
}
export function trafficBytes(value: number | null | undefined, rate = true) {
  if (value == null || !Number.isFinite(value)) return "—";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index++;
  }
  return `${trafficNumber(value)} ${units[index]}${rate ? "/s" : ""}`;
}
export function trafficTime(value?: string | null, brief = false) {
  if (!value) return tr("尚未更新");
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return tr("未知");
  return new Intl.DateTimeFormat(locale.value, {
    timeZone: "Asia/Shanghai",
    ...(brief ? {} : { month: "2-digit", day: "2-digit" }),
    hour: "2-digit",
    minute: "2-digit",
    ...(brief ? {} : { second: "2-digit" }),
    hour12: false
  }).format(date);
}
export function trafficChart(
  points: TrafficPoint[],
  metrics: { key: keyof Omit<TrafficPoint, "at">; name: string }[],
  unit = ""
) {
  return {
    color: ["#1677ff", "#14b8a6", "#f59e0b"],
    tooltip: { trigger: "axis", renderMode: "richText" },
    legend: { bottom: 0 },
    grid: { left: 12, right: 20, top: 32, bottom: 42, containLabel: true },
    xAxis: {
      type: "time",
      axisLabel: {
        formatter: (value: number) =>
          trafficTime(new Date(value).toISOString(), true)
      }
    },
    yAxis: { type: "value", name: unit, min: 0 },
    series: metrics.map(metric => ({
      name: tr(metric.name),
      type: "line",
      showSymbol: false,
      connectNulls: false,
      lineStyle: { width: 2 },
      data: points.map(point => [point.at, point[metric.key]])
    }))
  };
}
