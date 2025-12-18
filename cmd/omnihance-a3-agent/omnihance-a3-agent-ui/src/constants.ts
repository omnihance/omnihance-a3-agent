export const APP_NAME = 'Omnihance A3 Agent';

export const APP_DESCRIPTION =
  'Omnihance A3 Agent is a platform for managing your A3 Online game servers.';

export const queryKeys = {
  status: ['status'] as const,
  session: ['session'] as const,
  metricsSummary: ['metrics-summary'] as const,
  metricsCharts: ['metrics-charts'] as const,
  monsters: ['monsters'] as const,
  maps: ['maps'] as const,
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
  revisionSummary: (path: string) => ['revision-summary', path] as const,
} as const;
