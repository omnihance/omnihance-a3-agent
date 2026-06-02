import { Link } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type React from 'react';
import { useMemo, useState } from 'react';
import {
  AlertTriangle,
  ArrowLeft,
  Database,
  Loader2,
  Map as MapIcon,
  RefreshCw,
  Search,
  Server,
  Users,
} from 'lucide-react';
import { toast } from 'sonner';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { usePermissions } from '@/hooks/use-permissions';
import { queryKeys } from '@/constants';
import {
  APIError,
  APIValidationError,
  getServerView,
  getServerViewMainMaps,
  getServerViewZoneDropDetails,
  getServerViewZoneDrops,
  getServerViewZoneMaps,
  getServerViewZoneShopDetails,
  getServerViewZoneShops,
  getServerViewZoneSpawns,
  startServerViewSync,
  type ServerViewInfoSection,
  type ServerViewServerInfo,
  type ServerViewSpawnSummaryRow,
} from '@/lib/api';
import {
  formatExactDateTime,
  formatRelativeDateTime,
  formatStatusLabel,
  useDebouncedValue,
} from '@/lib/util';

export function ServerViewPage() {
  const queryClient = useQueryClient();
  const { hasPermission } = usePermissions();
  const canSync = hasPermission('manage_server');

  const {
    data,
    isLoading,
    isError,
    error,
    refetch: refetchOverview,
  } = useQuery({
    queryKey: queryKeys.serverView,
    queryFn: getServerView,
    refetchInterval: (query) => {
      const overview = query.state.data;
      return overview?.sync.running ? 3000 : false;
    },
  });

  const syncMutation = useMutation({
    mutationFn: startServerViewSync,
    onSuccess: async () => {
      toast.success('Server view sync started');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.serverView }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.serverViewSyncStatus,
        }),
      ]);
    },
    onError: (error) => {
      const message =
        error instanceof APIError
          ? error.getErrorMessage()
          : error instanceof Error
            ? error.message
            : 'Failed to start server view sync';
      toast.error(message);
    },
  });

  const latestRun = data?.sync.latest_run;
  const hasWarnings = (data?.sync.warnings.length || 0) > 0;

  return (
    <div className="space-y-6 p-4 lg:p-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Server View</h1>
          <p className="text-muted-foreground">
            Synced game server configuration and data overview
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {data?.sync.running && (
            <Badge variant="outline">
              <Loader2 className="h-3 w-3 animate-spin" />
              <span>Syncing</span>
            </Badge>
          )}
          {latestRun && (
            <span className="text-sm text-muted-foreground">
              Last sync {formatRelativeDateTime(latestRun.finished_at)}
            </span>
          )}
          {canSync && (
            <Button
              onClick={() => syncMutation.mutate()}
              disabled={syncMutation.isPending || data?.sync.running}
              aria-label="Sync server view data"
            >
              {syncMutation.isPending || data?.sync.running ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
              <span>Sync Data</span>
            </Button>
          )}
        </div>
      </div>

      {latestRun && (
        <div className="flex flex-wrap gap-2 text-sm text-muted-foreground">
          <Badge variant="outline">{formatStatusLabel(latestRun.status)}</Badge>
          <span>Started {formatExactDateTime(latestRun.started_at)}</span>
          {latestRun.finished_at && (
            <span>Finished {formatExactDateTime(latestRun.finished_at)}</span>
          )}
        </div>
      )}

      {hasWarnings && data && (
        <Alert className="border-amber-500/50 bg-amber-500/10 text-amber-950 dark:text-amber-100">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Sync completed with warnings</AlertTitle>
          <AlertDescription>
            {data.sync.warnings.slice(0, 5).map((warning) => (
              <div key={warning.id}>
                {warning.source}: {warning.message}
              </div>
            ))}
            {data.sync.warnings.length > 5 && (
              <div>{data.sync.warnings.length - 5} more warning(s)</div>
            )}
          </AlertDescription>
        </Alert>
      )}

      {isError ? (
        <Alert className="border-destructive/50 text-destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Server view unavailable</AlertTitle>
          <AlertDescription className="space-y-3">
            <div>{serverViewErrorMessage(error)}</div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
                void refetchOverview();
              }}
            >
              <RefreshCw className="h-4 w-4" />
              <span>Retry</span>
            </Button>
          </AlertDescription>
        </Alert>
      ) : isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <div className="grid gap-6 xl:grid-cols-2">
          {(data?.servers || []).map((server) => (
            <ServerInfoCard key={server.server_type} server={server} />
          ))}
        </div>
      )}
    </div>
  );
}

