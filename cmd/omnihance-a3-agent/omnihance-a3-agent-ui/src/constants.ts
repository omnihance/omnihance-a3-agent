export const APP_NAME = 'Omnihance A3 Agent';

export const APP_DESCRIPTION =
  'Omnihance A3 Agent is a platform for managing your A3 Online game servers.';

export const queryKeys = {
  status: ['status'] as const,
  session: ['session'] as const,
  metricsSummary: ['metrics-summary'] as const,
  metricsCharts: (params?: { range?: string; from?: number; to?: number }) => {
    if (params !== undefined) {
      return ['metrics-charts', params] as const;
    }

    return ['metrics-charts'] as const;
  },
  gameClientDataCounts: ['game-client-data-counts'] as const,
  monsters: ['monsters'] as const,
  maps: ['maps'] as const,
  items: ['items'] as const,
  fileTree: (path?: string, showDotfiles?: boolean) => {
    if (path !== undefined && showDotfiles !== undefined) {
      return ['file-tree', path, showDotfiles] as const;
    }
    if (path !== undefined) {
      return ['file-tree', path] as const;
    }
    return ['file-tree'] as const;
  },
  textFile: (path: string) => ['text-file', path] as const,
  npcFile: (path: string) => ['npc-file', path] as const,
  spawnFile: (path: string) => ['spawn-file', path] as const,
  dropFile: (path: string) => ['drop-file', path] as const,
  itemFile: (path: string) => ['item-file', path] as const,
  itemCombinationDataFile: (path: string) =>
    ['item-combination-data-file', path] as const,
  questFile: (path: string) => ['quest-file', path] as const,
  revisionSummary: (path: string) => ['revision-summary', path] as const,
  users: (page: number, pageSize: number, search?: string) => {
    if (search !== undefined && search !== '') {
      return ['users', page, pageSize, search] as const;
    }
    return ['users', page, pageSize] as const;
  },
  userStatuses: ['user-statuses'] as const,
  serverProcesses: ['server-processes'] as const,
  serverProcessStatus: (id: number) => ['server-process-status', id] as const,
  directoryShortcuts: ['directory-shortcuts'] as const,
  settings: ['settings'] as const,
  sqlServerOdbcDsns: ['sql-server-odbc-dsns'] as const,
  backupJobs: ['backup-jobs'] as const,
  backupJob: (id: number) => ['backup-job', id] as const,
  backupRuns: (jobId: number, page?: number, pageSize?: number) => {
    if (page !== undefined && pageSize !== undefined) {
      return ['backup-runs', jobId, page, pageSize] as const;
    }

    return ['backup-runs', jobId] as const;
  },
  backupRun: (runId: number) => ['backup-run', runId] as const,
  backupPathSearch: (query?: string, kind?: string) =>
    ['backup-path-search', query || '', kind || 'input'] as const,
  backupSqlServerDefaults: ['backup-sql-server-defaults'] as const,
  serverView: ['server-view'] as const,
  serverViewSyncStatus: ['server-view-sync-status'] as const,
  serverViewMainMaps: (query?: string) =>
    ['server-view-main-maps', query || ''] as const,
  serverViewZoneMaps: (query?: string) =>
    ['server-view-zone-maps', query || ''] as const,
  serverViewZoneSpawns: (mapQuery?: string, npcQuery?: string) =>
    ['server-view-zone-spawns', mapQuery || '', npcQuery || ''] as const,
  serverViewZoneSpawnDetails: (mapId: number, npcQuery?: string) =>
    ['server-view-zone-spawn-details', mapId, npcQuery || ''] as const,
  serverViewZoneDrops: (query?: string) =>
    ['server-view-zone-drops', query || ''] as const,
  serverViewZoneDropDetails: (npcId: number, query?: string) =>
    ['server-view-zone-drop-details', npcId, query || ''] as const,
  serverViewZoneShops: (query?: string) =>
    ['server-view-zone-shops', query || ''] as const,
  serverViewZoneShopDetails: (npcId: number, query?: string) =>
    ['server-view-zone-shop-details', npcId, query || ''] as const,
} as const;
