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
} as const;
