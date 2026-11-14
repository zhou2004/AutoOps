/** Accept an origin, a gateway prefix, or an already versioned API base. */
export function apiV1Base(configured = "") {
  const base = configured.trim().replace(/\/+$/, "");
  return base.endsWith("/api/v1") ? base : `${base}/api/v1`;
}
