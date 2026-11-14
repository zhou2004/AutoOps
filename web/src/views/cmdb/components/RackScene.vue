<template>
  <div ref="host" class="rack-scene" :aria-label="tr('数据中心三维场景')">
    <div class="scene-caption">
      <span>{{
        focused ? "RACK INSPECTION" : "DATA CENTER / DIGITAL VIEW"
      }}</span
      ><strong>{{ focused ? selectedRack?.name : roomName }}</strong
      ><small>{{
        focused
          ? tr("{door} · 点击设备抽出查看", {
              door: openDoor ? tr("机柜门打开") : tr("机柜门关闭")
            })
          : tr("机柜与设备按台账 U 位生成")
      }}</small>
    </div>
    <div class="scene-tools">
      <button @click="resetView">{{ tr("机房全景") }}</button
      ><button @click="topView">{{ tr("俯视") }}</button
      ><button :class="{ active: tour }" @click="toggleTour">
        {{ tour ? tr("停止巡游") : tr("自动巡游") }}</button
      ><button
        :class="{ active: capacityMode }"
        @click="capacityMode = !capacityMode"
      >
        {{ tr("容量视图") }}</button
      ><button :class="{ active: airflow }" @click="airflow = !airflow">
        {{ tr("气流示意") }}</button
      ><button @click="exportImage">{{ tr("截图") }}</button>
    </div>
    <div v-if="focused" class="inspection-tools">
      <button @click="openDoor = !openDoor">
        {{ openDoor ? tr("关闭柜门") : tr("打开柜门") }}</button
      ><button v-if="extracted" @click="resetDevice">
        {{ tr("收回设备") }}</button
      ><button
        @click="
          focused = false;
          resetView();
        "
      >
        {{ tr("返回机房") }}
      </button>
    </div>
    <div
      v-if="hover"
      class="scene-tooltip"
      :style="{ left: `${hover.x}px`, top: `${hover.y}px` }"
    >
      {{ hover.text }}
    </div>
    <div v-if="failure" class="scene-failure">
      <el-alert :title="failure" type="warning" :closable="false" />
      <p>{{ tr("机柜目录与 U 位列表仍然可用。") }}</p>
    </div>
    <div class="scene-bottom">
      <span><i />{{ focused ? tr("设备巡检视角") : tr("交互式三维场景") }}</span
      ><small>{{
        airflow
          ? tr("气流为视觉示意，未接入传感器")
          : tr("状态灯按台账显示；闪烁为视觉效果")
      }}</small
      ><span>{{ tr("拖动旋转 · 滚轮缩放 · 点击下钻") }}</span>
    </div>
  </div>
</template>
<script setup>
import { tr } from "@/locales";
import {
  ref,
  inject,
  computed,
  watch,
  onMounted,
  onBeforeUnmount,
  onActivated,
  onDeactivated
} from "vue";
import * as T from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import { RoomEnvironment } from "three/addons/environments/RoomEnvironment.js";
import { makeRack, makeRoom, dispose, U } from "../three/facilityModels";
const props = defineProps({
    racks: { type: Array, default: () => [] },
    selected: Number,
    roomName: String
  }),
  emit = defineEmits(["select"]);
const dark = inject("facilityDark", ref(false));
const host = ref(),
  failure = ref(""),
  focused = ref(false),
  openDoor = ref(true),
  tour = ref(false),
  capacityMode = ref(false),
  airflow = ref(false),
  hover = ref(null),
  extracted = ref(null);
let renderer,
  scene,
  camera,
  controls,
  observer,
  group,
  roomGroup,
  environment,
  frame = 0,
  last = 0,
  lastRender = 0,
  active = true,
  flight,
  down,
  models = [],
  bounds = new T.Box3(),
  airPoints,
  airPositions;
