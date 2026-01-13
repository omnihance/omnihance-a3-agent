import { lazy, Suspense } from 'react';
import { Loader2 } from 'lucide-react';
import type { ChartData } from '@/lib/api';

const ReactECharts = lazy(() => import('echarts-for-react'));

interface MetricChartProps {
  chartData: ChartData;
}

function MetricChartContent({ chartData }: MetricChartProps) {
  return (
    <ReactECharts
      option={chartData.options}
      style={{ height: '200px', width: '100%' }}
      opts={{ renderer: 'canvas' }}
    />
  );
}

export function MetricChart({ chartData }: MetricChartProps) {
  return (
    <Suspense
      fallback={
        <div className="flex h-[200px] items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      }
    >
      <MetricChartContent chartData={chartData} />
    </Suspense>
  );
}
