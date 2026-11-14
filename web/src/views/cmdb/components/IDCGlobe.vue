<template>
  <div ref="host" class="idc-globe">
    <div class="globe-vignette" />
    <div class="globe-caption">
      <span>GLOBAL INFRASTRUCTURE</span>
      <h2>{{ tr("连接每一座数据中心") }}</h2>
      <p>{{ tr("旋转地球探索 IDC · 点击坐标进入机房") }}</p>
    </div>
    <div class="globe-pins">
      <button
        v-for="r in located"
        :key="r.id"
        :ref="el => (pinElements[r.id] = el)"
        class="globe-pin"
        :aria-label="tr('进入 {name}', { name: r.name })"
        @pointerenter="spinning = false"
        @focus="spinning = false"
        @click="enter(r)"
      >
        <i /><span
          >{{ r.city || r.shortName || r.name
          }}<small>{{ r.name }}</small></span
        >
      </button>
    </div>
    <div v-if="failure" class="globe-failure">
      {{ tr(failure) }}{{ tr("，请从数据中心目录进入。") }}
    </div>
    <div class="globe-toolbar">
      <button @click="spinning = !spinning">
        {{ spinning ? tr("暂停自转") : tr("开启自转") }}</button
      ><button @click="home">{{ tr("全球视角") }}</button
      ><small>{{ tr("地理底图 © Natural Earth · 经纬度定位") }}</small>
    </div>
  </div>
</template>
<script setup>
import { tr } from "@/locales";
import {
  ref,
  inject,
  computed,
  onMounted,
  onBeforeUnmount,
  onActivated,
  onDeactivated,
  watch
} from "vue";
import * as T from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import land from "@/assets/maps/land-110m.json";
import { dispose } from "../three/facilityModels";
const props = defineProps({ rooms: { type: Array, default: () => [] } }),
  emit = defineEmits(["enter"]);
const dark = inject("facilityDark", ref(false));
let earth;
const host = ref(),
  failure = ref(""),
  spinning = ref(true),
  pinElements = {};
const located = computed(() =>
  props.rooms.filter(
    r => Number.isFinite(r.latitude) && Number.isFinite(r.longitude)
  )
);
let renderer,
  scene,
  camera,
  controls,
  observer,
  root,
  markers,
  frame = 0,
  last = 0,
  active = true,
  flight,
  pendingRoom,
  rings = [];
const reduced = window.matchMedia("(prefers-reduced-motion: reduce)");
const point = (lat, lon, r = 2) =>
  new T.Vector3(
    Math.cos((lat * Math.PI) / 180) * Math.cos((lon * Math.PI) / 180) * r,
    Math.sin((lat * Math.PI) / 180) * r,
    -Math.cos((lat * Math.PI) / 180) * Math.sin((lon * Math.PI) / 180) * r
  );
