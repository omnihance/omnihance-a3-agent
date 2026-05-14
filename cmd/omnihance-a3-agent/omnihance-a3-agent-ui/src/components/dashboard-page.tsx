import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { Component, type ReactNode, useState } from 'react';
import {
  Cpu,
  ExternalLink,
  Loader2,
  MemoryStick,
  Globe,
  Activity,
  CheckCircle2,
  XCircle,
  Layers,
  Megaphone,
  type LucideIcon,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  type MetricCard,
  type ChartData,
  getMetricsSummary,
  getMetricsCharts,
} from '@/lib/api';
import { MetricChart } from '@/components/metric-chart';
import { APP_NAME, queryKeys } from '@/constants';
import { useStatus } from '@/hooks/use-status';

const METRIC_ICONS: Record<string, LucideIcon> = {
  cpu_usage_percentage: Cpu,
  memory_usage_percentage: MemoryStick,
  process_count: Layers,
};

const METRICS_TIME_WINDOWS = [
  { label: 'Last hour', value: '1h' },
  { label: 'Last 6 hours', value: '6h' },
  { label: 'Last 1 day', value: '1d' },
  { label: 'Last 7 days', value: '7d' },
] as const;

type MetricsWindowParams = {
  range?: string;
  from?: number;
  to?: number;
};

function DashboardPageContent() {
  const [metricsWindow, setMetricsWindow] = useState<MetricsWindowParams>({
    range: '1h',
  });

  const {
    status,
    isLoading: statusLoading,
    isError: statusError,
  } = useStatus();

  const latestVersion = status?.latest_version;
  const latestReleaseURL = status?.latest_release_url;
  const showVersionBanner =
    status?.new_version_available === true &&
    latestVersion !== null &&
    latestVersion !== undefined &&
    latestReleaseURL !== null &&
    latestReleaseURL !== undefined;
  const versionBannerMessage =
    status?.version?.toLowerCase() === 'dev'
      ? `Latest release ${latestVersion} is available. You are running a dev build.`
      : `New version ${latestVersion} is available. You are running ${status?.version ?? 'unknown'}.`;

  const { data: metricsSummary } = useQuery({
    queryKey: queryKeys.metricsSummary,
    queryFn: getMetricsSummary,
    enabled: status?.metrics_enabled ?? false,
    refetchInterval: 5000,
  });

  const { data: metricsCharts } = useQuery({
    queryKey: queryKeys.metricsCharts(metricsWindow),
    queryFn: () => getMetricsCharts(metricsWindow),
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
            </div>
            <p className="text-muted-foreground">
              {status?.version
                ? `Version ${status.version}`
                : 'Unknown Version'}
            </p>
          </div>
        </div>
      </div>

      {showVersionBanner && latestReleaseURL && (
        <Alert className="mb-6 border-amber-500/50 bg-amber-500/10 text-amber-950 dark:text-amber-100">
          <Megaphone className="h-4 w-4" />
          <AlertTitle>New version available</AlertTitle>
          <AlertDescription className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <span>{versionBannerMessage}</span>
            <Button asChild size="sm" variant="outline">
              <a
                href={latestReleaseURL}
                target="_blank"
                rel="noopener noreferrer"
                aria-label={`View ${latestVersion} release on GitHub`}
              >
                <ExternalLink className="h-4 w-4" />
                <span>View Release</span>
              </a>
            </Button>
          </AlertDescription>
        </Alert>
      )}

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
        <>
          <div className="mb-4 flex items-center justify-end">
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">Range</span>
              <Select
                value={metricsWindow.range ?? '1h'}
                onValueChange={(value) => {
                  setMetricsWindow({ range: value });
                }}
              >
                <SelectTrigger
                  className="w-[180px]"
                  aria-label="Select dashboard chart time range"
                >
                  <SelectValue placeholder="Select range" />
                </SelectTrigger>
                <SelectContent>
                  {METRICS_TIME_WINDOWS.map((windowRange) => (
                    <SelectItem
                      key={windowRange.value}
                      value={windowRange.value}
                    >
                      {windowRange.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
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
        </>
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
