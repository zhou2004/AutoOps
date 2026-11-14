<template>
  <div v-loading="loading" class="datacenter-page">
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      show-icon
      :closable="false"
      class="notice"
    />
    <el-alert
      v-if="autoLayoutCount"
      :title="
        tr(
          '{count} 个机柜尚未登记坐标，暂按自动排列展示；可在机柜布局中填写实测位置。',
          { count: autoLayoutCount }
        )
      "
      type="info"
      :closable="false"
      class="notice"
    />
    <el-alert
      v-if="issues"
      :title="tr('检测到历史 U 位重叠或越界，请修正台账中的位置与高度。')"
      type="warning"
      :closable="false"
      class="notice"
    />
    <div v-if="racks.length" class="room-panels-toolbar">
      <button
        class="rack-list-toggle"
        :aria-expanded="listOpen"
        @click="
          listOpen = !listOpen;
          unitsOpen = false;
        "
      >
        {{ tr("机柜目录") }}
      </button>
      <button
        class="rack-units-toggle"
        :aria-expanded="unitsOpen"
        @click="
          unitsOpen = !unitsOpen;
          listOpen = false;
        "
      >
        {{ tr("U 位面板") }}
      </button>
    </div>
    <div
      v-if="racks.length"
      class="workspace"
      :class="{ 'list-open': listOpen, 'units-open': unitsOpen }"
    >
      <aside class="rack-list">
        <div class="panel-title">
          {{ tr("机柜目录") }}<small>{{ racks.length }}</small>
        </div>
        <el-input
          v-model="search"
          :placeholder="tr('查找机柜、设备、IP 或 SN')"
          clearable
        />
        <button
          v-for="r in filteredRacks"
          :key="r.id"
          class="rack-choice"
          :class="{ active: r.id === selectedID }"
          @click="selectRack(r.id, true)"
        >
          <AssetIcon kind="cabinet" /><span
            ><strong>{{ r.name }}</strong
            ><small>{{ r.position || tr("位置未登记") }}</small
            ><el-progress
              :percentage="Math.round((r.usedUnit / r.unitNum) * 100)"
              :stroke-width="4"
              :show-text="false"
            /><small>{{ r.usedUnit }} / {{ r.unitNum }} U</small></span
          ><span v-if="r.conflictUnits.length" class="conflict-dot">!</span>
        </button>
        <el-empty
          v-if="!filteredRacks.length"
          :description="tr('无匹配设备或机柜')"
          :image-size="60"
        />
      </aside>
      <RackScene
        ref="scene"
        :racks="racks"
        :selected="selectedID"
        :room-name="topology.room.name"
        @select="pick"
      />
      <aside v-if="selected" class="rack-detail">
        <div class="panel-title">
          <span><AssetIcon kind="cabinet" :size="20" />{{ selected.name }}</span
          ><el-button
            v-if="!demo && canPlatform('cmdb:write')"
            size="small"
            @click="editLayout"
            >{{ tr("布局") }}</el-button
          >
        </div>
        <p class="rack-subtitle">
          {{ selected.widthMm || 600 }} × {{ selected.depthMm || 1000 }} mm ·
          {{ selected.unitNum }} U
        </p>
        <div class="legend">
          <span class="physical">{{ tr("物理机") }}</span
          ><span class="network">{{ tr("网络设备") }}</span
          ><span class="restricted">{{ tr("受限资产") }}</span>
        </div>
        <div class="rack-elevation">
          <div
            v-for="u in units"
            :key="u"
            class="u-row"
            :class="{ conflict: selected.conflictUnits.includes(u) }"
          >
            <span class="u-number">{{ u }}U</span
            ><button
              v-if="deviceAt(u)"
              class="u-device"
              :class="deviceAt(u).kind"
              @click="showDevice(deviceAt(u))"
            >
              <AssetIcon
                v-if="isTop(deviceAt(u), u)"
                :kind="deviceAt(u).kind"
                :device-type="deviceAt(u).deviceType"
                :size="18"
              /><span>{{
                isTop(deviceAt(u), u) ? deviceAt(u).name || deviceAt(u).sn : "↳"
              }}</span
              ><small v-if="isTop(deviceAt(u), u)"
                >{{ deviceAt(u).unitHeight }}U</small
              >
            </button>
            <div v-else class="u-empty">{{ tr("空闲") }}</div>
          </div>
        </div>
        <p class="rack-subtitle">{{ tr("U1 从底部起算 · 1U = 44.45 mm") }}</p>
        <el-alert
          v-if="selected.unpositioned"
          :title="
            tr('{count} 台设备尚未登记 U 位', { count: selected.unpositioned })
          "
          type="warning"
          :closable="false"
        />
        <button
          v-for="d in selected.devices.filter(d => !d.unitPosition)"
          :key="`${d.kind}-${d.id}`"
          class="unplaced-device"
          @click="showDevice(d)"
        >
          <AssetIcon :kind="d.kind" :device-type="d.deviceType" :size="20" />{{
            d.name || d.sn
          }}
        </button>
      </aside>
    </div>
    <el-empty
      v-else-if="!loading && !error"
      :description="
        roomID
          ? tr('该机房暂无可见机柜，请在台账中新增或检查授权')
          : tr('暂无可见机房，请先创建机房或申请授权')
      "
    />
    <el-dialog
      v-model="statsVisible"
      :title="tr('机房统计详情')"
      width="min(720px, 94vw)"
      class="facility-surface"
      :data-facility-theme="dark ? 'dark' : 'light'"
    >
      <div v-if="topology" class="facility-stat-grid">
        <article>
          <span>{{ tr("机柜") }}</span
          ><strong>{{ racks.length }}</strong
          ><small>{{ tr("当前可见范围") }}</small>
        </article>
        <article>
          <span>{{ tr("U 位占用") }}</span
          ><strong
            >{{ used }}<em>/ {{ capacity }} U</em></strong
          ><small
            >{{ capacity ? Math.round((used / capacity) * 100) : 0
            }}{{ tr("% 使用率") }}</small
          >
        </article>
        <article>
          <span>{{ tr("可用 U 位") }}</span
          ><strong>{{ capacity - used }}</strong
          ><small>{{ tr("已计入受限资产占用") }}</small>
        </article>
        <article>
          <span>{{ tr("待定位设备") }}</span
          ><strong>{{ unresolved }}</strong
          ><small>{{ tr("未分配机柜或 U 位") }}</small>
        </article>
      </div>
      <el-card
        v-if="topology?.unplaced.length"
        shadow="never"
        class="unplaced-card"
        ><template #header>{{ tr("未分配机柜的资产") }}</template
        ><el-table :data="topology.unplaced"
          ><el-table-column :label="tr('设备')"
            ><template #default="{ row }"
              ><span class="device-name"
                ><AssetIcon
                  :kind="row.kind"
                  :device-type="row.deviceType"
                  :size="24"
                />{{ row.name || row.sn }}</span
              ></template
            ></el-table-column
          ><el-table-column
            prop="manageIp"
            :label="tr('管理 IP')"
          /><el-table-column prop="sn" :label="tr('序列号')" /><el-table-column
            :label="tr('操作')"
            ><template #default="{ row }"
              ><el-button link type="primary" @click="showDevice(row)">{{
                tr("查看资产")
              }}</el-button></template
            ></el-table-column
          ></el-table
        ></el-card
      >
    </el-dialog>
    <el-drawer
      v-model="deviceVisible"
      class="facility-device-drawer facility-surface"
      :data-facility-theme="dark ? 'dark' : 'light'"
      modal-class="facility-device-overlay"
      :title="tr('U 位资产详情')"
      size="min(400px, 100vw)"
      :modal="false"
      :lock-scroll="false"
      @closed="scene?.resetDevice()"
      ><template v-if="device"
        ><div class="device-heading">
          <AssetIcon
            :kind="device.kind"
            :device-type="device.deviceType"
            :size="42"
          />
          <h2>{{ device.name || device.sn }}</h2>
        </div>
        <el-descriptions :column="1" border
          ><el-descriptions-item :label="tr('类型')">{{
            device.kind === "physical"
              ? tr("物理机")
              : tr(deviceTypes[device.deviceType]) || tr("网络设备")
          }}</el-descriptions-item
          ><el-descriptions-item :label="tr('机柜')">{{
            racks.find(r => r.id === device.cabinetId)?.name || tr("未分配")
          }}</el-descriptions-item
          ><el-descriptions-item :label="tr('U 位')">{{
            device.unitPosition
              ? `${device.unitPosition}–${device.unitPosition + device.unitHeight - 1} U`
              : tr("未定位")
          }}</el-descriptions-item
          ><el-descriptions-item :label="tr('高度')"
            >{{ device.unitHeight }} U</el-descriptions-item
          ><el-descriptions-item :label="tr('品牌 / 型号')"
            >{{ device.brand }} {{ device.model }}</el-descriptions-item
          ><el-descriptions-item :label="tr('管理 IP')">{{
            device.manageIp || tr("未登记")
          }}</el-descriptions-item
          ><el-descriptions-item :label="tr('序列号')">{{
            device.sn
          }}</el-descriptions-item
          ><el-descriptions-item :label="tr('台账运行状态')">{{
            { 1: tr("运行中"), 2: tr("关机"), 3: tr("离线") }[device.status] ||
            tr("未知")
          }}</el-descriptions-item></el-descriptions
        ><el-button
          v-if="!demo"
          type="primary"
          class="detail-action"
          @click="openAsset"
          >{{ tr("前往资产管理") }}</el-button
        >
        <p v-else class="demo-device-note">
          {{ tr("示例设备 · 面板与机箱为三维示意模型") }}
        </p></template
      ></el-drawer
    >
    <el-dialog
      v-model="layoutVisible"
      :title="tr('机柜布局与尺寸')"
      width="min(560px, 94vw)"
      ><el-form label-width="130px"
        ><el-form-item :label="tr('X 坐标 (米)')"
          ><el-input-number
            v-model="layout.layoutX"
            :min="-500"
            :max="500"
            :precision="3" /></el-form-item
        ><el-form-item :label="tr('Z 坐标 (米)')"
          ><el-input-number
            v-model="layout.layoutZ"
            :min="-500"
            :max="500"
            :precision="3" /></el-form-item
        ><el-form-item :label="tr('朝向')"
          ><el-select v-model="layout.rotation"
            ><el-option
              v-for="n in [0, 90, 180, 270]"
              :key="n"
              :label="`${n}°`"
              :value="n" /></el-select></el-form-item
        ><el-form-item :label="tr('宽度 (mm)')"
          ><el-input-number
            v-model="layout.widthMm"
            :min="400"
            :max="1200"
            :step="100" /></el-form-item
        ><el-form-item :label="tr('深度 (mm)')"
          ><el-input-number
            v-model="layout.depthMm"
            :min="400"
            :max="2000"
            :step="100"
        /></el-form-item>
        <p>
          {{
            tr(
              "坐标为机柜底面中心点，机房原点由你统一约定。朝向 0° 时柜门朝向 +Z。"
            )
          }}
        </p></el-form
      ><template #footer
        ><el-button @click="layoutVisible = false">{{ tr("取消") }}</el-button
        ><el-button type="primary" :loading="saving" @click="saveLayout">{{
          tr("保存布局")
        }}</el-button></template
      ></el-dialog
    >
  </div>
