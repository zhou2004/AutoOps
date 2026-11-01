export const backupMethods: Record<
  string,
  { label: string; description: string }
> = {
  logical: {
    label: "逻辑全量",
    description:
      "导出数据库结构与数据，适合首次备份和按备份时刻恢复；不支持指定时间点恢复。"
  },
  physical: {
    label: "物理全量",
    description:
      "备份数据库文件。MySQL 需要匹配版本的 XtraBackup；PostgreSQL 需要先配置 pgBackRest 与 WAL 归档。"
  },
  incremental: {
    label: "增量备份",
    description:
      "依赖已有物理基线及完整恢复链，系统会选择并记录依赖备份。首次使用请先创建物理全量。"
  },
  differential: {
    label: "差异备份",
    description: "PostgreSQL 备份最近全量之后的变化，依赖 pgBackRest 全量基线。"
  },
  "binlog-archive": {
    label: "全量基线 + binlog 归档",
    description:
      "MySQL 导出物理基线与已关闭 binlog。恢复时需选择归档覆盖范围内的明确目标时间。"
  },
  "wal-archive": {
    label: "全量基线 + WAL 归档",
    description:
      "PostgreSQL 导出基线与连续 WAL。仅在已核验归档覆盖的时间窗口内支持时间点恢复。"
  },
  rdb: {
    label: "RDB 快照",
    description:
      "备份 Redis 数据快照。Cluster 按主分片备份，恢复逐分片校验，不提供跨分片统一事务快照。"
  }
};
export const backupMethodLabel = (method?: string) =>
  backupMethods[method || ""]?.label || method || "—";
export function recoveryWindow(
  backup?: any
): { start: number; end: number } | undefined {
  const artifacts = Object.values(backup?.manifest?.artifacts || {}) as any[];
  if (
    backup?.status !== "complete" ||
    !artifacts.length ||
    artifacts.length !== backup?.manifest?.members?.length ||
    artifacts.some(
      a =>
        !a.archiveCheck ||
        !["mysql", "postgresql"].includes(a.engine) ||
        a.method === "logical" ||
        !a.backupStop ||
        !a.archiveThrough
    )
  )
    return;
  const start = Math.max(...artifacts.map(a => Number(a.backupStop) * 1000));
  const end = Math.min(...artifacts.map(a => Date.parse(a.archiveThrough)));
  return Number.isFinite(start) && Number.isFinite(end) && end >= start
    ? { start, end }
    : undefined;
}
