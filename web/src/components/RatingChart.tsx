import ReactECharts from "echarts-for-react";
import type { EChartsOption } from "echarts";
import type { HistoryPoint } from "@/api/types";
import {
  ECHARTS_PALETTE,
  AXIS_LABEL_COLOR,
  AXIS_LINE_COLOR,
  SPLIT_LINE_COLOR,
  DAN_BOUNDARIES,
} from "@/lib/echarts-theme";

interface RatingChartProps {
  points: HistoryPoint[];
  height?: number;
}

export default function RatingChart({ points, height = 420 }: RatingChartProps) {
  const data = points.map((p) => [p.played_at, p.rating] as [string, number]);

  const option: EChartsOption = {
    backgroundColor: "transparent",
    color: ECHARTS_PALETTE,
    grid: { left: 48, right: 24, top: 24, bottom: 40 },
    tooltip: {
      trigger: "axis",
      backgroundColor: "#18181b",
      borderColor: "#3f3f46",
      textStyle: { color: "#fafafa" },
    },
    xAxis: {
      type: "time",
      axisLine: { lineStyle: { color: AXIS_LINE_COLOR } },
      axisLabel: { color: AXIS_LABEL_COLOR },
      splitLine: { show: false },
    },
    yAxis: {
      type: "value",
      scale: true,
      axisLine: { lineStyle: { color: AXIS_LINE_COLOR } },
      axisLabel: { color: AXIS_LABEL_COLOR },
      splitLine: { lineStyle: { color: SPLIT_LINE_COLOR } },
    },
    series: [
      {
        name: "等级分",
        type: "line",
        smooth: true,
        showSymbol: true,
        symbolSize: 5,
        data,
        lineStyle: { width: 2 },
        markLine: {
          silent: true,
          symbol: "none",
          lineStyle: { color: SPLIT_LINE_COLOR, type: "dashed" },
          label: {
            color: AXIS_LABEL_COLOR,
            formatter: "{b}",
            position: "insideEndTop",
          },
          data: DAN_BOUNDARIES.map((b) => ({ yAxis: b.value, name: b.label })),
        },
      },
    ],
  };

  return (
    <div data-testid="rating-chart">
      <ReactECharts
        option={option}
        style={{ height, width: "100%" }}
        notMerge
      />
    </div>
  );
}
