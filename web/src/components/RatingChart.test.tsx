import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

const optionSpy = vi.fn();
vi.mock("echarts-for-react", () => ({
  default: (props: { option: unknown }) => {
    optionSpy(props.option);
    return <div data-testid="echarts-mock" />;
  },
}));

import RatingChart from "@/components/RatingChart";

describe("RatingChart", () => {
  it("renders a single line series with dan markLine", () => {
    render(
      <RatingChart
        points={[
          { played_at: "2026-06-01T00:00:00Z", rating: 1500 },
          { played_at: "2026-06-02T00:00:00Z", rating: 1516 },
        ]}
      />,
    );
    expect(screen.getByTestId("rating-chart")).toBeInTheDocument();
    const option = optionSpy.mock.calls.at(-1)?.[0] as {
      series: { type: string; data: unknown[]; markLine: { data: unknown[] } }[];
    };
    expect(option.series).toHaveLength(1);
    expect(option.series[0].type).toBe("line");
    expect(option.series[0].data).toHaveLength(2);
    expect(option.series[0].markLine.data.length).toBe(9);
  });
});
