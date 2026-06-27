import { describe, it, expect } from "vitest";
import fs from "node:fs";
import path from "node:path";
import { danOf, danLabel, danColor } from "@/lib/rank";

function loadCases(): { rating: number; expected: number }[] {
  const csvPath = path.resolve(
    __dirname,
    "./__fixtures__/rank_cases.csv",
  );
  const raw = fs.readFileSync(csvPath, "utf-8").trim();
  const lines = raw.split(/\r?\n/);
  // skip header
  return lines.slice(1).map((line) => {
    const [rating, expected] = line.split(",");
    return { rating: Number(rating), expected: Number(expected) };
  });
}

describe("danOf", () => {
  const cases = loadCases();
  it("loads at least the documented boundary cases", () => {
    expect(cases.length).toBeGreaterThanOrEqual(14);
  });
  it.each(cases)(
    "rating $rating -> dan $expected",
    ({ rating, expected }) => {
      expect(danOf(rating)).toBe(expected);
    },
  );
});

describe("danLabel", () => {
  it("returns 未定级 for dan 0", () => {
    expect(danLabel(0)).toBe("未定级");
  });
  it.each([
    [1, "段 1"],
    [3, "段 3"],
    [9, "段 9"],
  ])("dan %i -> %s", (dan, label) => {
    expect(danLabel(dan)).toBe(label);
  });
});

describe("danColor", () => {
  it.each([
    [0, "#9ca3af"],
    [1, "#4a9eff"],
    [2, "#4a9eff"],
    [3, "#4a9eff"],
    [4, "#8b5cf6"],
    [5, "#8b5cf6"],
    [6, "#8b5cf6"],
    [7, "#e0c47d"],
    [8, "#e0c47d"],
    [9, "#f08080"],
  ])("dan %i -> %s", (dan, hex) => {
    expect(danColor(dan)).toBe(hex);
  });
});
