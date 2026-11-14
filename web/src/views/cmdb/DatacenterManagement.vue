<template>
  <div
    ref="shell"
    class="facility-shell facility-surface"
    :data-facility-theme="dark ? 'dark' : 'light'"
    :style="{ '--facility-height': height }"
    :class="{ 'inventory-mode': tab === 'inventory' }"
  >
    <header class="facility-header">
      <div class="facility-title">
        <AssetIcon kind="idc" :size="27" />
        <div>
          <h1>{{ tr("机房管理") }}</h1>
          <small>DATA CENTER OPERATIONS</small>
        </div>
      </div>
      <nav class="view-tabs">
        <button :class="{ active: tab === 'visual' }" @click="setTab('visual')">
          {{ tr("可视化总览") }}</button
        ><button
          :class="{ active: tab === 'inventory' }"
          @click="setTab('inventory')"
        >
          {{ tr("机房与机柜台账") }}
        </button>
      </nav>
      <div class="facility-actions">
        <button v-if="tab === 'visual'" @click="showStats">
          {{ tr("统计详情") }}
        </button>
        <span v-if="demo" class="demo-badge">{{ tr("演示场景") }}</span
        ><button :class="{ selected: demo }" @click="toggleDemo">
          {{ demo ? tr("返回真实资产") : tr("体验示例场景") }}
        </button>
      </div>
    </header>
    <div v-if="tab === 'inventory'" class="inventory-content">
      <Machine embedded />
    </div>
    <template v-else>
      <div class="facility-breadcrumb">
        <button
          v-if="!selectedRoom"
          class="directory-toggle"
          :aria-expanded="directoryOpen"
          @click="directoryOpen = !directoryOpen"
        >
          {{ tr("IDC 目录") }}
        </button>
        <button @click="returnWorld">{{ tr("全球 IDC") }}</button
        ><template v-if="selectedRoom"
          ><span>/</span><strong>{{ selectedRoom.name }}</strong
          ><button class="back-world" @click="returnWorld">
            {{ tr("← 返回世界地图") }}
          </button></template
        ><span v-else class="world-desc">{{
          tr("世界地图 · 数据中心 · 机柜 · U 位设备")
        }}</span
        ><span class="data-origin">{{
          demo ? tr("示例数据 / 仅演示") : tr("CMDB 资产台账")
        }}</span>
      </div>
      <el-alert v-if="error" :title="error" type="error" :closable="false" />
      <Transition name="facility-scene" mode="out-in">
        <section
          v-if="!selectedRoom"
          key="world"
          v-loading="loading"
          class="global-layout"
          :class="{ 'directory-open': directoryOpen }"
        >
          <aside class="global-panel">
            <div class="section-kicker">RESOURCE DISTRIBUTION</div>
            <h2>{{ tr("数据中心分布") }}</h2>
            <div class="section-kicker">DATA CENTER DIRECTORY</div>
            <el-input
              v-model="search"
              :placeholder="tr('搜索名称、地区、城市')"
              clearable
              class="room-search"
            /><button
              v-for="r in filteredRooms"
              :key="r.id"
              class="idc-item"
              @click="enterRoom(r)"
              @mouseenter="globe?.locate(r)"
            >
              <span
                class="idc-dot"
                :class="{ disabled: r.status !== 1 }"
              /><span
                ><strong>{{ r.name }}</strong
                ><small>{{
                  [r.country, r.city].filter(Boolean).join(" / ") ||
                  r.address ||
                  tr("地区未登记")
                }}</small></span
              ><span class="idc-arrow">↗</span>
            </button>
            <p v-if="!filteredRooms.length" class="directory-empty">
              {{ tr("暂无匹配 IDC。可登记机房，或体验示例场景。") }}
            </p>
            <button class="inventory-link" @click="setTab('inventory')">
              {{ tr("管理机房与地理位置 →") }}
            </button>
          </aside>
          <IDCGlobe ref="globe" :rooms="rooms" @enter="enterRoom" />
          <aside class="global-info">
            <div class="section-kicker">SPATIAL NAVIGATION</div>
            <h3>{{ tr("从全球到每个 U 位") }}</h3>
            <ol>
              <li>
                <b>01</b>
                <div>
                  <strong>{{ tr("全球数据中心") }}</strong
                  ><small>{{ tr("经纬度定位与区域分布") }}</small>
                </div>
              </li>
              <li>
                <b>02</b>
                <div>
                  <strong>{{ tr("IDC 机房总览") }}</strong
                  ><small>{{ tr("机柜阵列、空间与容量") }}</small>
                </div>
              </li>
              <li>
                <b>03</b>
                <div>
                  <strong>{{ tr("机柜内部") }}</strong
                  ><small>{{ tr("开门巡检与逐 U 查看") }}</small>
                </div>
              </li>
              <li>
                <b>04</b>
                <div>
                  <strong>{{ tr("设备面板") }}</strong
                  ><small>{{ tr("服务器 / 交换机抽出查看") }}</small>
                </div>
              </li>
            </ol>
            <div class="panel-divider" />
            <div class="location-note">
              <span>{{ tr("定位完整度") }}</span
              ><strong
                >{{
                  rooms.length
                    ? Math.round((located.length / rooms.length) * 100)
                    : 0
                }}<small>%</small></strong
              >
              <div class="location-track">
                <i
                  :style="{
                    width: `${rooms.length ? (located.length / rooms.length) * 100 : 0}%`
                  }"
                />
              </div>
              <p>
                {{
                  tr(
                    "未登记经纬度的机房保留在目录中，可直接进入。地图按已保存坐标显示。"
                  )
                }}
              </p>
            </div>
            <p class="model-note">
              {{
                demo
                  ? tr("当前为示例园区和资产。切回真实资产后显示已授权机房。")
                  : tr(
                      "三维机柜与 U 位对应真实台账，场景建筑与气流为展示示意。"
                    )
              }}
            </p>
          </aside>
        </section>
        <Datacenter3D
          v-else
          ref="roomView"
          :key="`${demo}-${selectedRoom.id}`"
          :initial-room="selectedRoom"
          :demo="demo"
          @world="returnWorld"
        />
      </Transition>
    </template>
    <el-dialog
      v-model="statsVisible"
      :title="tr('数据中心统计')"
      width="min(560px, 94vw)"
      class="facility-surface"
      :data-facility-theme="dark ? 'dark' : 'light'"
    >
      <div class="facility-stat-grid">
        <article>
          <span>{{ tr("可见 IDC") }}</span
          ><strong>{{ rooms.length }}</strong
          ><small>{{ tr("当前账号可见范围") }}</small>
        </article>
        <article>
          <span>{{ tr("已定位") }}</span
          ><strong>{{ located.length }}</strong
          ><small>{{ tr("已登记经纬度") }}</small>
        </article>
        <article>
          <span>{{ tr("待定位") }}</span
          ><strong>{{ rooms.length - located.length }}</strong
          ><small>{{ tr("仍可从目录进入") }}</small>
        </article>
        <article>
          <span>{{ tr("国家 / 地区") }}</span
          ><strong>{{ regions }}</strong
          ><small>{{ tr("按台账登记统计") }}</small>
        </article>
      </div>
    </el-dialog>
  </div>
