export const RANK_FLOOR = 1050.0;

/**
 * 与后端 domain.Dan 完全一致的段位映射。
 * rating < 1050 -> 0；否则 (rating-800)/200 向下取整，超过 9 截断为 9。
 */
export function danOf(rating: number): number {
  if (rating < RANK_FLOOR) {
    return 0;
  }
  const tier = Math.floor((rating - 800.0) / 200.0);
  if (tier > 9) {
    return 9;
  }
  return tier;
}

export function danLabel(dan: number): string {
  return dan === 0 ? "未定级" : `段 ${dan}`;
}

/** 段位徽章配色：段0灰 / 1-3蓝 / 4-6紫 / 7-8金 / 9红 */
export function danColor(dan: number): string {
  if (dan === 0) return "#9ca3af";
  if (dan <= 3) return "#4a9eff";
  if (dan <= 6) return "#8b5cf6";
  if (dan <= 8) return "#e0c47d";
  return "#f08080";
}
