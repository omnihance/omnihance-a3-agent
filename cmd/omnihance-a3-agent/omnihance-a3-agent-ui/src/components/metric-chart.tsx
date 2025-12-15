import ReactECharts from 'echarts-for-react';
import type { ChartData } from '@/lib/api';

interface MetricChartProps {
  chartData: ChartData;
}

export function MetricChart({ chartData }: MetricChartProps) {
  return (
    <ReactECharts
      option={chartData.options}
      style={{ height: '200px', width: '100%' }}
      opts={{ renderer: 'canvas' }}
    />
  );
}
