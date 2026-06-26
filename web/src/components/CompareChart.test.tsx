import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

const optionSpy = vi.fn();
vi.mock("echarts-for-react", () => ({
  default: (props: { option: unknown }) => {
    optionSpy(props.option);
    return <div data-testid="echarts-mock" />;
  },
}));

import CompareChart from "@/components/CompareChart";

describe("CompareChart", () => {
  it("renders one line per series with axis-trigger tooltip", () => {
    render(
      <CompareChart
        series={[
          {
            username: "alice",
            color: "#4a9eff",
            points: [{ played_at: "2026-06-01T00:00:00Z", rating: 1500 }],
          },
          {
            username: "bob",
            color: "",
            points: [{ played_at: "2026-06-01T00:00:00Z", rating: 1480 }],
          },
        ]}
      />,
    );
    expect(screen.getByTestId("compare-chart")).toBeInTheDocument();
    const option = optionSpy.mock.calls.at(-1)?.[0] as {
      series: { lineStyle: { color: string } }[];
      tooltip: { trigger: string };
    };
    expect(option.series).toHaveLength(2);
    expect(option.tooltip.trigger).toBe("axis");
    // bob has empty color -> falls back to palette[1]
    expect(option.series[1].lineStyle.color).toBe("#7fd6a3");
  });
});