function earthTexture() {
  const c = document.createElement("canvas");
  c.width = 2048;
  c.height = 1024;
  const ctx = c.getContext("2d");
  ctx.fillStyle = dark.value ? "#071d35" : "#397f9c";
  ctx.fillRect(0, 0, c.width, c.height);
  ctx.strokeStyle = dark.value ? "#103852" : "#519bb3";
  ctx.lineWidth = 1;
  for (let x = 0; x <= 2048; x += 2048 / 24) {
    ctx.beginPath();
    ctx.moveTo(x, 0);
    ctx.lineTo(x, 1024);
    ctx.stroke();
  }
  for (let y = 0; y <= 1024; y += 1024 / 12) {
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(2048, y);
    ctx.stroke();
  }
  for (const polygon of land) {
    ctx.beginPath();
    polygon.forEach(([lon, lat], i) => {
      const x = ((lon + 180) / 360) * 2048,
        y = ((90 - lat) / 180) * 1024;
      i ? ctx.lineTo(x, y) : ctx.moveTo(x, y);
    });
    ctx.closePath();
    ctx.fillStyle = dark.value ? "#23516a" : "#8fbac3";
    ctx.fill();
    ctx.strokeStyle = dark.value ? "#41a6b5" : "#b7dce2";
    ctx.lineWidth = 1.2;
    ctx.stroke();
  }
  const texture = new T.CanvasTexture(c);
  texture.colorSpace = T.SRGBColorSpace;
  texture.anisotropy = 4;
  return texture;
}
function buildMarkers() {
  if (!root) return;
  if (markers) {
    root.remove(markers);
    dispose(markers);
  }
  markers = new T.Group();
  root.add(markers);
  rings = [];
  for (const r of located.value) {
    const pos = point(r.latitude, r.longitude, 2.025),
      g = new T.Group();
    g.position.copy(pos);
    g.quaternion.setFromUnitVectors(
      new T.Vector3(0, 0, 1),
      pos.clone().normalize()
    );
    markers.add(g);
    const dot = new T.Mesh(
      new T.CircleGeometry(0.026, 20),
      new T.MeshBasicMaterial({ color: 0x9cffe2, side: T.DoubleSide })
    );
    g.add(dot);
    const ring = new T.Mesh(
      new T.RingGeometry(0.04, 0.047, 40),
      new T.MeshBasicMaterial({
        color: 0x47f6ce,
        transparent: true,
        side: T.DoubleSide,
        depthWrite: false
      })
    );
    g.add(ring);
    rings.push(ring);
    const stem = new T.BufferGeometry().setFromPoints([
      new T.Vector3(0, 0, 0),
      new T.Vector3(0, 0, 0.18)
    ]);
    g.add(new T.Line(stem, new T.LineBasicMaterial({ color: 0x72eccf })));
  }
}
function moveTo(pos, target = new T.Vector3(), room = null) {
  if (!controls) return;
  flight = {
    start: performance.now(),
    from: camera.position.clone(),
    to: pos,
    targetFrom: controls.target.clone(),
    target
  };
  pendingRoom = room;
  spinning.value = false;
}
function enter(r) {
  if (!camera || !controls) {
    emit("enter", r);
    return;
  }
  moveTo(
    point(r.latitude, r.longitude, 3.6),
    point(r.latitude, r.longitude, 0.6),
    r
  );
}
function home() {
  if (camera) moveTo(point(22, 105, 6.6 * Math.max(1, 1 / camera.aspect)));
}
function locate(r) {
  if (camera && Number.isFinite(r.latitude) && Number.isFinite(r.longitude))
    moveTo(
      point(r.latitude, r.longitude, 6.0 * Math.max(1, 1 / camera.aspect))
    );
}
function resize() {
  if (!renderer || !host.value) return;
  const w = host.value.clientWidth,
    h = host.value.clientHeight;
  if (!w || !h) return;
  const ratio = Math.max(1, h / w) / Math.max(1, 1 / camera.aspect);
  if (controls && Math.abs(ratio - 1) > 0.001) {
    camera.position
      .sub(controls.target)
      .multiplyScalar(ratio)
      .add(controls.target);
    if (flight) {
      flight.from
        .sub(flight.targetFrom)
        .multiplyScalar(ratio)
        .add(flight.targetFrom);
      flight.to.sub(flight.target).multiplyScalar(ratio).add(flight.target);
    }
  }
  renderer.setSize(w, h);
  camera.aspect = w / h;
  camera.updateProjectionMatrix();
}
function animate(t) {
  frame = 0;
  if (!active || document.hidden || failure.value) return;
  const dt = Math.min((t - last) / 1000, 0.05);
  last = t;
  if (flight) {
    const k = Math.min((t - flight.start) / (reduced.matches ? 1 : 850), 1),
      s = k * k * (3 - 2 * k);
    camera.position.lerpVectors(flight.from, flight.to, s);
    controls.target.lerpVectors(flight.targetFrom, flight.target, s);
    if (k === 1) {
      flight = null;
      if (pendingRoom) {
        const room = pendingRoom;
        pendingRoom = null;
        emit("enter", room);
      }
    }
  }
  controls.autoRotate = spinning.value && !reduced.matches && !flight;
  controls.update(dt);
  rings.forEach((r, i) => {
    const k = reduced.matches ? 0 : (t / 1900 + i * 0.17) % 1;
    r.scale.setScalar(1 + k * 2.6);
    r.material.opacity = 0.8 * (1 - k);
  });
  for (const r of located.value) {
    const el = pinElements[r.id];
    if (!el) continue;
    const p = point(r.latitude, r.longitude, 2.2);
    const facing = camera.position.clone().sub(p).dot(p) > 0;
    const projected = p.clone().project(camera);
    el.style.display = facing && projected.z < 1 ? "flex" : "none";
    el.style.left = `${(projected.x * 0.5 + 0.5) * host.value.clientWidth}px`;
    el.style.top = `${(-projected.y * 0.5 + 0.5) * host.value.clientHeight}px`;
  }
  renderer.render(scene, camera);
  frame = requestAnimationFrame(animate);
}
function start() {
  if (
    renderer &&
    controls &&
    !failure.value &&
    !frame &&
    active &&
    !document.hidden
  ) {
    last = performance.now();
    frame = requestAnimationFrame(animate);
  }
}
function visibility() {
  if (document.hidden) {
    cancelAnimationFrame(frame);
    frame = 0;
  } else start();
}
function lost(e) {
  e.preventDefault();
  failure.value = "3D 图形上下文已中断";
  cancelAnimationFrame(frame);
  frame = 0;
}
onMounted(() => {
  try {
    scene = new T.Scene();
    camera = new T.PerspectiveCamera(42, 1, 0.1, 100);
    camera.position.copy(point(22, 105, 6.6));
    renderer = new T.WebGLRenderer({ alpha: true, antialias: true });
    renderer.setPixelRatio(Math.min(devicePixelRatio, 1.75));
    renderer.domElement.addEventListener("webglcontextlost", lost);
    host.value.prepend(renderer.domElement);
    scene.add(new T.AmbientLight(0xb4ddff, 2));
    const light = new T.DirectionalLight(0xc8eeff, 2);
    light.position.set(-4, 5, -3);
    scene.add(light);
    root = new T.Group();
    scene.add(root);
    earth = new T.Mesh(
      new T.SphereGeometry(2, 96, 64),
      new T.MeshStandardMaterial({
        map: earthTexture(),
        metalness: 0.2,
        roughness: 0.8
      })
    );
    root.add(earth);
    const atmosphere = new T.Mesh(
      new T.SphereGeometry(2.075, 64, 48),
      new T.ShaderMaterial({
        transparent: true,
        side: T.BackSide,
        blending: T.AdditiveBlending,
        depthWrite: false,
        vertexShader:
          "varying vec3 n; varying vec3 v; void main(){ vec4 mv=modelViewMatrix*vec4(position,1.0);n=normalize(normalMatrix*normal);v=normalize(-mv.xyz);gl_Position=projectionMatrix*mv;}",
        fragmentShader:
          "varying vec3 n; varying vec3 v; void main(){float f=pow(1.0-abs(dot(normalize(n),normalize(v))),3.0);gl_FragColor=vec4(0.12,0.55,0.95,f*0.8);}"
      })
    );
    root.add(atmosphere);
    const stars = [];
    for (let i = 0; i < 850; i++) {
      const a = i * 2.39996,
        y = 1 - (2 * (i + 0.5)) / 850,
        r = Math.sqrt(1 - y * y);
      stars.push(Math.cos(a) * r * 23, y * 23, Math.sin(a) * r * 23);
    }
    const starGeo = new T.BufferGeometry();
    starGeo.setAttribute("position", new T.Float32BufferAttribute(stars, 3));
    scene.add(
      new T.Points(
        starGeo,
        new T.PointsMaterial({
          color: 0x5e8fb1,
          size: 0.025,
          transparent: true,
          opacity: 0.65
        })
      )
    );
    controls = new OrbitControls(camera, renderer.domElement);
    controls.enablePan = false;
    controls.enableDamping = true;
    controls.minDistance = 3;
    controls.maxDistance = 25;
    controls.autoRotateSpeed = 0.18;
    controls.addEventListener("start", () => {
      flight = null;
      pendingRoom = null;
      spinning.value = false;
    });
    observer = new ResizeObserver(resize);
    observer.observe(host.value);
    buildMarkers();
    resize();
    start();
    document.addEventListener("visibilitychange", visibility);
  } catch (e) {
    failure.value = "当前浏览器无法启用三维地球";
  }
});
watch(() => props.rooms, buildMarkers);
watch(dark, () => {
  if (!earth) return;
  earth.material.map.dispose();
  earth.material.map = earthTexture();
  earth.material.needsUpdate = true;
});
onActivated(() => {
  active = true;
  start();
});
onDeactivated(() => {
  active = false;
  cancelAnimationFrame(frame);
  frame = 0;
});
onBeforeUnmount(() => {
  active = false;
  cancelAnimationFrame(frame);
  document.removeEventListener("visibilitychange", visibility);
  observer?.disconnect();
  controls?.dispose();
  dispose(scene);
  if (renderer) {
    renderer.domElement.removeEventListener("webglcontextlost", lost);
    renderer.dispose();
    renderer.forceContextLoss();
    renderer.domElement.remove();
  }
});
defineExpose({ locate });
</script>
<style scoped>
.idc-globe {
  position: relative;
  height: 100%;
  min-height: 0;
  min-width: 0;
  background: radial-gradient(
    ellipse at 55% 45%,
    var(--facility-scene-center),
    var(--facility-scene-edge)
  );
  overflow: hidden;
}
.idc-globe :deep(canvas) {
  display: block;
  position: absolute;
  inset: 0;
  touch-action: none;
}
.globe-vignette {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 1;
}
.globe-caption {
  position: absolute;
  top: 22px;
  left: 24px;
  pointer-events: none;
  color: var(--facility-muted);
  z-index: 2;
}
.globe-caption > span {
  font: 9px monospace;
  letter-spacing: 3px;
  color: var(--facility-accent);
}
.globe-caption h2 {
  font-size: 20px;
  font-weight: 500;
  color: var(--facility-text);
  margin: 9px 0;
}
.globe-caption p {
  font-size: 12px;
}
.globe-pins {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 2;
}
.globe-pin {
  position: absolute;
  display: flex;
  align-items: center;
  gap: 8px;
  border: 0;
  background: transparent;
  color: var(--facility-text);
  cursor: pointer;
  pointer-events: auto;
  transform: translate(-6px, -50%);
  text-align: left;
  padding: 0;
  filter: drop-shadow(0 2px 5px #00111f33);
}
.globe-pin i {
  width: 9px;
  height: 9px;
  background: #54ddb6;
  border: 2px solid #eefff9;
  border-radius: 50%;
  box-shadow: 0 0 10px #35e8b455;
}
.globe-pin > span {
  font-size: 12px;
  background: color-mix(in srgb, var(--facility-panel) 92%, transparent);
  border: 1px solid var(--facility-line);
  border-radius: 4px;
  padding: 5px 9px;
  white-space: nowrap;
}
.globe-pin small {
  display: block;
  font-size: 9px;
  color: var(--facility-muted);
  margin-top: 3px;
}
.globe-toolbar {
  position: absolute;
  bottom: 16px;
  left: 24px;
  right: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 3;
}
.globe-toolbar button {
  border: 1px solid var(--facility-line);
  background: var(--facility-panel);
  color: var(--facility-text);
  border-radius: 5px;
  padding: 7px 10px;
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
}
.globe-toolbar small {
  margin-left: auto;
  font-size: 10px;
  color: var(--facility-muted);
}
.globe-failure {
  position: absolute;
  top: 45%;
  padding: 16px;
  width: 100%;
  text-align: center;
  color: var(--facility-warning);
  z-index: 4;
}
@media (max-width: 750px), (max-height: 650px) {
  .globe-caption {
    top: 16px;
    left: 16px;
  }
  .globe-caption h2 {
    font-size: 16px;
  }
  .globe-caption p,
  .globe-caption > span {
    display: none;
  }
  .globe-toolbar {
    left: 12px;
    bottom: 12px;
  }
  .globe-toolbar small {
    display: none;
  }
  .globe-pin small {
    display: none;
  }
}
</style>
