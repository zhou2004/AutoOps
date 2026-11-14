// Explicit opt-in visual demonstration; never submitted to inventory APIs.
export const demoRooms = [
  {
    id: 90001,
    name: "北京 · 北方数据中心",
    shortName: "BEIJING",
    country: "中国",
    city: "北京",
    longitude: 116.4074,
    latitude: 39.9042,
    address: "演示园区 · A 栋 / 数据大厅",
    level: "T3",
    status: 1
  },
  {
    id: 90002,
    name: "上海 · 华东数据中心",
    shortName: "SHANGHAI",
    country: "中国",
    city: "上海",
    longitude: 121.4737,
    latitude: 31.2304,
    address: "演示园区 · B 栋",
    level: "T3",
    status: 1
  },
  {
    id: 90003,
    name: "新加坡 · 亚太数据中心",
    shortName: "SINGAPORE",
    country: "新加坡",
    city: "新加坡",
    longitude: 103.8198,
    latitude: 1.3521,
    address: "演示园区 · APAC",
    level: "T3",
    status: 1
  },
  {
    id: 90004,
    name: "法兰克福 · 欧洲数据中心",
    shortName: "FRANKFURT",
    country: "德国",
    city: "法兰克福",
    longitude: 8.6821,
    latitude: 50.1109,
    address: "演示园区 · EU",
    level: "T3",
    status: 1
  },
  {
    id: 90005,
    name: "阿什本 · 北美数据中心",
    shortName: "ASHBURN",
    country: "美国",
    city: "阿什本",
    longitude: -77.4874,
    latitude: 39.0438,
    address: "演示园区 · US",
    level: "T3",
    status: 1
  }
];
export function demoTopology(room) {
  const cabinets = Array.from({ length: 10 }, (_, i) => {
    const id = 700 + i,
      devices = [];
    for (let n = 0; n < 13; n++) {
      const network = n >= 11;
      devices.push({
        id: id * 100 + n,
        kind: network ? "network" : "physical",
        name: network
          ? `汇聚交换机-${i + 1}-${n - 10}`
          : `计算节点-${i + 1}-${n + 1}`,
        sn: `DEMO-${id}-${n}`,
        manageIp: `10.20.${i + 1}.${n + 1}`,
        brand: network ? "NETWORK" : "SERVER",
        model: network ? "24 PORT GbE" : "2U COMPUTE",
        unitPosition: network ? 39 + (n - 11) : 1 + n * 3,
        unitHeight: network ? 1 : 2,
        cabinetId: id,
        status: 1,
        deviceType: 2,
        assetStatus: 2
      });
    }
    return {
      id,
      idcId: room.id,
      name: `${i < 5 ? "A" : "B"}-${String((i % 5) + 1).padStart(2, "0")}`,
      position: `${i < 5 ? "A" : "B"} 排 ${(i % 5) + 1} 号`,
      unitNum: 42,
      usedUnit: 24,
      widthMm: 600,
      depthMm: 1000,
      layoutX: (i % 5) * 1.1,
      layoutZ: Math.floor(i / 5) * 2.7,
      rotation: 0,
      devices,
      occupiedUnits: devices.flatMap(d =>
        Array.from({ length: d.unitHeight }, (_, i) => d.unitPosition + i)
      ),
      conflictUnits: [],
      invalidPlacements: 0,
      unpositioned: 0
    };
  });
  return { room, cabinets, unplaced: [] };
}
