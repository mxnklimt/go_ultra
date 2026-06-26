import ReactECharts from "echarts-for-react";
import type { EChartsOption } from "echarts";
import type { CompareSeries } from "@/api/types";
import {
  ECHARTS_PALETTE,
  AXIS_LABEL_COLOR,
  AXIS_LINE_COLOR,
  SPLIT_LINE_COLOR,
} from "@/lib/echarts-theme";

interface CompareChartProps {
  series: CompareSeries[];
  height?: number;
}

export default function CompareChart({ series, height = 480 }: CompareChartProps) {
  const echartsSeries = series.map((s, i) => ({
    name: s.username,
    type: "line" as const,
    smooth: true,
    showSymbol: false,
    data: s.points.map(
      (p) => [p.played_at, p.rating] as [string, number],
    ),
    lineStyle: {
      width: 2,
      color: s.color || ECHARTS_PALETTE[i % ECHARTS_PALETTE.length],
    },
    itemStyle: {
      color: s.color || ECHARTS_PALETTE[i % ECHARTS_PALETTE.length],
    },
  }));

  const option: EChartsOption = {
    backgroundColor: "transparent",
    legend: {
      data: series.map((s) => s.username),
      textStyle: { color: AXIS_LABEL_COLOR },
      top: 0,
    },
    grid: { left: 48, right: 24, top: 40, bottom: 40 },
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "line", snap: true },
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
    series: echartsSeries,
  };

  return (
    <div data-testid="compare-chart">
      <ReactECharts
        option={option}
        style={{ height, width: "100%" }}
        notMerge
      />
    </div>
  );
}
