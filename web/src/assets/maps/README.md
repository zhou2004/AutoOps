# Globe land silhouette

`land-110m.json` contains only rounded longitude/latitude polygon coordinates from Natural Earth's 1:110m land dataset (`ne_110m_land.geojson`). No country boundary or jurisdiction layer is included. Coordinates were rounded to three decimal places to reduce the bundled asset size.

- Source: https://github.com/nvkelso/natural-earth-vector/blob/master/geojson/ne_110m_land.geojson
- License: public domain, https://www.naturalearthdata.com/about/terms-of-use/
- Retrieved: 2026-11-14

Rendered locally by `views/cmdb/components/IDCGlobe.vue`. No map provider API, tiles, access token or external runtime request is required.