export function ServerViewMainMapsPage() {
  return <ServerViewMapsPage serverType="main" title="Main Server Maps" />;
}

export function ServerViewZoneMapsPage() {
  return <ServerViewMapsPage serverType="zone" title="Zone Server Maps" />;
}

export function ServerViewZoneSpawnsPage() {
  const [query, setQuery] = useState('');
  const debouncedQuery = useDebouncedValue(query, 250);

  const {
    data = [],
    isLoading,
    error,
  } = useQuery({
    queryKey: queryKeys.serverViewZoneSpawns('', ''),
    queryFn: () =>
      getServerViewZoneSpawns({
        mapQuery: '',
        npcQuery: '',
      }),
  });

  const mapRows = useMemo(
    () => buildSpawnMapRows(data, debouncedQuery),
    [data, debouncedQuery],
  );

  return (
    <ServerViewTablePage
      title="Monster Spawns"
      description="Spawn maps grouped from raw zone spawn rows"
      backHref="/server-view"
    >
      <SearchInput
        value={query}
        onChange={setQuery}
        placeholder="Search map or monster"
        ariaLabel="Search spawn maps by map or monster"
      />
      <DataTable
        loading={isLoading}
        error={error}
        emptyLabel="No spawn data synced"
        rowCount={mapRows.length}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Map Name</TableHead>
              <TableHead className="text-right">Map ID</TableHead>
              <TableHead className="text-right">Monsters</TableHead>
              <TableHead className="text-right">Total Spawns</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {mapRows.map((row) => (
              <TableRow key={row.map_id}>
                <TableCell className="font-medium">
                  <Link
                    to="/server-view/zone/spawns/$mapId"
                    params={{ mapId: String(row.map_id) }}
                    className="underline-offset-4 hover:underline"
                  >
                    {row.map_display}
                  </Link>
                </TableCell>
                <TableCell className="text-right">{row.map_id}</TableCell>
                <TableCell className="text-right">
                  {row.monster_count}
                </TableCell>
                <TableCell className="text-right">{row.total_count}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataTable>
    </ServerViewTablePage>
  );
}

export function ServerViewZoneSpawnDetailsPage({ mapId }: { mapId: number }) {
  const [query, setQuery] = useState('');
  const debouncedQuery = useDebouncedValue(query, 250);

  const {
    data = [],
    isLoading,
    error,
  } = useQuery({
    queryKey: queryKeys.serverViewZoneSpawnDetails(mapId, debouncedQuery),
    queryFn: () =>
      getServerViewZoneSpawns({
        mapQuery: String(mapId),
        npcQuery: debouncedQuery,
      }),
  });

  const rows = useMemo(
    () =>
      data
        .filter((row) => row.map_id === mapId)
        .sort((left, right) => left.npc_id - right.npc_id),
    [data, mapId],
  );
  const mapDisplay = rows[0]?.map_display || String(mapId);

  return (
    <ServerViewTablePage
      title={`Spawn Details: ${mapDisplay}`}
      description="Monster counts aggregated from raw spawn rows for this map"
      backHref="/server-view/zone/spawns"
    >
      <SearchInput
        value={query}
        onChange={setQuery}
        placeholder="Search monster name or ID"
        ariaLabel="Search map spawn details by monster"
      />
      <DataTable
        loading={isLoading}
        error={error}
        emptyLabel="No spawn rows found"
        rowCount={rows.length}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>NPC Name</TableHead>
              <TableHead className="text-right">NPC ID</TableHead>
              <TableHead className="text-right">Count</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={`${row.map_id}-${row.npc_id}`}>
                <TableCell className="font-medium">{row.npc_display}</TableCell>
                <TableCell className="text-right">{row.npc_id}</TableCell>
                <TableCell className="text-right">{row.count}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataTable>
    </ServerViewTablePage>
  );
}

export function ServerViewZoneDropsPage() {
  const [query, setQuery] = useState('');
  const debouncedQuery = useDebouncedValue(query, 250);
  const {
    data = [],
    isLoading,
    error,
  } = useQuery({
    queryKey: queryKeys.serverViewZoneDrops(debouncedQuery),
    queryFn: () => getServerViewZoneDrops(debouncedQuery),
  });

  return (
    <ServerViewTablePage
      title="Monster Drops"
      description="NPCs with synced drop rows; search also matches dropped item names and IDs"
      backHref="/server-view"
    >
      <SearchInput
        value={query}
        onChange={setQuery}
        placeholder="Search NPC or dropped item"
        ariaLabel="Search monster drops"
      />
      <DataTable
        loading={isLoading}
        error={error}
        emptyLabel="No drop data synced"
        rowCount={data.length}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>NPC Name</TableHead>
              <TableHead className="text-right">NPC ID</TableHead>
              <TableHead className="text-right">Drop Item Count</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((row) => (
              <TableRow key={row.npc_id}>
                <TableCell className="font-medium">
                  <Link
                    to="/server-view/zone/drops/$npcId"
                    params={{ npcId: String(row.npc_id) }}
                    className="underline-offset-4 hover:underline"
                  >
                    {row.npc_display}
                  </Link>
                </TableCell>
                <TableCell className="text-right">
                  <Link
                    to="/server-view/zone/drops/$npcId"
                    params={{ npcId: String(row.npc_id) }}
                    className="underline-offset-4 hover:underline"
                  >
                    {row.npc_id}
                  </Link>
                </TableCell>
                <TableCell className="text-right">
                  {row.drop_item_count}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataTable>
    </ServerViewTablePage>
  );
}

export function ServerViewZoneDropDetailsPage({ npcId }: { npcId: number }) {
  const [query, setQuery] = useState('');
  const debouncedQuery = useDebouncedValue(query, 250);
  const {
    data = [],
    isLoading,
    error,
  } = useQuery({
    queryKey: queryKeys.serverViewZoneDropDetails(npcId, debouncedQuery),
    queryFn: () => getServerViewZoneDropDetails(npcId, debouncedQuery),
  });

  return (
    <ServerViewTablePage
      title={`Drop Details: NPC ${npcId}`}
      description="Raw drop item rows transformed with uploaded item names"
      backHref="/server-view/zone/drops"
    >
      <SearchInput
        value={query}
        onChange={setQuery}
        placeholder="Search item name or ID"
        ariaLabel="Search drop details"
      />
      <DataTable
        loading={isLoading}
        error={error}
        emptyLabel="No drop rows found"
        rowCount={data.length}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Item Name</TableHead>
              <TableHead className="text-right">Item ID</TableHead>
              <TableHead className="text-right">Drop Rate</TableHead>
              <TableHead className="text-right">Group Code</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((row) => (
              <TableRow key={`${row.row_index}-${row.item_id}`}>
                <TableCell className="font-medium">
                  {row.item_display}
                </TableCell>
                <TableCell className="text-right">{row.item_id}</TableCell>
                <TableCell className="text-right">{row.drop_rate}</TableCell>
                <TableCell className="text-right">{row.group_code}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataTable>
    </ServerViewTablePage>
  );
}

export function ServerViewZoneShopsPage() {
  const [query, setQuery] = useState('');
  const debouncedQuery = useDebouncedValue(query, 250);
  const {
    data = [],
    isLoading,
    error,
  } = useQuery({
    queryKey: queryKeys.serverViewZoneShops(debouncedQuery),
    queryFn: () => getServerViewZoneShops(debouncedQuery),
  });

  return (
    <ServerViewTablePage
      title="Shops"
      description="NPC shop files synced from zone server shop data"
      backHref="/server-view"
    >
      <SearchInput
        value={query}
        onChange={setQuery}
        placeholder="Search NPC or shop item"
        ariaLabel="Search shops by NPC or item"
      />
      <DataTable
        loading={isLoading}
        error={error}
        emptyLabel="No shop data synced"
        rowCount={data.length}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>NPC Name</TableHead>
              <TableHead className="text-right">NPC ID</TableHead>
              <TableHead className="text-right">Item Count</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((row) => (
              <TableRow key={row.npc_id}>
                <TableCell className="font-medium">
                  <Link
                    to="/server-view/zone/shops/$npcId"
                    params={{ npcId: String(row.npc_id) }}
                    className="underline-offset-4 hover:underline"
                  >
                    {row.npc_display}
                  </Link>
                </TableCell>
                <TableCell className="text-right">
                  <Link
                    to="/server-view/zone/shops/$npcId"
                    params={{ npcId: String(row.npc_id) }}
                    className="underline-offset-4 hover:underline"
                  >
                    {row.npc_id}
                  </Link>
                </TableCell>
                <TableCell className="text-right">{row.item_count}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataTable>
    </ServerViewTablePage>
  );
}

export function ServerViewZoneShopDetailsPage({ npcId }: { npcId: number }) {
  const [query, setQuery] = useState('');
  const debouncedQuery = useDebouncedValue(query, 250);
  const {
    data = [],
    isLoading,
    error,
  } = useQuery({
    queryKey: queryKeys.serverViewZoneShopDetails(npcId, debouncedQuery),
    queryFn: () => getServerViewZoneShopDetails(npcId, debouncedQuery),
  });

  return (
    <ServerViewTablePage
      title={`Shop Details: NPC ${npcId}`}
      description="Shop item lines transformed with uploaded item names"
      backHref="/server-view/zone/shops"
    >
      <SearchInput
        value={query}
        onChange={setQuery}
        placeholder="Search item name or ID"
        ariaLabel="Search shop details"
      />
      <DataTable
        loading={isLoading}
        error={error}
        emptyLabel="No shop rows found"
        rowCount={data.length}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Item Name</TableHead>
              <TableHead className="text-right">Item ID</TableHead>
              <TableHead className="text-right">Line</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((row) => (
              <TableRow key={`${row.line_number}-${row.item_id}`}>
                <TableCell className="font-medium">
                  {row.item_display}
                </TableCell>
                <TableCell className="text-right">{row.item_id}</TableCell>
                <TableCell className="text-right">{row.line_number}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataTable>
    </ServerViewTablePage>
  );
}

function ServerViewMapsPage({
  serverType,
  title,
}: {
  serverType: 'main' | 'zone';
  title: string;
}) {
  const [query, setQuery] = useState('');
  const debouncedQuery = useDebouncedValue(query, 250);
  const queryKey =
    serverType === 'main'
      ? queryKeys.serverViewMainMaps(debouncedQuery)
      : queryKeys.serverViewZoneMaps(debouncedQuery);
  const queryFn =
    serverType === 'main'
      ? () => getServerViewMainMaps(debouncedQuery)
      : () => getServerViewZoneMaps(debouncedQuery);

  const { data = [], isLoading, error } = useQuery({ queryKey, queryFn });

  return (
    <ServerViewTablePage
      title={title}
      description="Map-to-zone assignments with uploaded map names when available"
      backHref="/server-view"
    >
      <SearchInput
        value={query}
        onChange={setQuery}
        placeholder="Search map name or ID"
        ariaLabel="Search maps"
      />
      <DataTable
        loading={isLoading}
        error={error}
        emptyLabel="No map data synced"
        rowCount={data.length}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Map Name</TableHead>
              <TableHead className="text-right">Map ID</TableHead>
              <TableHead className="text-right">Zone ID</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((row) => (
              <TableRow key={row.map_id}>
                <TableCell className="font-medium">{row.map_display}</TableCell>
                <TableCell className="text-right">{row.map_id}</TableCell>
                <TableCell className="text-right">{row.zone_id}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataTable>
    </ServerViewTablePage>
  );
}

function buildSpawnMapRows(
  rows: ServerViewSpawnSummaryRow[],
  query: string,
): ServerViewSpawnMapRow[] {
  const grouped = new Map<number, ServerViewSpawnMapAccumulator>();
  for (const row of rows) {
    if (!matchesSpawnMapSearch(row, query)) {
      continue;
    }

    const existing = grouped.get(row.map_id);
    if (existing) {
      existing.total_count += row.count;
      existing.monster_ids.add(row.npc_id);
      continue;
    }

    grouped.set(row.map_id, {
      map_id: row.map_id,
      map_display: row.map_display,
      total_count: row.count,
      monster_ids: new Set([row.npc_id]),
    });
  }

  return Array.from(grouped.values())
    .map((row) => ({
      map_id: row.map_id,
      map_display: row.map_display,
      monster_count: row.monster_ids.size,
      total_count: row.total_count,
    }))
    .sort((left, right) => left.map_id - right.map_id);
}

function matchesSpawnMapSearch(
  row: ServerViewSpawnSummaryRow,
  query: string,
): boolean {
  const normalizedQuery = query.trim().toLowerCase();
  if (normalizedQuery === '') {
    return true;
  }

  return (
    containsQuery(row.map_display, normalizedQuery) ||
    containsQuery(row.map_id, normalizedQuery) ||
    containsQuery(row.npc_display, normalizedQuery) ||
    containsQuery(row.npc_id, normalizedQuery)
  );
}

function containsQuery(value: string | number, query: string): boolean {
  return String(value).toLowerCase().includes(query);
}

function ServerInfoCard({ server }: { server: ServerViewServerInfo }) {
  const sections = useMemo(
    () => server.sections.filter((section) => section.rows.length > 0),
    [server.sections],
  );

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-4">
          <div>
            <CardTitle className="flex items-center gap-2">
              <ServerTypeIcon serverType={server.server_type} />
              {server.label} Info
            </CardTitle>
            <CardDescription className="break-all">
              {server.path || server.unavailable_reason || 'Not configured'}
            </CardDescription>
          </div>
          <Badge variant={server.configured ? 'outline' : 'secondary'}>
            {server.configured ? 'Configured' : 'Missing Path'}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {server.available_actions.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {server.available_actions.map((action) =>
              server.configured ? (
                <Button key={action.href} asChild variant="outline" size="sm">
                  <Link to={action.href}>{action.label}</Link>
                </Button>
              ) : (
                <Button
                  key={action.href}
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled
                  aria-disabled="true"
                  tabIndex={-1}
                >
                  {action.label}
                </Button>
              ),
            )}
          </div>
        )}
        {sections.length === 0 ? (
          <div className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
            {server.configured
              ? 'No synced SvrInfo rows yet.'
              : 'Configure this server path in Settings.'}
          </div>
        ) : (
          <div className="max-h-96 overflow-auto rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Section</TableHead>
                  <TableHead>Key</TableHead>
                  <TableHead>Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sections.flatMap((section) =>
                  section.rows.map((row, index) => (
                    <TableRow
                      key={`${section.name}-${row.key}-${row.value_index}-${index}`}
                    >
                      <TableCell className="text-muted-foreground">
                        {sectionLabel(section, index)}
                      </TableCell>
                      <TableCell className="font-medium">{row.key}</TableCell>
                      <TableCell className="break-all">{row.value}</TableCell>
                    </TableRow>
                  )),
                )}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ServerViewTablePage({
  title,
  description,
  backHref,
  children,
}: {
  title: string;
  description: string;
  backHref: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-6 p-4 lg:p-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <Button asChild variant="outline" size="sm" className="mb-3">
            <Link to={backHref}>
              <ArrowLeft className="h-4 w-4" />
              <span>Back</span>
            </Link>
          </Button>
          <h1 className="text-2xl font-bold tracking-tight">{title}</h1>
          <p className="text-muted-foreground">{description}</p>
        </div>
      </div>
      <Card>
        <CardContent className="space-y-4 pt-6">{children}</CardContent>
      </Card>
    </div>
  );
}

function DataTable({
  loading,
  error,
  emptyLabel,
  rowCount,
  children,
}: {
  loading: boolean;
  error?: unknown;
  emptyLabel: string;
  rowCount: number;
  children: React.ReactElement;
}) {
  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <Alert className="border-destructive/50 text-destructive">
        <AlertTriangle className="h-4 w-4" />
        <AlertTitle>Unable to load data</AlertTitle>
        <AlertDescription>{serverViewErrorMessage(error)}</AlertDescription>
      </Alert>
    );
  }

  if (rowCount === 0) {
    return (
      <div className="rounded-md border border-dashed p-8 text-center text-sm text-muted-foreground">
        {emptyLabel}
      </div>
    );
  }

  return <div className="overflow-x-auto rounded-md border">{children}</div>;
}

function SearchInput({
  value,
  onChange,
  placeholder,
  ariaLabel,
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  ariaLabel: string;
}) {
  return (
    <div className="relative">
      <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
      <Input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        aria-label={ariaLabel}
        className="pl-9"
      />
    </div>
  );
}

function ServerTypeIcon({ serverType }: { serverType: string }) {
  switch (serverType) {
    case 'main':
      return <Server className="h-5 w-5" />;
    case 'account':
      return <Users className="h-5 w-5" />;
    case 'zone':
      return <MapIcon className="h-5 w-5" />;
    case 'battle':
      return <Database className="h-5 w-5" />;
    default:
      return <Server className="h-5 w-5" />;
  }
}

function sectionLabel(section: ServerViewInfoSection, rowIndex: number) {
  return rowIndex === 0 ? section.name : '';
}

function serverViewErrorMessage(error: unknown): string {
  if (error instanceof APIError) {
    return error.getErrorMessage();
  }

  if (error instanceof APIValidationError) {
    return error.getValidationErrors().join('; ');
  }

  if (error instanceof Error) {
    return error.message;
  }

  return 'Failed to load server view data';
}

type ServerViewSpawnMapRow = {
  map_id: number;
  map_display: string;
  monster_count: number;
  total_count: number;
};

type ServerViewSpawnMapAccumulator = {
  map_id: number;
  map_display: string;
  total_count: number;
  monster_ids: Set<number>;
};