</template>
<script setup>
import { tr } from "@/locales";
import { computed, onMounted, ref, inject } from "vue";
import { useRouter, useRoute } from "vue-router";
import { ElMessage } from "element-plus";
import cmdb from "@/api/cmdb";
import request from "@/utils/request";
import { canPlatform } from "@/utils/platformAccess";
import AssetIcon from "@/components/AssetIcon.vue";
import RackScene from "./components/RackScene.vue";
import { demoTopology } from "./three/demoFacility";
const dark = inject("facilityDark", ref(false));
const statsVisible = ref(false),
  listOpen = ref(false),
  unitsOpen = ref(false);
defineExpose({
  showSummary: () => {
    statsVisible.value = true;
  }
});
const props = defineProps({ initialRoom: Object, demo: Boolean });
const router = useRouter(),
  route = useRoute(),
  rooms = ref([]),
  roomID = ref(),
  topology = ref(null),
  loading = ref(false),
  error = ref(""),
  selectedID = ref(),
  search = ref(""),
  scene = ref();
const deviceVisible = ref(false),
  device = ref(),
  layoutVisible = ref(false),
  layout = ref({}),
  saving = ref(false);
const deviceTypes = {
  1: "路由器",
  2: "交换机",
  3: "防火墙",
  4: "负载均衡",
  5: "其他网络设备"
};
const racks = computed(() => topology.value?.cabinets || []),
  selected = computed(() => racks.value.find(r => r.id === selectedID.value)),
  units = computed(() =>
    Array.from(
      { length: selected.value?.unitNum || 0 },
      (_, i) => selected.value.unitNum - i
    )
  );