const selectedRack = computed(() =>
  props.racks.find(r => r.id === props.selected)
);
const reduced = window.matchMedia("(prefers-reduced-motion: reduce)");
function build() {
  if (!renderer) return;
  flight = null;
  extracted.value = null;
  focused.value = false;
  if (group) {
    scene.remove(group);
    dispose(group);
  }
  group = new T.Group();
  scene.add(group);
  const textureCache = new Map();
  models = props.racks.map((r, i) => makeRack(group, r, i, textureCache));
  bounds.setFromObject(group);
  if (bounds.isEmpty())
    bounds.set(new T.Vector3(-2, 0, -2), new T.Vector3(2, 2, 2));
  roomGroup = new T.Group();
  group.add(roomGroup);
  const room = makeRoom(roomGroup, bounds);
  airPositions = new Float32Array(180 * 3);
  const geo = new T.BufferGeometry();
  geo.setAttribute("position", new T.BufferAttribute(airPositions, 3));
  airPoints = new T.Points(
    geo,
    new T.PointsMaterial({
      color: 0x65e5ff,
      size: 0.032,
      transparent: true,
      opacity: 0.65,
      depthWrite: false,
      blending: T.AdditiveBlending
    })
  );
  airPoints.userData.room = room;
  roomGroup.add(airPoints);
  resetView();
}
function transition(position, target) {
  tour.value = false;
  flight = {
    time: performance.now(),
    from: camera.position.clone(),
    to: position,
    fromTarget: controls.target.clone(),
    target
  };
}
function resetView() {
  if (!camera || !controls) return;
  focused.value = false;
  resetDevice();
  const c = bounds.getCenter(new T.Vector3()),
    s = bounds.getSize(new T.Vector3()),
    span = Math.max(s.x, s.z, 4) * Math.max(1, 1 / camera.aspect);
  transition(
    new T.Vector3(c.x + span * 1.06, span * 0.96, c.z + span * 1.46),
    new T.Vector3(c.x, 0.65, c.z)
  );
}
function topView() {
  if (!camera || !controls) return;
  focused.value = false;
  resetDevice();
  const c = bounds.getCenter(new T.Vector3()),
    s = bounds.getSize(new T.Vector3());
  transition(
    new T.Vector3(
      c.x,
      Math.max(s.x, s.z, 4) * 1.65 * Math.max(1, 1 / camera.aspect),
      c.z + 0.01
    ),
    new T.Vector3(c.x, 0, c.z)
  );
}
function focus(id) {
  const rack = models.find(r => r.record.id === id);
  if (!rack) return;
  focused.value = true;
  openDoor.value = true;
  resetDevice();
  // Fit the full-height cabinet and open door, including narrow viewports.
  const visibleHeight = Math.max(
    rack.h * 1.3,
    (rack.w * 1.9) / Math.max(camera.aspect, 0.4)
  );
  const distance =
    visibleHeight / (2 * Math.tan(T.MathUtils.degToRad(camera.fov / 2))) +
    rack.d / 2;
  const p = rack.root.position,
    offset = new T.Vector3(0.6, 0.2, distance).applyAxisAngle(
      new T.Vector3(0, 1, 0),
      rack.root.rotation.y
    );
  transition(
    new T.Vector3(p.x + offset.x, rack.h * 0.55 + offset.y, p.z + offset.z),
    new T.Vector3(p.x, rack.h * 0.5, p.z)
  );
}
function inspectDevice(device) {
  if (device.restricted) return;
  const rack = models.find(r => r.record.id === device.cabinetId);
  if (!rack) return;
  focus(rack.record.id);
  extracted.value = {
    rack: device.cabinetId,
    id: device.id,
    kind: device.kind
  };
  const item = rack.devices.find(
    d => d.device.id === device.id && d.device.kind === device.kind
  );
  if (!item) return;
  const target = rack.root.localToWorld(
    new T.Vector3(0, item.body.position.y, rack.d / 2 + 0.18)
  );
  const pos = target
    .clone()
    .add(
      new T.Vector3(0.48, 0.32, 1.22)
        .multiplyScalar(Math.max(1, 1 / camera.aspect))
        .applyAxisAngle(new T.Vector3(0, 1, 0), rack.root.rotation.y)
    );
  transition(pos, target);
}
function resetDevice() {
  extracted.value = null;
}
function toggleTour() {
  focused.value = false;
  resetDevice();
  flight = null;
  tour.value = !tour.value;
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
        .sub(flight.fromTarget)
        .multiplyScalar(ratio)
        .add(flight.fromTarget);
      flight.to.sub(flight.target).multiplyScalar(ratio).add(flight.target);
    }
  }
  camera.aspect = w / h;
  camera.updateProjectionMatrix();
  renderer.setSize(w, h);
}
function animate(t) {
  frame = 0;
  if (!active || document.hidden || failure.value) return;
  const dt = Math.min((t - last) / 1000, 0.05);
  last = t;
  if (flight) {
    const k = Math.min((t - flight.time) / (reduced.matches ? 1 : 900), 1),
      s = k * k * (3 - 2 * k);
    camera.position.lerpVectors(flight.from, flight.to, s);
    controls.target.lerpVectors(flight.fromTarget, flight.target, s);
    if (k === 1) flight = null;
  }
  controls.autoRotate = tour.value && !reduced.matches && !flight;
  controls.update(dt);
  const ease = reduced.matches ? 1 : 1 - Math.exp(-dt * 7);
  for (const rack of models) {
    const selected = rack.record.id === props.selected;
    rack.root.visible = !focused.value || selected;
    const rotation =
      selected && focused.value && openDoor.value ? -Math.PI * 0.64 : 0;
    rack.pivot.rotation.y += (rotation - rack.pivot.rotation.y) * ease;
    rack.glowMat.color.setHex(selected ? 0x6be4ff : 0x247588);
    rack.strip.material.opacity = selected ? 1 : 0.5;
    rack.capacity.visible = capacityMode.value;
    const capHeight = Math.max(0.025, rack.record.usedUnit * U);
    rack.capacity.scale.y = capHeight / 0.02;
    rack.capacity.position.y = 0.13 + capHeight / 2;
    for (const item of rack.devices) {
      const selectedDevice =
        extracted.value &&
        selected &&
        extracted.value.id === item.device.id &&
        extracted.value.kind === item.device.kind;
      const z = item.baseZ + (selectedDevice ? 0.62 : 0);
      item.body.position.z += (z - item.body.position.z) * ease;
      if (!reduced.matches && item.device.status === 1)
        item.ledMat.color.setHSL(
          0.43,
          0.8,
          0.45 + 0.25 * Math.sin(t * 0.002 + item.device.id)
        );
    }
  }
  if (roomGroup) {
    roomGroup.visible = !focused.value;
    airPoints.visible = airflow.value;
    if (airflow.value) {
      const r = airPoints.userData.room;
      for (let i = 0; i < 180; i++) {
        airPositions[i * 3] =
          r.center.x - r.width * 0.38 + ((i % 30) / 30) * r.width * 0.76;
        airPositions[i * 3 + 1] =
          0.12 +
          (((reduced.matches ? 0 : t * 0.0003) + Math.floor(i / 30) * 0.16) %
            1) *
            1.85;
        airPositions[i * 3 + 2] =
          r.center.z + Math.sin(i * 1.7) * r.depth * 0.35;
      }
      airPoints.geometry.attributes.position.needsUpdate = true;
    }
  }
  if (t - lastRender > 32 || flight) {
    renderer.render(scene, camera);
    lastRender = t;
  }
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
function hit(e) {
  if (!renderer) return null;
  const rect = renderer.domElement.getBoundingClientRect(),
    ray = new T.Raycaster();
  ray.setFromCamera(
    new T.Vector2(
      ((e.clientX - rect.left) / rect.width) * 2 - 1,
      (-(e.clientY - rect.top) / rect.height) * 2 + 1
    ),
    camera
  );
  const pickable = [];
  for (const rack of models) {
    if (!rack.root.visible) continue;
    rack.root.traverse(o => {
      if (o.userData.rackId) pickable.push(o);
    });
  }
  return ray.intersectObjects(pickable, false)[0]?.object.userData;
}
function pointerDown(e) {
  down = { x: e.clientX, y: e.clientY };
}
function pointerUp(e) {
  if (!down || Math.hypot(e.clientX - down.x, e.clientY - down.y) > 5) return;
  const found = hit(e);
  if (found) {
    emit("select", found);
    if (found.device) {
      inspectDevice(found.device);
    } else focus(found.rackId);
  }
}
let hoverAt = 0;
function pointerMove(e) {
  if (performance.now() - hoverAt < 90) return;
  hoverAt = performance.now();
  const found = hit(e);
  if (!found) {
    hover.value = null;
    return;
  }
  const rect = host.value.getBoundingClientRect();
  hover.value = {
    text: found.device
      ? found.device.name || tr("受限资产")
      : props.racks.find(r => r.id === found.rackId)?.name,
    x: Math.min(e.clientX - rect.left + 12, rect.width - 160),
    y: e.clientY - rect.top - 30
  };
}
function clearHover() {
  hover.value = null;
}
function lost(e) {
  e.preventDefault();
  failure.value = "3D 图形上下文已中断，请刷新恢复";
  cancelAnimationFrame(frame);
  frame = 0;
}
function exportImage() {
  if (!renderer) return;
  renderer.render(scene, camera);
  const a = document.createElement("a");
  a.download = `${props.roomName || tr("机房")}-3D.png`;
  a.href = renderer.domElement.toDataURL("image/png");
  a.click();
}
onMounted(() => {
  try {
    scene = new T.Scene();
    scene.fog = new T.FogExp2(dark.value ? 0x071522 : 0xe5eef6, 0.012);
    camera = new T.PerspectiveCamera(38, 1, 0.01, 2000);
    camera.position.set(7, 7, 11);
    renderer = new T.WebGLRenderer({
      alpha: true,
      antialias: true,
      preserveDrawingBuffer: true
    });
    renderer.setPixelRatio(Math.min(devicePixelRatio, 1.75));
    renderer.toneMapping = T.ACESFilmicToneMapping;
    renderer.toneMappingExposure = dark.value ? 1.3 : 1.1;
    renderer.shadowMap.enabled = true;
    renderer.shadowMap.type = T.PCFSoftShadowMap;
    host.value.prepend(renderer.domElement);
    const pmrem = new T.PMREMGenerator(renderer),
      env = new RoomEnvironment();
    environment = pmrem.fromScene(env, 0.04);
    scene.environment = environment.texture;
    dispose(env);
    pmrem.dispose();
    scene.add(new T.HemisphereLight(0xd9f3ff, 0x254257, 2));
    const sun = new T.DirectionalLight(0xd6ecff, 3);
    sun.position.set(4, 9, 6);
    sun.castShadow = true;
    sun.shadow.mapSize.set(2048, 2048);
    Object.assign(sun.shadow.camera, {
      left: -12,
      right: 12,
      top: 12,
      bottom: -12,
      near: 1,
      far: 35
    });
    sun.shadow.bias = -0.0004;
    scene.add(sun);
    const fill = new T.DirectionalLight(0x49b9f5, 1.2);
    fill.position.set(-4, 3, -4);
    scene.add(fill);
    controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.minDistance = 0.25;
    controls.maxDistance = 1500;
    controls.maxPolarAngle = Math.PI / 2 - 0.025;
    controls.autoRotateSpeed = 0.45;
    controls.addEventListener("start", () => {
      flight = null;
      tour.value = false;
    });
    renderer.domElement.addEventListener("pointerdown", pointerDown);
    renderer.domElement.addEventListener("pointerup", pointerUp);
    renderer.domElement.addEventListener("pointermove", pointerMove);
    renderer.domElement.addEventListener("pointerleave", clearHover);
    renderer.domElement.addEventListener("webglcontextlost", lost);
    observer = new ResizeObserver(resize);
    observer.observe(host.value);
    build();
    resize();
    start();
    document.addEventListener("visibilitychange", visibility);
  } catch (e) {
    failure.value = "当前浏览器无法启用 WebGL 3D 渲染";
  }
});
watch(() => props.racks, build);
watch(dark, value => {
  scene?.fog?.color.setHex(value ? 0x071522 : 0xe5eef6);
  if (renderer) renderer.toneMappingExposure = value ? 1.3 : 1.1;
});
onActivated(() => {
  active = true;
  resize();
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
  environment?.dispose();
  if (renderer) {
    for (const [name, fn] of [
      ["pointerdown", pointerDown],
      ["pointerup", pointerUp],
      ["pointermove", pointerMove],
      ["pointerleave", clearHover],
      ["webglcontextlost", lost]
    ])
      renderer.domElement.removeEventListener(name, fn);
    renderer.dispose();
    renderer.forceContextLoss();
    renderer.domElement.remove();
  }
});
defineExpose({ focus, resetView, inspectDevice, resetDevice });
</script>
<style scoped>
.rack-scene {
  position: relative;
  height: 100%;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  background: radial-gradient(
    ellipse at 50% 45%,
    var(--facility-scene-center),
    var(--facility-scene-edge)
  );
}
.rack-scene :deep(canvas) {
  display: block;
  width: 100%;
  height: 100%;
  touch-action: none;
  position: absolute;
  inset: 0;
}
.scene-caption {
  background: color-mix(in srgb, var(--facility-panel) 84%, transparent);
  padding: 7px 10px;
  border-radius: 5px;
  max-width: calc(100% - 28px);
  position: absolute;
  left: 18px;
  top: 18px;
  pointer-events: none;
  z-index: 2;
}
.scene-caption > span {
  display: block;
  color: var(--facility-accent);
  font: 9px monospace;
  letter-spacing: 2px;
}
.scene-caption strong {
  display: block;
  color: var(--facility-text);
  font-size: 18px;
  font-weight: 500;
  margin: 6px 0;
}
.scene-caption small {
  color: var(--facility-muted);
  font-size: 11px;
}
.scene-tools {
  position: absolute;
  right: 14px;
  top: 16px;
  display: flex;
  gap: 5px;
  max-width: 60%;
  overflow-x: auto;
  z-index: 3;
  padding-bottom: 2px;
}
.scene-tools button,
.inspection-tools button {
  border: 1px solid var(--facility-line);
  background: color-mix(in srgb, var(--facility-panel) 92%, transparent);
  color: var(--facility-text);
  border-radius: 4px;
  padding: 6px 8px;
  font-size: 11px;
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
}
.scene-tools button.active {
  background: var(--facility-selected);
  border-color: var(--facility-accent);
  color: var(--facility-accent);
}
.inspection-tools {
  position: absolute;
  left: 18px;
  top: 100px;
  display: flex;
  gap: 6px;
  z-index: 3;
}
.scene-bottom {
  position: absolute;
  bottom: 12px;
  left: 18px;
  right: 18px;
  display: flex;
  justify-content: space-between;
  gap: 10px;
  color: var(--facility-muted);
  font-size: 10px;
  pointer-events: none;
}
.scene-bottom i {
  display: inline-block;
  width: 5px;
  height: 5px;
  background: #22b58c;
  border-radius: 50%;
  margin-right: 6px;
}
.scene-failure {
  position: absolute;
  inset: 145px 20px auto;
  color: var(--facility-text);
}
.scene-tooltip {
  position: absolute;
  z-index: 3;
  pointer-events: none;
  border: 1px solid var(--facility-line);
  background: var(--facility-panel);
  color: var(--facility-text);
  padding: 6px 10px;
  border-radius: 4px;
  font-size: 12px;
}
@media (max-width: 1550px) {
  .scene-tools {
    top: 10px;
    left: 12px;
    right: 12px;
    max-width: none;
  }
  .scene-caption {
    top: 50px;
    left: 14px;
  }
  .inspection-tools {
    top: 126px;
    left: 14px;
  }
  .scene-bottom > span:last-child {
    display: none;
  }
}
@media (max-width: 900px), (max-height: 650px) {
  .scene-caption > span,
  .scene-caption small {
    display: none;
  }
  .scene-caption strong {
    font-size: 14px;
  }
  .inspection-tools {
    top: 84px;
  }
  .scene-bottom {
    left: 12px;
    right: 12px;
  }
  .scene-bottom > span {
    display: none;
  }
}
</style>
