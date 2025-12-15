import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { Component, type ReactNode } from 'react';
import {
  Cpu,
  Loader2,
  MemoryStick,
  Globe,
  Activity,
  CheckCircle2,
  XCircle,
  type LucideIcon,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  type MetricCard,
  type ChartData,
  getMetricsSummary,
  getMetricsCharts,
} from '@/lib/api';
import { MetricChart } from '@/components/metric-chart';
import { APP_NAME } from '@/constants';
import { useStatus } from '@/hooks/use-status';

const METRIC_ICONS: Record<string, LucideIcon> = {
  cpu_usage_percentage: Cpu,
  memory_usage_percentage: MemoryStick,
};

function DashboardPageContent() {
  const {
    status,
    isLoading: statusLoading,
    isError: statusError,
  } = useStatus();

  const { data: metricsSummary } = useQuery({
    queryKey: ['metrics-summary'],
    queryFn: getMetricsSummary,
    enabled: status?.metrics_enabled ?? false,
    refetchInterval: 5000,
  });

  const { data: metricsCharts } = useQuery({
    queryKey: ['metrics-charts'],
    queryFn: () => getMetricsCharts(),
    enabled: status?.metrics_enabled ?? false,
    refetchInterval: 5000,
  });

  return (
    <div className="p-4 lg:p-6">
      {/* Header */}
      <div className="mb-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight">
                {status?.name || APP_NAME}
              </h1>
              {status?.new_version_available && (
                <Badge variant="secondary">Update Available</Badge>
              )}
            </div>
            <p className="text-muted-foreground">
              {status?.version
                ? `Version ${status.version}`
                : 'Unknown Version'}
            </p>
          </div>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">
              {APP_NAME} Status
            </CardTitle>
            <Globe className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-sm font-medium break-all">
              {statusLoading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : status ? (
                status.name
              ) : (
                'Not available'
              )}
            </div>
            <div className="flex items-center gap-2 mt-1">
              <p className="text-xs text-muted-foreground">
                Version {status?.version || 'unknown'}
              </p>
              {status && (
                <Badge
                  variant="outline"
                  className={
                    statusLoading
                      ? 'border-muted-foreground/50'
                      : statusError
                        ? 'border-destructive bg-destructive/10 text-destructive'
                        : 'border-green-500 bg-green-500/10 text-green-600 dark:text-green-400'
                  }
                >
                  {statusLoading ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : statusError ? (
                    <>
                      <XCircle className="h-3 w-3" />
                      <span>Error</span>
                    </>
                  ) : (
                    <>
                      <CheckCircle2 className="h-3 w-3" />
                      <span>Online</span>
                    </>
                  )}
                </Badge>
              )}
            </div>
          </CardContent>
        </Card>
        {metricsSummary?.cards.map((card: MetricCard) => {
          const Icon = METRIC_ICONS[card.metric_name] || Activity;
          return (
            <Card key={card.metric_name}>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium">
                  {card.name}
                </CardTitle>
                <Icon className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-sm font-medium">{card.display_value}</div>
                <p className="text-xs text-muted-foreground mt-1">
                  {card.description}
                </p>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {/* Charts */}
      {metricsCharts?.charts && metricsCharts.charts.length > 0 && (
        <div className="mb-6 grid gap-4 lg:grid-cols-2">
          {metricsCharts.charts.map((chart: ChartData, index: number) => (
            <Card key={index}>
              <CardHeader>
                <CardTitle>{chart.title}</CardTitle>
              </CardHeader>
              <CardContent>
                <MetricChart chartData={chart} />
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

class DashboardPageErrorBoundary extends Component<
  { children: ReactNode },
  ErrorBoundaryState
> {
  constructor(props: { children: ReactNode }) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidUpdate(prevProps: { children: ReactNode }) {
    if (prevProps.children !== this.props.children) {
      this.setState({ hasError: false, error: null });
    }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex h-96 flex-col items-center justify-center">
          <h2 className="text-xl font-semibold">Project not found</h2>
          <Link to="/dashboard">
            <Button variant="link">Back to Dashboard</Button>
          </Link>
        </div>
      );
    }

    return this.props.children;
  }
}

export function DashboardPage() {
  return (
    <DashboardPageErrorBoundary>
      <DashboardPageContent />
    </DashboardPageErrorBoundary>
  );
}
