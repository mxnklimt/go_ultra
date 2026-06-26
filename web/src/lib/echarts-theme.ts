export const ECHARTS_PALETTE = [
  "#4a9eff",
  "#7fd6a3",
  "#8b5cf6",
  "#e0c47d",
  "#f08080",
];

export const AXIS_LINE_COLOR = "#3f3f46";
export const AXIS_LABEL_COLOR = "#a1a1aa";
export const SPLIT_LINE_COLOR = "#27272a";

/** 段位下边界（与 rank.ts 区间一致），用于曲线段位水平参考线 */
export const DAN_BOUNDARIES: { value: number; label: string }[] = [
  { value: 1050, label: "段 1" },
  { value: 1200, label: "段 2" },
  { value: 1400, label: "段 3" },
  { value: 1600, label: "段 4" },
  { value: 1800, label: "段 5" },
  { value: 2000, label: "段 6" },
  { value: 2200, label: "段 7" },
  { value: 2400, label: "段 8" },
  { value: 2600, label: "段 9" },
];