</template>
<script setup>
import { tr } from "@/locales";
import { ref, computed, onMounted, onBeforeUnmount, watch, provide } from "vue";
import { useRoute, useRouter } from "vue-router";
import AssetIcon from "@/components/AssetIcon.vue";
import Machine from "@/views/system/Machine.vue";
import Datacenter3D from "./Datacenter3D.vue";
import IDCGlobe from "./components/IDCGlobe.vue";
import { demoRooms } from "./three/demoFacility";
import api from "@/api/cmdb";
import {
  useFacilityAppearance,
  useFacilityViewport
} from "./composables/useFacilityView";
import "./three/facilityTheme.css";
const { dark } = useFacilityAppearance();
provide("facilityDark", dark);
const roomView = ref(),
  statsVisible = ref(false),
  directoryOpen = ref(false);
function showStats() {
  if (selectedRoom.value) roomView.value?.showSummary();
  else statsVisible.value = true;
}
const route = useRoute(),
  router = useRouter(),
  shell = ref(),
  globe = ref(),
  tab = ref(route.query.tab === "inventory" ? "inventory" : "visual"),
  rooms = ref([]),
  selectedRoom = ref(null),
  demo = ref(false),
  error = ref(""),
  loading = ref(false),
  search = ref("");
const { height } = useFacilityViewport(shell);
const located = computed(() =>
    rooms.value.filter(
      r => Number.isFinite(r.latitude) && Number.isFinite(r.longitude)
    )
  ),
  regions = computed(
    () => new Set(rooms.value.map(r => r.country).filter(Boolean)).size
  ),
  filteredRooms = computed(() =>
    rooms.value.filter(r =>
      [r.name, r.country, r.city, r.address].some(v =>
        String(v || "")
          .toLowerCase()
          .includes(search.value.trim().toLowerCase())
      )
    )
  );
