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
    [1500, 1500, 8.00],
    [1900, 1500, 1.45],
    [1500, 1900, 14.55],
  ])("computeDelta(%i,%i) = %f", (winner, loser, expected) => {
    expect(computeDelta(winner, loser)).toBeCloseTo(expected, 2);
  });
});

describe("previewMatch", () => {
  it("self win is zero-sum", () => {
    const r = previewMatch(1500, 1500, "win");
    expect(r.self_delta).toBeCloseTo(8.00, 2);
    expect(r.opponent_delta).toBeCloseTo(-8.00, 2);
    expect(r.self_after).toBeCloseTo(1508.00, 2);
    expect(r.opponent_after).toBeCloseTo(1492.00, 2);
    expect(r.self_delta + r.opponent_delta).toBeCloseTo(0, 2);
  });

  it("self loss: opponent is winner", () => {
    const r = previewMatch(1500, 1500, "loss");
    expect(r.self_delta).toBeCloseTo(-8.00, 2);
    expect(r.opponent_delta).toBeCloseTo(8.00, 2);
    expect(r.self_after).toBeCloseTo(1492.00, 2);
    expect(r.opponent_after).toBeCloseTo(1508.00, 2);
  });

  it("upset win against stronger opponent gains more", () => {
    const r = previewMatch(1500, 1900, "win");
    expect(r.self_delta).toBeCloseTo(14.55, 2);
    expect(r.opponent_delta).toBeCloseTo(-14.55, 2);
  });
});
