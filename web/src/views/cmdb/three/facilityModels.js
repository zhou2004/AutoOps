import * as T from "three";
import { RoundedBoxGeometry } from "three/addons/geometries/RoundedBoxGeometry.js";
export const U = 0.04445;
export function dispose(root) {
  const geometries = new Set(),
    materials = new Set(),
    textures = new Set();
  root?.traverse(o => {
    if (o.isInstancedMesh) o.dispose();
    if (o.geometry) geometries.add(o.geometry);
    for (const m of o.material
      ? Array.isArray(o.material)
        ? o.material
        : [o.material]
      : []) {
      materials.add(m);
      for (const key of ["map", "normalMap", "roughnessMap", "alphaMap"])
        if (m[key]) textures.add(m[key]);
    }
  });
  textures.forEach(x => x.dispose());
  materials.forEach(x => x.dispose());
  geometries.forEach(x => x.dispose());
}
export function material(color, metalness = 0.5, roughness = 0.5) {
  return new T.MeshStandardMaterial({ color, metalness, roughness });
}
export function box(parent, w, h, d, x, y, z, mat, round = 0) {
  const mesh = new T.Mesh(
    round
      ? new RoundedBoxGeometry(w, h, d, 2, round)
      : new T.BoxGeometry(w, h, d),
    typeof mat === "number" ? material(mat) : mat
  );
  mesh.position.set(x, y, z);
  mesh.castShadow = true;
  mesh.receiveShadow = true;
  parent.add(mesh);
  return mesh;
}
export function textSprite(parent, text, w, h, x, y, z, color = "#bfeaff") {
  const canvas = document.createElement("canvas");
  canvas.width = 1024;
  canvas.height = 128;
  const c = canvas.getContext("2d");
  c.fillStyle = color;
  c.font = "500 52px sans-serif";
  c.textAlign = "center";
  c.textBaseline = "middle";
  c.fillText(text, 512, 64, 990);
  const texture = new T.CanvasTexture(canvas);
  texture.colorSpace = T.SRGBColorSpace;
  const mesh = new T.Sprite(
    new T.SpriteMaterial({ map: texture, depthTest: true })
  );
  mesh.position.set(x, y, z);
  mesh.scale.set(w, h, 1);
  parent.add(mesh);
  return mesh;
}
// Procedural faceplates are original illustrations. Port lights are visual accents, not live traffic.
function topPanelTexture() {
  const canvas = document.createElement("canvas");
  canvas.width = 512;
  canvas.height = 768;
  const ctx = canvas.getContext("2d");
  ctx.fillStyle = "#758390";
  ctx.fillRect(0, 0, 512, 768);
  ctx.strokeStyle = "#4a5966";
  ctx.lineWidth = 2;
  ctx.strokeRect(12, 12, 488, 744);
  for (let x = 48; x < 470; x += 18) {
    for (let y = 70; y < 240; y += 16) {
      ctx.fillStyle = "#293b49";
      ctx.fillRect(x, y, 9, 4);
      ctx.fillStyle = "#a7b7c3";
      ctx.fillRect(x, y + 4, 9, 1);
    }
  }
  ctx.fillStyle = "#aab7c0";
  ctx.fillRect(44, 490, 200, 115);
  ctx.fillStyle = "#364653";
  ctx.font = "bold 13px monospace";
  ctx.fillText("SYSTEM / TOP COVER", 56, 514);
  for (let n = 0; n < 6; n++) ctx.fillRect(56, 529 + n * 9, 140 - n * 9, 2);
  for (const x of [24, 488])
    for (const y of [24, 380, 744]) {
      ctx.fillStyle = "#354653";
      ctx.beginPath();
      ctx.arc(x, y, 5, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = "#c6d1d8";
      ctx.fillRect(x - 3, y - 1, 6, 2);
    }
  const texture = new T.CanvasTexture(canvas);
  texture.colorSpace = T.SRGBColorSpace;
  texture.anisotropy = 4;
  return texture;
}
export function deviceTexture(device) {
  const canvas = document.createElement("canvas");
  canvas.width = 1536;
  canvas.height = 256;
  const c = canvas.getContext("2d");
  const network = device.kind === "network";
  const gradient = c.createLinearGradient(0, 0, 0, 256);
  gradient.addColorStop(0, network ? "#d2dae1" : "#56616c");
  gradient.addColorStop(0.45, network ? "#8998a8" : "#26313d");
  gradient.addColorStop(1, network ? "#536779" : "#101924");
  c.fillStyle = gradient;
  c.fillRect(0, 0, 1536, 256);
  c.strokeStyle = "#e1e8ee";
  c.lineWidth = 2;
  c.strokeRect(4, 4, 1528, 248);
  for (const x of [22, 1514])
    for (const y of [30, 226]) {
      c.fillStyle = "#060c13";
      c.beginPath();
      c.arc(x, y, 7, 0, Math.PI * 2);
      c.fill();
      c.strokeStyle = "#91a4b6";
      c.beginPath();
      c.moveTo(x - 4, y);
      c.lineTo(x + 4, y);
      c.stroke();
    }
  if (network) {
    const ports = device.deviceType === 1 ? 8 : 24;
    const rows = ports > 12 ? 2 : 1;
    for (let n = 0; n < ports; n++) {
      const x = 245 + (n % 12) * 76,
        y = rows === 2 ? 45 + Math.floor(n / 12) * 88 : 91;
      c.fillStyle = "#23313e";
      c.fillRect(x - 4, y - 4, 64, 64);
      c.fillStyle = "#03090e";
      c.fillRect(x, y, 56, 48);
      c.fillRect(x + 13, y + 44, 30, 10);
      c.fillStyle = "#c0a165";
      for (let j = 0; j < 8; j++) c.fillRect(x + 6 + j * 6, y + 4, 2, 10);
      c.fillStyle = n % 4 === 0 ? "#77edc1" : "#2e6159";
      c.fillRect(x + 1, y + 63, 12, 4);
      c.fillStyle = "#eaf3fc";
      c.font = "15px monospace";
      c.fillText(String(n + 1), x + 21, y - 10);
    }
    c.fillStyle = "#132733";
    c.fillRect(1235, 70, 110, 110);
    c.strokeStyle = "#6c859b";
    for (let i = 0; i < 4; i++)
      c.strokeRect(1240 + (i % 2) * 50, 75 + Math.floor(i / 2) * 50, 40, 35);
  } else {
    const rows = device.unitHeight > 1 ? 2 : 1;
    for (let n = 0; n < 12; n++) {
      const x = 190 + (n % 6) * 175,
        y = rows === 2 ? 24 + Math.floor(n / 6) * 105 : 70;
      if (rows === 1 && n >= 6) break;
      c.fillStyle = "#0b121b";
      c.fillRect(x, y, 161, 94);
      c.strokeStyle = "#687887";
      c.strokeRect(x + 2, y + 2, 157, 90);
      c.fillStyle = "#202f3e";
      c.fillRect(x + 8, y + 8, 120, 77);
      c.strokeStyle = "#09121b";
      c.lineWidth = 3;
      for (let q = 0; q < 10; q++) {
        c.beginPath();
        c.moveTo(x + 13, y + 13 + q * 6);
        c.lineTo(x + 118, y + 13 + q * 6);
        c.stroke();
      }
      c.fillStyle = "#bed1dc";
      c.fillRect(x + 136, y + 15, 8, 60);
      c.fillStyle = "#43bdad";
      c.fillRect(x + 14, y + 83, 24, 3);
    }
  }
  c.fillStyle = network ? "#17374a" : "#d6e8f5";
  c.font = "bold 26px sans-serif";
  c.fillText(
    (device.brand || (network ? "NETWORK" : "SERVER")).slice(0, 12),
    45,
    73
  );
  c.font = "17px monospace";
  c.fillText(
    (device.model || (network ? "ETHERNET" : "RACK SERVER")).slice(0, 16),
    45,
    107
  );
  c.fillStyle = "#62e6b8";
  c.beginPath();
  c.arc(1438, 60, 8, 0, Math.PI * 2);
  c.fill();
  c.fillStyle = "#192b3c";
  for (let n = 0; n < 14; n++) c.fillRect(1360, 100 + n * 7, 110, 2);
  const texture = new T.CanvasTexture(canvas);
  texture.colorSpace = T.SRGBColorSpace;
  texture.anisotropy = 4;
  return texture;
}
export function makeRack(parent, r, index, textureCache = new Map()) {
  const root = new T.Group();
  root.position.set(
    r.layoutX ?? (index % 5) * 1.1,
    0,
    r.layoutZ ?? Math.floor(index / 5) * 2.8
  );
  root.rotation.y = T.MathUtils.degToRad(r.rotation || 0);
  parent.add(root);
  const w = (r.widthMm || 600) / 1000,
    d = (r.depthMm || 1000) / 1000,
    h = r.unitNum * U + 0.22;
  const frame = material(0x111c29, 0.75, 0.36),
    edge = material(0x536579, 0.65, 0.35),
    inside = material(0x1c2b3a, 0.4, 0.65);
  const base = box(root, w, 0.11, d, 0, 0.055, 0, frame, 0.018);
  base.userData = { rackId: r.id };
  box(root, w, 0.075, d, 0, h, 0, frame, 0.012);
  for (const side of [-1, 1]) {
    box(
      root,
      0.025,
      h - 0.04,
      d - 0.03,
      side * (w / 2 - 0.014),
      h / 2,
      0,
      frame
    );
    box(
      root,
      0.025,
      h - 0.15,
      0.04,
      side * (w / 2 - 0.05),
      h / 2,
      d / 2 - 0.07,
      edge
    );
  }
  box(root, w - 0.045, h - 0.1, 0.025, 0, h / 2, -d / 2, inside);
  // Repeated mounting holes share one draw call per cabinet.
  const holes = new T.InstancedMesh(
    new T.BoxGeometry(0.012, 0.008, 0.005),
    material(0x97adba),
    r.unitNum * 2
  );
  const holeMatrix = new T.Matrix4();
  for (let i = 0; i < r.unitNum; i++) {
    for (let j = 0; j < 2; j++) {
      holeMatrix.makeTranslation(
        (j ? 1 : -1) * (w / 2 - 0.048),
        0.145 + i * U,
        d / 2 - 0.045
      );
      holes.setMatrixAt(i * 2 + j, holeMatrix);
    }
  }
  holes.receiveShadow = true;
  holes.userData = { rackId: r.id };
  root.add(holes);
  const badge = box(
    root,
    w - 0.07,
    0.07,
    0.015,
    0,
    h - 0.07,
    d / 2 + 0.01,
    material(0x0b2942)
  );
  badge.userData = { rackId: r.id };
  textSprite(root, r.name, w * 0.8, 0.06, 0, h - 0.065, d / 2 + 0.03);
  const pivot = new T.Group();
  pivot.position.set(-w / 2 + 0.02, 0.12, d / 2 + 0.015);
  root.add(pivot);
  const door = new T.Group();
  door.position.x = (w - 0.04) / 2;
  pivot.add(door);
  const dh = h - 0.25;
  for (const x of [-1, 1])
    box(door, 0.018, dh, 0.024, (x * (w - 0.045)) / 2, dh / 2, 0, edge);
  for (const y of [0, dh]) box(door, w - 0.025, 0.018, 0.025, 0, y, 0, edge);
  const glass = box(
    door,
    w - 0.07,
    dh - 0.03,
    0.006,
    0,
    dh / 2,
    0,
    new T.MeshPhysicalMaterial({
      color: 0x59839e,
      transparent: true,
      opacity: 0.14,
      metalness: 0.1,
      roughness: 0.2,
      depthWrite: false
    })
  );
  glass.userData = { rackId: r.id };
  box(door, 0.018, 0.2, 0.035, w / 2 - 0.09, dh * 0.48, 0.03, 0xa8b6c3, 0.006);
  const glowMat = new T.MeshBasicMaterial({ color: 0x45cbff });
  const strip = box(
    root,
    0.009,
    h - 0.23,
    0.008,
    -w / 2 + 0.025,
    h / 2,
    d / 2 + 0.033,
    glowMat
  );
  const devices = [];
  for (const dev of r.devices) {
    if (
      dev.unitPosition < 1 ||
      dev.unitPosition + dev.unitHeight - 1 > r.unitNum
    )
      continue;
    const body = new T.Group();
    const height = dev.unitHeight * U - 0.003;
    body.position.set(
      0,
      0.13 + (dev.unitPosition - 1) * U + height / 2,
      d / 2 - 0.09
    );
    root.add(body);
    const depth = d * 0.79;
    const chassis = box(
      body,
      w - 0.1,
      height,
      depth,
      0,
      0,
      -depth / 2,
      dev.restricted ? 0x384553 : 0x82909d,
      0.003
    );
    chassis.userData = { rackId: r.id, device: dev };
    if (!dev.restricted) {
      if (!textureCache.has("top-panel"))
        textureCache.set("top-panel", topPanelTexture());
      const cover = new T.Mesh(
        new T.PlaneGeometry(w - 0.103, depth - 0.005),
        new T.MeshStandardMaterial({
          map: textureCache.get("top-panel"),
          metalness: 0.55,
          roughness: 0.6
        })
      );
      cover.rotation.x = -Math.PI / 2;
      cover.position.set(0, height / 2 + 0.001, -depth / 2);
      cover.userData = { rackId: r.id, device: dev };
      body.add(cover);
    }

    const key = [
      dev.kind,
      dev.deviceType,
      dev.brand,
      dev.model,
      dev.unitHeight
    ].join("|");
    if (!dev.restricted && !textureCache.has(key))
      textureCache.set(key, deviceTexture(dev));
    const front = box(
      body,
      w - 0.085,
      height,
      0.012,
      0,
      0,
      0.008,
      dev.restricted
        ? material(0x3a4550)
        : new T.MeshStandardMaterial({
            map: textureCache.get(key),
            metalness: 0.35,
            roughness: 0.55
          })
    );
    front.userData = { rackId: r.id, device: dev };
    for (const side of [-1, 1])
      box(
        body,
        0.008,
        Math.min(height * 0.7, 0.055),
        0.018,
        side * (w / 2 - 0.055),
        0,
        0.024,
        0xc4d1da,
        0.003
      );
    const ledMat = new T.MeshBasicMaterial({
      color: dev.status === 1 ? 0x61efbd : 0xf2ad55
    });
    if (!dev.restricted)
      box(
        body,
        0.006,
        0.006,
        0.005,
        w / 2 - 0.087,
        height * 0.17,
        0.019,
        ledMat
      );
    devices.push({ body, baseZ: body.position.z, device: dev, ledMat });
  }
  const capacity = box(
    root,
    w * 0.9,
    0.02,
    d * 0.8,
    0,
    0.13,
    0,
    new T.MeshBasicMaterial({
      color: 0x24b9e6,
      transparent: true,
      opacity: 0.33,
      depthWrite: false
    })
  );
  capacity.visible = false;
  return { root, pivot, devices, strip, glowMat, w, d, h, capacity, record: r };
}
export function makeRoom(parent, bounds) {
  const c = bounds.getCenter(new T.Vector3()),
    s = bounds.getSize(new T.Vector3()),
    w = Math.max(7, s.x + 3.4),
    d = Math.max(6, s.z + 3.2),
    h = Math.max(2.8, s.y + 0.4);
  const floor = material(0x688295, 0.35, 0.6);
  box(parent, w, 0.15, d, c.x, -0.1, c.z, floor, 0.03);
  box(parent, w + 0.14, 0.07, d + 0.14, c.x, -0.21, c.z, 0x19394d, 0.02);
  const lines = [];
  for (let x = -w / 2; x <= w / 2; x += 0.6)
    lines.push(c.x + x, -0.02, c.z - d / 2, c.x + x, -0.02, c.z + d / 2);
  for (let z = -d / 2; z <= d / 2; z += 0.6)
    lines.push(c.x - w / 2, -0.02, c.z + z, c.x + w / 2, -0.02, c.z + z);
  const geo = new T.BufferGeometry();
  geo.setAttribute("position", new T.Float32BufferAttribute(lines, 3));
  parent.add(
    new T.LineSegments(
      geo,
      new T.LineBasicMaterial({
        color: 0x324e62,
        transparent: true,
        opacity: 0.7
      })
    )
  );
  const wall = material(0x344d65, 0.25, 0.7);
  box(parent, w, 0.3, 0.12, c.x, 0.13, c.z - d / 2, wall);
  box(parent, 0.12, h, d, c.x - w / 2, h / 2, c.z, wall);
  for (let i = 0; i < Math.ceil(d / 0.7); i++) {
    box(
      parent,
      0.025,
      h - 0.14,
      0.018,
      c.x - w / 2 + 0.07,
      h / 2,
      c.z - d / 2 + i * 0.7,
      0x7495ac
    );
  }
  box(
    parent,
    0.025,
    0.018,
    d - 0.1,
    c.x - w / 2 + 0.08,
    h - 0.1,
    c.z,
    new T.MeshBasicMaterial({ color: 0x6edfff })
  );
  for (let i = 0; i < 3; i++) {
    const x = c.x - w / 2 + 0.65 + i * 0.65,
      z = c.z - d / 2 + 0.35;
    box(parent, 0.5, 1.8, 0.42, x, 0.9, z, 0x8eafc1, 0.02);
    for (let q = 0; q < 16; q++)
      box(parent, 0.4, 0.016, 0.012, x, 0.5 + q * 0.065, z + 0.22, 0x1d364b);
    box(parent, 0.15, 0.12, 0.01, x, 1.62, z + 0.23, 0x0d3449);
    textSprite(parent, "COOLING", 0.4, 0.06, x, 1.75, z + 0.24);
  }
  for (const z of [c.z - d * 0.24, c.z + d * 0.24]) {
    for (const dx of [-0.13, 0.13])
      box(parent, w - 0.5, 0.035, 0.024, c.x + 0, h - 0.08, z + dx, 0x54738b);
    for (let x = -w / 2 + 0.3; x < w / 2 - 0.2; x += 0.25)
      box(parent, 0.018, 0.02, 0.28, c.x + x, h - 0.08, z, 0x718b9c);
    box(
      parent,
      w - 0.5,
      0.008,
      0.012,
      c.x,
      h - 0.055,
      z,
      new T.MeshBasicMaterial({ color: 0x28c6d7 })
    );
  }
  const stripe = new T.MeshBasicMaterial({ color: 0xf5bf62 });
  box(parent, w - 0.3, 0.003, 0.018, c.x, -0.012, c.z + d / 2 - 0.2, stripe);
  box(parent, 0.018, 0.003, d - 0.3, c.x + w / 2 - 0.2, -0.012, c.z, stripe);
  textSprite(
    parent,
    "DATA HALL / IT INFRASTRUCTURE",
    3.4,
    0.25,
    c.x,
    2.4,
    c.z - d / 2 + 0.1,
    "#bdeaff"
  );
  return { center: c, width: w, depth: d, height: h };
}