const capacity = computed(() => racks.value.reduce((n, r) => n + r.unitNum, 0)),
  used = computed(() => racks.value.reduce((n, r) => n + r.usedUnit, 0)),
  unresolved = computed(
    () =>
      racks.value.reduce((n, r) => n + r.unpositioned, 0) +
      (topology.value?.unplaced.length || 0)
  ),
  autoLayoutCount = computed(
    () => racks.value.filter(r => r.layoutX == null || r.layoutZ == null).length
  ),
  issues = computed(() =>
    racks.value.some(r => r.conflictUnits.length || r.invalidPlacements)
  );
const filteredRacks = computed(() => {
  const q = search.value.trim().toLowerCase();
  return racks.value.filter(
    r =>
      !q ||
      [
        r.name,
        r.position,
        ...r.devices.flatMap(d => [d.name, d.sn, d.manageIp])
      ].some(v =>
        String(v || "")
          .toLowerCase()
          .includes(q)
      )
  );
});
function unpack(r) {
  if (r.data.code !== 200) throw new Error(r.data.message || tr("读取失败"));
  return r.data.data;
}
let generation = 0;
async function loadRoom() {
  const version = ++generation;
  topology.value = null;
  deviceVisible.value = false;
  error.value = "";
  if (!roomID.value) return;
  loading.value = true;
  try {
    const data = unpack(
      await request({ url: `cmdb/idc/${roomID.value}/topology`, method: "get" })
    );
    if (version !== generation) return;
    topology.value = data;
    if (!data.cabinets.some(r => r.id === selectedID.value))
      selectedID.value =
        data.cabinets.find(r => r.id === Number(route.query.cabinetId))?.id ||
        data.cabinets[0]?.id;
  } catch (e) {
    if (version === generation) error.value = e.message;
  } finally {
    if (version === generation) loading.value = false;
  }
}
async function reload() {
  if (props.initialRoom) {
    roomID.value = props.initialRoom.id;
    if (props.demo) {
      topology.value = demoTopology(props.initialRoom);
      selectedID.value = topology.value.cabinets[0]?.id;
      return;
    }
    await loadRoom();
    return;
  }
  loading.value = true;
  try {
    rooms.value = unpack(await cmdb.getIdcAll()).list || [];
    if (!rooms.value.some(r => r.id === roomID.value))
      roomID.value =
        rooms.value.find(r => r.id === Number(route.query.idcId))?.id ||
        rooms.value[0]?.id;
    await loadRoom();
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
}
function selectRack(id, focus = false) {
  if (focus) listOpen.value = false;
  selectedID.value = id;
  if (focus) scene.value?.focus(id);
}
function pick(hit) {
  selectRack(hit.rackId);
  if (hit.device) showDevice(hit.device);
}
function deviceAt(u) {
  return selected.value?.devices.find(
    d =>
      d.unitPosition > 0 &&
      u >= d.unitPosition &&
      u < d.unitPosition + d.unitHeight
  );
}
function isTop(d, u) {
  return (
    u === Math.min(selected.value.unitNum, d.unitPosition + d.unitHeight - 1)
  );
}
function showDevice(d) {
  if (d.restricted) {
    ElMessage.info(tr("此位置已占用，资产详情需要授权"));
    return;
  }
  device.value = d;
  scene.value?.inspectDevice(d);
  deviceVisible.value = true;
}
function openAsset() {
  if (props.demo) {
    ElMessage.info(tr("这是演示设备，未写入真实资产"));
    return;
  }
  router.push({
    path: device.value.kind === "physical" ? "/cmdb/physical" : "/cmdb/network",
    query: { keyword: device.value.sn || device.value.name }
  });
}
function editLayout() {
  const r = selected.value;
  layout.value = {
    ...r,
    layoutX: r.layoutX ?? 0,
    layoutZ: r.layoutZ ?? 0,
    widthMm: r.widthMm || 600,
    depthMm: r.depthMm || 1000
  };
  layoutVisible.value = true;
}
async function saveLayout() {
  saving.value = true;
  try {
    unpack(
      await cmdb.updateCabinet(layout.value.id, {
        ...layout.value,
        layoutX: layout.value.layoutX ?? null,
        layoutZ: layout.value.layoutZ ?? null
      })
    );
    layoutVisible.value = false;
    await loadRoom();
    ElMessage.success(tr("布局已保存"));
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    saving.value = false;
  }
}
onMounted(reload);
</script>
<style scoped>
.datacenter-page {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--facility-bg);
  color: var(--facility-text);
  overflow: hidden;
}
.workspace {
  position: relative;
  display: grid;
  grid-template-columns: 195px minmax(0, 1fr) 235px;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.rack-list,
.rack-detail {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 14px;
  background: var(--facility-panel);
}
.rack-list {
  border-right: 1px solid var(--facility-line);
}
.rack-detail {
  border-left: 1px solid var(--facility-line);
  display: flex;
  flex-direction: column;
}
.panel-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
  color: var(--facility-text);
  font-size: 13px;
  font-weight: 600;
  flex-shrink: 0;
}
.panel-title > span {
  display: flex;
  align-items: center;
  gap: 8px;
}
.panel-title small {
  color: var(--facility-muted);
}
.rack-choice {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  text-align: left;
  padding: 10px 8px;
  margin-top: 8px;
  border: 1px solid var(--facility-line);
  border-radius: 6px;
  background: var(--facility-panel-alt);
  color: var(--facility-muted);
  cursor: pointer;
}
.rack-choice.active {
  border-color: var(--facility-accent);
  background: var(--facility-selected);
  color: var(--facility-accent);
}
.rack-choice > span {
  flex: 1;
  min-width: 0;
}
.rack-choice strong,
.rack-choice small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
}
.rack-choice strong {
  font-size: 12px;
}
.rack-choice small {
  font-size: 10px;
  margin: 5px 0;
}
.rack-choice .el-progress {
  margin: 7px 0;
}
.rack-subtitle {
  color: var(--facility-muted);
  font-size: 11px;
  margin: 4px 0;
  flex-shrink: 0;
}
.legend {
  display: flex;
  gap: 6px;
  font-size: 10px;
  margin: 10px 0;
  flex-shrink: 0;
}
.legend span {
  padding: 4px;
  border-radius: 3px;
}
.physical {
  background: var(--facility-physical);
  color: var(--facility-physical-text);
}
.network {
  background: var(--facility-network);
  color: var(--facility-network-text);
}
.restricted {
  background: var(--facility-restricted);
  color: var(--facility-muted);
}
.rack-elevation {
  background: var(--facility-bg);
  border: 1px solid var(--facility-line);
  padding: 5px;
  border-radius: 5px;
  flex: 1;
  min-height: 100px;
  overflow: auto;
}
.u-row {
  display: flex;
  height: 25px;
  gap: 6px;
  align-items: stretch;
}
.u-number {
  font-size: 10px;
  width: 26px;
  text-align: right;
  line-height: 25px;
  color: var(--facility-muted);
  flex-shrink: 0;
}
.u-device,
.u-empty {
  border: 0;
  border-radius: 2px;
  margin: 1px 0;
  min-width: 0;
  flex: 1;
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 0 6px;
  font-size: 11px;
}
.u-device {
  cursor: pointer;
}
.u-device span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  text-align: left;
}
.u-empty {
  background: var(--facility-empty);
  color: var(--facility-muted);
  font-size: 10px;
}
.u-row.conflict {
  outline: 1px solid #f97316;
}
.conflict-dot {
  color: #f97316;
}
.notice {
  margin: 0;
  flex-shrink: 0;
  border-radius: 0;
}
.unplaced-card {
  margin-top: 18px;
}
.unplaced-device,
.device-name {
  display: flex;
  align-items: center;
  gap: 6px;
}
.unplaced-device {
  padding: 8px 0;
  border: 0;
  background: transparent;
  color: var(--facility-text);
  cursor: pointer;
}
.detail-action {
  margin-top: 20px;
}
.room-panels-toolbar {
  display: none;
  flex-shrink: 0;
  padding: 5px 12px;
  border-bottom: 1px solid var(--facility-line);
  background: var(--facility-panel);
  gap: 8px;
}
.room-panels-toolbar button {
  border: 1px solid var(--facility-line);
  background: var(--facility-panel-alt);
  color: var(--facility-accent);
  padding: 4px 9px;
  border-radius: 4px;
  font-size: 11px;
  cursor: pointer;
}
.rack-list-toggle {
  display: none;
}
@media (max-width: 1280px) {
  .workspace {
    grid-template-columns: 180px minmax(0, 1fr);
  }
  .room-panels-toolbar {
    display: flex;
    justify-content: flex-end;
  }
  .rack-detail {
    display: none;
    position: absolute;
    z-index: 7;
    right: 0;
    top: 0;
    bottom: 0;
    width: min(280px, 85%);
    box-shadow: -8px 0 24px #0002;
  }
  .units-open .rack-detail {
    display: flex;
  }
}
@media (max-width: 900px) {
  .workspace {
    display: block;
  }
  .rack-list {
    display: none;
    position: absolute;
    z-index: 7;
    left: 0;
    top: 0;
    bottom: 0;
    width: min(260px, 85%);
    box-shadow: 8px 0 24px #0002;
  }
  .list-open .rack-list {
    display: block;
  }
  .rack-list-toggle {
    display: inline-block;
  }
  .room-panels-toolbar {
    justify-content: flex-start;
  }
}
</style>