let requestId = 0;
async function loadRooms() {
  const id = ++requestId;
  error.value = "";
  if (demo.value) {
    rooms.value = demoRooms;
    loading.value = false;
    return;
  }
  loading.value = true;
  try {
    const r = await api.getIdcAll();
    if (id !== requestId) return;
    if (r.data.code !== 200)
      throw new Error(r.data.message || tr("无法读取机房"));
    rooms.value = r.data.data.list || [];
    const room = rooms.value.find(r => r.id === Number(route.query.idcId));
    if (room && tab.value === "visual") selectedRoom.value = room;
  } catch (e) {
    if (id === requestId) {
      rooms.value = [];
      error.value = e.message;
    }
  } finally {
    if (id === requestId) loading.value = false;
  }
}
async function setTab(value) {
  tab.value = value;
  if (value === "inventory") demo.value = false;
  await router.replace({
    path: "/cmdb/datacenter",
    query: value === "inventory" ? { tab: value } : {}
  });
  if (value === "visual") loadRooms();
}
function enterRoom(room) {
  directoryOpen.value = false;
  selectedRoom.value = room;
  if (!demo.value)
    router.replace({ path: "/cmdb/datacenter", query: { idcId: room.id } });
}
function returnWorld() {
  selectedRoom.value = null;
  router.replace({ path: "/cmdb/datacenter", query: {} });
}
function toggleDemo() {
  requestId++;
  demo.value = !demo.value;
  selectedRoom.value = null;
  tab.value = "visual";
  router.replace({ path: "/cmdb/datacenter", query: {} });
  loadRooms();
}
watch(
  () => [route.query.tab, route.query.idcId],
  ([value, idcId], [previousTab, previousId]) => {
    if (route.path !== "/cmdb/datacenter") return;
    tab.value = value === "inventory" ? "inventory" : "visual";
    if (tab.value === "inventory") {
      demo.value = false;
      selectedRoom.value = null;
    } else if (idcId && (idcId !== previousId || previousTab === "inventory")) {
      demo.value = false;
      loadRooms();
    } else if (!idcId && previousId) {
      selectedRoom.value = null;
    }
  }
);
onMounted(loadRooms);
onBeforeUnmount(() => requestId++);
</script>
<style scoped>
.facility-shell {
  margin: 12px !important;
  padding: 0 !important;
  width: calc(100% - 24px);
  height: var(--facility-height);
  min-height: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--facility-line);
  border-radius: 10px;
  background: var(--facility-bg);
  overflow: hidden;
}
.facility-shell.inventory-mode {
  height: auto;
  min-height: var(--facility-height);
}
.facility-header {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-shrink: 0;
  padding: 10px 18px;
  background: var(--facility-panel);
  border-bottom: 1px solid var(--facility-line);
}
.facility-title {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--facility-accent);
  flex-shrink: 0;
}
.facility-title h1 {
  margin: 0;
  color: var(--facility-text);
  font-size: 18px;
  font-weight: 600;
}
.facility-title small {
  display: block;
  margin-top: 3px;
  color: var(--facility-muted);
  font: 8px monospace;
  letter-spacing: 1.5px;
}
.view-tabs {
  display: flex;
  gap: 4px;
}
.view-tabs button,
.facility-actions button,
.back-world,
.directory-toggle {
  background: var(--facility-panel);
  border: 1px solid var(--facility-line);
  color: var(--facility-muted);
  border-radius: 5px;
  padding: 7px 10px;
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
}
.view-tabs button {
  border-color: transparent;
}
.view-tabs button.active {
  color: var(--facility-accent);
  background: var(--facility-selected);
  border-color: var(--facility-line);
}
.facility-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 7px;
}
.facility-actions .selected,
.demo-badge {
  color: var(--facility-warning);
}
.demo-badge {
  font-size: 11px;
  white-space: nowrap;
}
.facility-breadcrumb {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 18px;
  min-height: 38px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--facility-line);
  font-size: 12px;
}
.facility-breadcrumb > button:not(.back-world):not(.directory-toggle) {
  color: var(--facility-accent);
  background: transparent;
  border: 0;
  cursor: pointer;
  padding: 0;
}
.facility-breadcrumb strong {
  font-weight: 500;
  color: var(--facility-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.world-desc,
.data-origin {
  color: var(--facility-muted);
  font-size: 11px;
}
.data-origin {
  margin-left: auto;
  white-space: nowrap;
}
.global-layout {
  position: relative;
  flex: 1;
  display: grid;
  grid-template-columns: 230px minmax(0, 1fr) 210px;
  min-height: 0;
  overflow: hidden;
}
.global-panel,
.global-info {
  min-height: 0;
  overflow: auto;
  padding: 20px 18px;
  background: var(--facility-panel);
  border-right: 1px solid var(--facility-line);
  z-index: 4;
}
.global-info {
  border-right: 0;
  border-left: 1px solid var(--facility-line);
}
.section-kicker {
  font: 8px monospace;
  letter-spacing: 1.8px;
  color: var(--facility-muted);
}
.global-panel h2,
.global-info h3 {
  font-size: 15px;
  font-weight: 500;
  color: var(--facility-text);
  margin: 8px 0 20px;
}
.room-search {
  margin: 14px 0;
}
.idc-item {
  display: flex;
  gap: 9px;
  align-items: center;
  text-align: left;
  width: 100%;
  border: 0;
  border-bottom: 1px solid var(--facility-line);
  padding: 14px 0;
  background: transparent;
  color: var(--facility-text);
  cursor: pointer;
}
.idc-item:hover {
  color: var(--facility-accent);
}
.idc-item strong {
  font-size: 12px;
  font-weight: 500;
  display: block;
}
.idc-item small {
  display: block;
  color: var(--facility-muted);
  font-size: 10px;
  margin-top: 6px;
}
.idc-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #22b58c;
  flex-shrink: 0;
}
.idc-dot.disabled {
  background: #97a5b1;
}
.idc-arrow {
  margin-left: auto;
  color: var(--facility-accent);
}
.inventory-link {
  margin-top: 20px;
  color: var(--facility-accent);
  border: 0;
  background: transparent;
  font-size: 11px;
  cursor: pointer;
}
.global-info ol {
  list-style: none;
  padding: 0;
}
.global-info li {
  display: flex;
  gap: 12px;
  align-items: center;
  margin: 22px 0;
}
.global-info li b {
  font: 21px monospace;
  color: var(--facility-accent);
  opacity: 0.65;
}
.global-info li strong {
  display: block;
  color: var(--facility-text);
  font-size: 12px;
  font-weight: 500;
}
.global-info li small {
  display: block;
  font-size: 10px;
  color: var(--facility-muted);
  margin-top: 6px;
}
.panel-divider {
  height: 1px;
  background: var(--facility-line);
  margin: 20px 0;
}
.location-note > span {
  font-size: 11px;
  color: var(--facility-muted);
}
.location-note > strong {
  display: block;
  font: 30px monospace;
  color: var(--facility-accent);
  margin: 12px 0;
}
.location-note strong small {
  font-size: 14px;
  margin-left: 5px;
}
.location-track {
  height: 4px;
  border-radius: 3px;
  background: var(--facility-line);
  overflow: hidden;
}
.location-track i {
  height: 100%;
  display: block;
  background: var(--facility-accent);
}
.location-note p,
.model-note,
.directory-empty {
  font-size: 11px;
  color: var(--facility-muted);
  line-height: 1.8;
}
.model-note {
  margin-top: 24px;
}
.inventory-content {
  padding: 16px;
  background: var(--bg-page);
  color: var(--text-primary);
}
.directory-toggle {
  display: none;
}
.facility-shell:fullscreen {
  margin: 0 !important;
  width: 100%;
  border: 0;
  border-radius: 0;
}
.facility-scene-enter-active,
.facility-scene-leave-active {
  transition: opacity 0.2s;
}
.facility-scene-enter-from,
.facility-scene-leave-to {
  opacity: 0;
}
@media (max-width: 1440px) {
  .global-layout {
    grid-template-columns: 215px minmax(0, 1fr);
  }
  .global-info {
    display: none;
  }
  .facility-header {
    gap: 10px;
  }
  .facility-title small {
    display: none;
  }
}
@media (max-width: 1050px) {
  .facility-header {
    flex-wrap: wrap;
    padding: 8px 12px;
    gap: 6px;
  }
  .view-tabs {
    order: 3;
    width: 100%;
  }
  .facility-actions {
    gap: 4px;
  }
  .demo-badge {
    display: none;
  }
}
@media (max-width: 900px) {
  .directory-toggle {
    display: inline-block;
  }
  .global-layout {
    display: block;
  }
  .global-panel {
    display: none;
    position: absolute;
    top: 0;
    left: 0;
    bottom: 0;
    width: min(280px, 85%);
    box-shadow: 8px 0 24px #0002;
  }
  .directory-open .global-panel {
    display: block;
  }
}
@media (max-width: 600px) {
  .facility-shell {
    margin: 6px !important;
    width: calc(100% - 12px);
  }
  .facility-title h1 {
    font-size: 16px;
  }
  .facility-title svg {
    display: none;
  }
  .facility-actions {
    flex-wrap: wrap;
    justify-content: flex-end;
  }
  .facility-actions button {
    padding: 5px 6px;
    font-size: 11px;
  }
  .facility-breadcrumb {
    padding: 6px 10px;
    gap: 6px;
  }
  .world-desc,
  .data-origin {
    display: none;
  }
  .view-tabs button {
    padding: 5px 8px;
  }
  .back-world {
    font-size: 11px;
  }
}
@media (max-height: 600px) and (min-width: 1051px) {
  .facility-header {
    padding-block: 6px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .facility-scene-enter-active,
  .facility-scene-leave-active {
    transition: none;
  }
}
</style>
