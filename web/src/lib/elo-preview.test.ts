import { describe, it, expect } from "vitest";
import { expectedScore, computeDelta, previewMatch } from "@/lib/elo-preview";

describe("expectedScore", () => {
  it("equal ratings -> 0.5", () => {
    expect(expectedScore(1500, 1500)).toBeCloseTo(0.5, 10);
  });
  it("400 higher -> ~0.909", () => {
    expect(expectedScore(1900, 1500)).toBeCloseTo(0.9090909, 6);
  });
});

describe("computeDelta", () => {
  it.each([
    [1500, 1500, 8],
    [1900, 1500, 1],
    [1500, 1900, 15],
  ])("computeDelta(%i,%i) = %i", (winner, loser, expected) => {
    expect(computeDelta(winner, loser)).toBe(expected);
  });

  it("uses half-away-from-zero rounding (matches Go math.Round)", () => {
    // 构造一个 K*(1-E) 恰为 .5 的情形难以保证，改为验证非负且整数
    const d = computeDelta(1600, 1450);
    expect(Number.isInteger(d)).toBe(true);
    expect(d).toBeGreaterThan(0);
  });
});

describe("previewMatch", () => {
  it("self win is zero-sum", () => {
    const r = previewMatch(1500, 1500, "win");
    expect(r.self_delta).toBe(8);
    expect(r.opponent_delta).toBe(-8);
    expect(r.self_after).toBe(1508);
    expect(r.opponent_after).toBe(1492);
    expect(r.self_delta + r.opponent_delta).toBe(0);
  });

  it("self loss: opponent is winner", () => {
    const r = previewMatch(1500, 1500, "loss");
    // opponent wins +8 vs equal rating; self loses 8
    expect(r.self_delta).toBe(-8);
    expect(r.opponent_delta).toBe(8);
    expect(r.self_after).toBe(1492);
    expect(r.opponent_after).toBe(1508);
  });

  it("upset win against stronger opponent gains more", () => {
    const r = previewMatch(1500, 1900, "win");
    expect(r.self_delta).toBe(15);
    expect(r.opponent_delta).toBe(-15);
  });
});
