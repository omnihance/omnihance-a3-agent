import axios from 'axios';
import type { AxiosError } from 'axios';
import type { EChartsOption } from 'echarts';
import { z } from 'zod';

export const API_ROUTES = {
  AUTH_SIGN_IN: '/api/auth/sign-in',
  AUTH_SIGN_UP: '/api/auth/sign-up',
  SESSION: '/api/session',
  SESSION_SIGN_OUT: '/api/session/sign-out',
  SESSION_UPDATE_PASSWORD: '/api/session/update-password',
  STATUS: '/api/status',
  FILE_TREE: '/api/file-tree',
  NPC_FILE: '/api/file-tree/npc-file',
  TEXT_FILE: '/api/file-tree/text-file',
  SPAWN_FILE: '/api/file-tree/spawn-file',
  DROP_FILE: '/api/file-tree/drop-file',
  ITEM_FILE: '/api/file-tree/item-file',
  ITEM_COMBINATION_DATA_FILE: '/api/file-tree/item-combination-data',
  QUEST_FILE: '/api/file-tree/quest-file',
  REVERT_FILE: '/api/file-tree/revert-file',
  DUPLICATE_FILE: '/api/file-tree/duplicate-file',
  REVISION_COUNT: '/api/file-tree/revision-summary',
  METRICS_SUMMARY: '/api/metrics/summary',
  METRICS_CHARTS: '/api/metrics/charts',
  GAME_CLIENT_DATA_MONSTERS: '/api/game-client-data/monsters',
  GAME_CLIENT_DATA_COUNTS: '/api/game-client-data/counts',
  GAME_CLIENT_DATA_UPLOAD_MON_FILE: '/api/game-client-data/upload-mon-file',
  GAME_CLIENT_DATA_MAPS: '/api/game-client-data/maps',
  GAME_CLIENT_DATA_UPLOAD_MC_FILE: '/api/game-client-data/upload-mc-file',
  GAME_CLIENT_DATA_ITEMS: '/api/game-client-data/items',
  GAME_CLIENT_DATA_UPLOAD_IT0_FILE: '/api/game-client-data/upload-it0-file',
  GAME_CLIENT_DATA_UPLOAD_IT1_FILE: '/api/game-client-data/upload-it1-file',
  GAME_CLIENT_DATA_UPLOAD_IT2_FILE: '/api/game-client-data/upload-it2-file',
  GAME_CLIENT_DATA_UPLOAD_IT3_FILE: '/api/game-client-data/upload-it3-file',
  USERS: '/api/users',
  USER_STATUSES: '/api/users/statuses',
  USER_STATUS: (id: number) => `/api/users/${id}/status`,
  USER_PASSWORD: (id: number) => `/api/users/${id}/password`,
  SERVER_PROCESSES: '/api/server/processes',
  SERVER_PROCESS: (id: number) => `/api/server/processes/${id}`,
  SERVER_PROCESSES_REORDER: '/api/server/processes/reorder',
  SERVER_START: '/api/server/start',
  SERVER_STOP: '/api/server/stop',
  SERVER_PROCESS_START: (id: number) => `/api/server/processes/${id}/start`,
  SERVER_PROCESS_STOP: (id: number) => `/api/server/processes/${id}/stop`,
  SERVER_PROCESS_STATUS: (id: number) => `/api/server/processes/${id}/status`,
  DIRECTORY_SHORTCUTS: '/api/directory-shortcuts',
  DIRECTORY_SHORTCUT: (id: number) => `/api/directory-shortcuts/${id}`,
  SETTINGS: '/api/settings',
  SETTING: (key: string) => `/api/settings/${encodeURIComponent(key)}`,
  ODBC_SQLSERVER_DSNS: '/api/odbc/sqlserver-dsns',
  ODBC_SQLSERVER_DSN: (name: string) =>
    `/api/odbc/sqlserver-dsns/${encodeURIComponent(name)}`,
  ODBC_SQLSERVER_DEFAULT_DSNS: '/api/odbc/sqlserver-dsns/defaults',
  ODBC_SQLSERVER_DSN_TEST: '/api/odbc/sqlserver-dsns/test',
  BACKUP_JOBS: '/api/backups/jobs',
  BACKUP_JOB: (id: number) => `/api/backups/jobs/${id}`,
  BACKUP_JOB_RUN: (id: number) => `/api/backups/jobs/${id}/run`,
  BACKUP_JOB_CANCEL: (id: number) => `/api/backups/jobs/${id}/cancel`,
  BACKUP_JOB_RUNS: (id: number) => `/api/backups/jobs/${id}/runs`,
  BACKUP_RUN: (id: number) => `/api/backups/runs/${id}`,
  BACKUP_RUN_FILE_DOWNLOAD: (runId: number, fileId: number) =>
    `/api/backups/runs/${runId}/files/${fileId}/download`,
  BACKUP_PATH_SEARCH: '/api/backups/path-search',
  BACKUP_SQL_SERVER_DEFAULTS: '/api/backups/defaults/sql-server',
  SERVER_VIEW: '/api/server-view',
  SERVER_VIEW_SYNC_STATUS: '/api/server-view/sync/status',
  SERVER_VIEW_SYNC: '/api/server-view/sync',
  SERVER_VIEW_MAIN_MAPS: '/api/server-view/main/maps',
  SERVER_VIEW_ZONE_MAPS: '/api/server-view/zone/maps',
  SERVER_VIEW_ZONE_SPAWNS: '/api/server-view/zone/spawns',
  SERVER_VIEW_ZONE_DROPS: '/api/server-view/zone/drops',
  SERVER_VIEW_ZONE_DROP_DETAILS: (npcId: number) =>
    `/api/server-view/zone/drops/${npcId}`,
  SERVER_VIEW_ZONE_SHOPS: '/api/server-view/zone/shops',
  SERVER_VIEW_ZONE_SHOP_DETAILS: (npcId: number) =>
    `/api/server-view/zone/shops/${npcId}`,
} as const;

export class APIError extends Error {
  status: number;
  errorCode?: string;
  context?: string;
  errors?: string[];
  data?: ErrorResponse;

  constructor(message: string, status: number, data?: ErrorResponse) {
    super(message);
    this.name = 'APIError';
    this.status = status;
    this.errorCode = data?.errorCode;
    this.context = data?.context;
    this.errors = data?.errors;
    this.data = data;
  }

  getErrorMessage(): string {
    if (this.errors && this.errors.length > 0) {
      return this.errors[0];
    }

    return this.message;
  }

  getAllErrorMessages(): string[] {
    return this.errors || [this.message];
  }
}

export class APIValidationError extends Error {
  zodError: z.ZodError;
  responseData: unknown;

  constructor(message: string, zodError: z.ZodError, responseData: unknown) {
    super(message);
    this.name = 'APIValidationError';
    this.zodError = zodError;
    this.responseData = responseData;
  }

  getValidationErrors(): string[] {
    return this.zodError.issues.map((err: z.ZodIssue) => {
      const path = err.path.join('.');
      return path ? `${path}: ${err.message}` : err.message;
    });
  }
}

const ErrorResponseSchema = z.object({
  errorCode: z.string(),
  context: z.string(),
  errors: z.array(z.string()),
});

export type ErrorResponse = z.infer<typeof ErrorResponseSchema>;

const AuthRequestSchema = z.object({
  email: z.email(),
  password: z.string().min(6),
});

export type AuthRequest = z.infer<typeof AuthRequestSchema>;

const AuthResponseSchema = z.object({
  success: z.boolean(),
  message: z.string(),
});

export type AuthResponse = z.infer<typeof AuthResponseSchema>;

const GetSessionResponseSchema = z.object({
  session_id: z.string(),
  user_id: z.number().int(),
  email: z.email(),
  roles: z.array(z.string()),
  created_at: z.string(),
  expires_at: z.string(),
});

export type GetSessionResponse = z.infer<typeof GetSessionResponseSchema>;

const SignOutResponseSchema = z.object({
  success: z.boolean(),
  message: z.string(),
});

export type SignOutResponse = z.infer<typeof SignOutResponseSchema>;

const UpdatePasswordRequestSchema = z.object({
  current_password: z.string().min(1),
  new_password: z.string().min(6),
});

export type UpdatePasswordRequest = z.infer<typeof UpdatePasswordRequestSchema>;

const UpdatePasswordResponseSchema = z.object({
  success: z.boolean(),
  message: z.string(),
});

export type UpdatePasswordResponse = z.infer<
  typeof UpdatePasswordResponseSchema
>;

const StatusResponseSchema = z.object({
  name: z.string(),
  version: z.string(),
  setup_done: z.boolean(),
  new_version_available: z.boolean(),
  latest_version: z.string().nullable(),
  latest_release_url: z.string().nullable(),
  version_checked_at: z.string().nullable(),
  metrics_enabled: z.boolean(),
});

export type StatusResponse = z.infer<typeof StatusResponseSchema>;

const NPCAttackSchema = z.object({
  range: z.number().int().nonnegative(),
  area: z.number().int().nonnegative(),
  damage: z.number().int().nonnegative(),
  additional_damage: z.number().int().nonnegative(),
});

export type NPCAttack = z.infer<typeof NPCAttackSchema>;

const NPCFileAPIDataSchema = z.object({
  name: z.string(),
  id: z.number().int().nonnegative(),
  respawn_rate: z.number().int().nonnegative(),
  attack_type_info: z.number().int().min(0).max(255),
  target_selection_info: z.number().int().min(0).max(255),
  defense: z.number().int().min(0).max(255),
  additional_defense: z.number().int().min(0).max(255),
  attacks: z.array(NPCAttackSchema).length(3),
  attack_speed_low: z.number().int().nonnegative(),
  attack_speed_high: z.number().int().nonnegative(),
  movement_speed: z.number().int().nonnegative(),
  level: z.number().int().min(0).max(255),
  player_exp: z.number().int().nonnegative(),
  appearance: z.number().int().min(0).max(255),
  hp: z.number().int().nonnegative(),
  blue_attack_defense: z.number().int().nonnegative(),
  red_attack_defense: z.number().int().nonnegative(),
  grey_attack_defense: z.number().int().nonnegative(),
  mercenary_exp: z.number().int().nonnegative(),
});

export type NPCFileAPIData = z.infer<typeof NPCFileAPIDataSchema>;

type FileNode = {
  id: string;
  name: string;
  kind: 'directory' | 'file';
  depth: number;
  last_modified?: string;
  permissions?: string;
  file_size?: number;
  file_extension?: string;
  mime_type?: string;
  file_type?:
    | 'a3_npc_file'
    | 'a3_drop_file'
    | 'a3_it0_item_file'
    | 'a3_it0ex_item_file'
    | 'a3_it1_item_file'
    | 'a3_it2_item_file'
    | 'a3_it3_item_file'
    | 'a3_item_combination_data_file'
    | 'a3_map_file'
    | 'a3_spawn_file'
    | 'a3_unknown_file'
    | 'text_file'
    | 'a3_quest_file';
  is_editable: boolean;
  is_viewable: boolean;
  api_endpoint?: string;
  children?: FileNode[];
};

const FileNodeSchema: z.ZodType<FileNode> = z.lazy(() =>
  z.object({
    id: z.string(),
    name: z.string(),
    kind: z.enum(['directory', 'file']),
    depth: z.number().int(),
    last_modified: z.string().optional(),
    permissions: z.string().optional(),
    file_size: z.number().int().nonnegative().optional(),
    file_extension: z.string().optional(),
    mime_type: z.string().optional(),
    file_type: z
      .enum([
        'a3_npc_file',
        'a3_drop_file',
        'a3_it0_item_file',
        'a3_it0ex_item_file',
        'a3_it1_item_file',
        'a3_it2_item_file',
        'a3_it3_item_file',
        'a3_item_combination_data_file',
        'a3_map_file',
        'a3_spawn_file',
        'a3_quest_file',
        'a3_unknown_file',
        'text_file',
      ])
      .optional(),
    is_editable: z.boolean(),
    is_viewable: z.boolean(),
    api_endpoint: z.string().optional(),
    children: z.array(FileNodeSchema).optional(),
  }),
);

export type { FileNode };

const FileTreeResponseSchema = z.object({
  os: z.string(),
  file_tree: FileNodeSchema,
});

export type FileTreeResponse = z.infer<typeof FileTreeResponseSchema>;

export interface GetFileTreeParams {
  path?: string;
  show_dotfiles?: boolean;
}

export interface GetNPCFileParams {
  path: string;
}

export interface GetTextFileParams {
  path: string;
}

export interface GetSpawnFileParams {
  path: string;
}

export interface GetDropFileParams {
  path: string;
}

export interface GetItemFileParams {
  path: string;
}

export interface GetItemCombinationDataFileParams {
  path: string;
}

export interface GetQuestFileParams {
  path: string;
}

const UpdateFileResponseSchema = z.object({
  message: z.string(),
  revision_id: z.number().int(),
});

export type UpdateFileResponse = z.infer<typeof UpdateFileResponseSchema>;

const RevisionSummaryResponseSchema = z.object({
  count: z.number().int().nonnegative(),
  last_revision_at: z.number().int().nullable().optional(),
});

export type RevisionSummaryResponse = z.infer<
  typeof RevisionSummaryResponseSchema
>;

const DuplicateFileRequestSchema = z.object({
  source_path: z.string().min(1),
  new_file_name: z.string().min(1),
});

export type DuplicateFileRequest = z.infer<typeof DuplicateFileRequestSchema>;

const DuplicateFileResponseSchema = z.object({
  message: z.string(),
  duplicated_path: z.string(),
});

export type DuplicateFileResponse = z.infer<typeof DuplicateFileResponseSchema>;

const TextFileAPIDataSchema = z.object({
  content: z.string(),
});

export type TextFileAPIData = z.infer<typeof TextFileAPIDataSchema>;

const NPCSpawnAPIDataSchema = z.object({
  id: z.number().int().nonnegative(),
  x: z.number().int().min(0).max(255),
  y: z.number().int().min(0).max(255),
  unknown1: z.number().int().nonnegative(),
  orientation: z.number().int().min(0).max(255),
  spwan_step: z.number().int().min(0).max(255),
});

export type NPCSpawnAPIData = z.infer<typeof NPCSpawnAPIDataSchema>;

const SpawnFileAPIDataSchema = z.object({
  spawns: z.array(NPCSpawnAPIDataSchema),
});

export type SpawnFileAPIData = z.infer<typeof SpawnFileAPIDataSchema>;

const DropAPIDataSchema = z.object({
  item_code: z.number().int().min(0).max(65535),
  drop_rate: z.number().int().min(0).max(65535),
  drop_group: z.number().int().min(0).max(65535),
});

export type DropAPIData = z.infer<typeof DropAPIDataSchema>;

const DropFileAPIDataSchema = z.object({
  drops: z.array(DropAPIDataSchema),
});

export type DropFileAPIData = z.infer<typeof DropFileAPIDataSchema>;

const ItemFileTypeSchema = z.enum(['it0', 'it0ex', 'it1', 'it2', 'it3']);

const ItemFileLevelAPIDataSchema = z.object({
  level: z.number().int().min(0).max(255).optional(),
  additional_attribute: z.number().int().min(0).max(65535).optional(),
  strength: z.number().int().min(0).max(65535).optional(),
  dexterity: z.number().int().min(0).max(65535).optional(),
  intelligence: z.number().int().min(0).max(65535).optional(),
  attribute: z.number().int().min(0).max(65535).optional(),
  attribute_range: z.number().int().min(0).max(65535).optional(),
  blue_option: z.number().int().min(0).max(65535).optional(),
  red_option: z.number().int().min(0).max(65535).optional(),
  grey_option: z.number().int().min(0).max(65535).optional(),
});

export type ItemFileLevelAPIData = z.infer<typeof ItemFileLevelAPIDataSchema>;

const ItemFileItemAPIDataSchema = z.object({
  item_code_base: z.number().int().min(0).max(65535).optional(),
  row: z.number().int().min(0).max(65535).optional(),
  slot: z.number().int().min(0).max(65535).optional(),
  type: z.number().int().min(0).max(65535).optional(),
  item_code: z.number().int().nonnegative().optional(),
  name: z.string(),
  npc_price: z.number().int().nonnegative().optional(),
  required_level: z.number().int().min(0).max(65535).optional(),
  attribute: z.number().int().min(0).max(65535).optional(),
  blue_option: z.number().int().min(0).max(65535).optional(),
  red_option: z.number().int().min(0).max(65535).optional(),
  grey_option: z.number().int().min(0).max(65535).optional(),
  class: z.number().int().min(0).max(65535).optional(),
  skill_level: z.number().int().min(0).max(65535).optional(),
  levels: z.array(ItemFileLevelAPIDataSchema).optional(),
});

export type ItemFileItemAPIData = z.infer<typeof ItemFileItemAPIDataSchema>;

const ItemFileBaseItemAPIDataSchema = z.object({
  row: z.number().int().min(0).max(65535).optional(),
  item_code: z.number().int().nonnegative().optional(),
  name: z.string(),
});

export type ItemFileBaseItemAPIData = z.infer<
  typeof ItemFileBaseItemAPIDataSchema
>;

const ItemFileAPIDataSchema = z.object({
  item_file_type: ItemFileTypeSchema,
  items: z.array(ItemFileItemAPIDataSchema),
  base_items: z.array(ItemFileBaseItemAPIDataSchema).optional(),
});

export type ItemFileAPIData = z.infer<typeof ItemFileAPIDataSchema>;

const ItemCombinationUint16Schema = z.number().int().min(0).max(65535);

const ItemCombinationFormulaAPIDataSchema = z.object({
  ingredients: z.array(ItemCombinationUint16Schema).length(10),
  success_rate: ItemCombinationUint16Schema,
  outcome: ItemCombinationUint16Schema,
});

export type ItemCombinationFormulaAPIData = z.infer<
  typeof ItemCombinationFormulaAPIDataSchema
>;

const ItemCombinationDataFileAPIDataSchema = z.object({
  formulas: z.array(ItemCombinationFormulaAPIDataSchema),
});

export type ItemCombinationDataFileAPIData = z.infer<
  typeof ItemCombinationDataFileAPIDataSchema
>;

const ItemCombinationFormulaUpdateSchema =
  ItemCombinationFormulaAPIDataSchema.extend({
    success_rate: z.number().int().min(1).max(120),
  });

const ItemCombinationDataFileUpdateSchema = z.object({
  formulas: z.array(ItemCombinationFormulaUpdateSchema),
});

export const UNUSED_CONTINUATION = 0xffffffff;

const QuestLocationAPIDataSchema = z.object({
  x: z.number().int().min(0).max(255),
  y: z.number().int().min(0).max(255),
});

const ObjectiveAPIDataSchema = z.object({
  type: z.number().int().min(0).max(255),
  type_name: z.string().optional(),
  map_id: z.number().int().nonnegative(),
  location: QuestLocationAPIDataSchema,
  radius: z.number().int().min(0).max(255),
  target_id: z.number().int().nonnegative(),
  kill_count: z.number().int().nonnegative(),
  quest_item_id: z.number().int().min(0).max(65535),
  drop_items: z.array(z.number().int().min(0).max(65535).nullable()).length(3),
  required_item_count: z.number().int().nonnegative(),
  drop_probs: z.array(z.number().int().min(0).max(255)).length(3),
  name: z.string(),
  is_unused: z.boolean().optional(),
});

export type ObjectiveAPIData = z.infer<typeof ObjectiveAPIDataSchema>;

const QuestFileAPIDataSchema = z.object({
  quest_id: z.number().int().nonnegative(),
  giver_npc: z.number().int().nonnegative(),
  target_npc: z.number().int().nonnegative(),
  min_level: z.number().int().min(0).max(255),
  max_level: z.number().int().min(0).max(255),
  flags: z.number().int().nonnegative(),
  reward_items: z.array(z.number().int().nonnegative().nullable()).length(4),
  reward_counts: z.array(z.number().int().min(0).max(255).nullable()).length(4),
  exp_reward: z.number().int().nonnegative(),
  woonz_reward: z.number().int().nonnegative(),
  lore_reward: z.number().int().nonnegative(),
  objectives: z.array(ObjectiveAPIDataSchema).length(7),
  continuations: z.array(z.number().int().nonnegative().nullable()).length(3),
});

export type QuestFileAPIData = z.infer<typeof QuestFileAPIDataSchema>;

const MetricCardSchema = z.object({
  name: z.string(),
  metric_name: z.string(),
  description: z.string(),
  value: z.number(),
  display_value: z.string(),
});

export type MetricCard = z.infer<typeof MetricCardSchema>;

const MetricsSummaryResponseSchema = z.object({
  cards: z.array(MetricCardSchema),
});

export type MetricsSummaryResponse = z.infer<
  typeof MetricsSummaryResponseSchema
>;

const TimeRangeFilterSchema = z.object({
  key: z.string(),
  available_values: z.array(z.string()),
  default_value: z.string(),
});

export type TimeRangeFilter = z.infer<typeof TimeRangeFilterSchema>;

const ChartDataSchema = z.object({
  title: z.string(),
  metric_name: z.string(),
  options: z.any() as z.ZodType<EChartsOption>,
  filters: z.array(TimeRangeFilterSchema).optional(),
});

export type ChartData = z.infer<typeof ChartDataSchema>;

const MetricsChartsResponseSchema = z.object({
  charts: z.array(ChartDataSchema),
});

export type MetricsChartsResponse = z.infer<typeof MetricsChartsResponseSchema>;

const GameClientDataResponseSchema = z.object({
  id: z.number().int(),
  name: z.string(),
  item_type: z.enum(['it0', 'it1', 'it2', 'it3']).optional(),
});

export type GameClientDataResponse = z.infer<
  typeof GameClientDataResponseSchema
>;

const GameClientDataCountsResponseSchema = z.object({
  monsters: z.number().int().nonnegative(),
  maps: z.number().int().nonnegative(),
  items: z.object({
    it0: z.number().int().nonnegative(),
    it1: z.number().int().nonnegative(),
    it2: z.number().int().nonnegative(),
    it3: z.number().int().nonnegative(),
  }),
});

export type GameClientDataCountsResponse = z.infer<
  typeof GameClientDataCountsResponseSchema
>;

const UploadFileResponseSchema = z.object({
  message: z.string(),
  count: z.number().int(),
});

export type UploadFileResponse = z.infer<typeof UploadFileResponseSchema>;

const UserListItemSchema = z.object({
  id: z.number().int(),
  email: z.string().email(),
  roles: z.array(z.string()),
  status: z.enum(['pending', 'active', 'inactive', 'banned']),
  created_at: z.string(),
});

export type UserListItem = z.infer<typeof UserListItemSchema>;

const PaginationInfoSchema = z.object({
  totalCount: z.number().int().nonnegative(),
  page: z.number().int().positive(),
  pageSize: z.number().int().positive(),
});

export type PaginationInfo = z.infer<typeof PaginationInfoSchema>;

const ListUsersResponseSchema = z.object({
  data: z.array(UserListItemSchema),
  pagination: PaginationInfoSchema,
});

export type ListUsersResponse = z.infer<typeof ListUsersResponseSchema>;

const UpdateUserStatusRequestSchema = z.object({
  status: z.enum(['pending', 'active', 'inactive', 'banned']),
});

export type UpdateUserStatusRequest = z.infer<
  typeof UpdateUserStatusRequestSchema
>;

const UpdateUserStatusResponseSchema = z.object({
  success: z.boolean(),
  message: z.string(),
});

export type UpdateUserStatusResponse = z.infer<
  typeof UpdateUserStatusResponseSchema
>;

const SetUserPasswordRequestSchema = z.object({
  password: z.string().min(6),
});

export type SetUserPasswordRequest = z.infer<
  typeof SetUserPasswordRequestSchema
>;

const SetUserPasswordResponseSchema = z.object({
  success: z.boolean(),
  message: z.string(),
});

export type SetUserPasswordResponse = z.infer<
  typeof SetUserPasswordResponseSchema
>;

const GetUserStatusesResponseSchema = z.object({
  statuses: z.array(z.string()),
});

export type GetUserStatusesResponse = z.infer<
  typeof GetUserStatusesResponseSchema
>;

const axiosInstance = axios.create({
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
});

axiosInstance.interceptors.response.use(
  (response) => response,
  (error: AxiosError<unknown>) => {
    if (error.response) {
      const status = error.response.status;
      const errorData = error.response.data;

      const parsedError = ErrorResponseSchema.safeParse(errorData);

      if (parsedError.success) {
        const errorMessage =
          parsedError.data.errors?.[0] ||
          parsedError.data.errorCode ||
          error.message ||
          `Request failed with status ${status}`;

        return Promise.reject(
          new APIError(errorMessage, status, parsedError.data),
        );
      }

      const errorMessage =
        error.message || `Request failed with status ${status}`;

      return Promise.reject(new APIError(errorMessage, status));
    }

    if (error.request) {
      return Promise.reject(
        new APIError('Network error: Unable to reach the server', 0),
      );
    }

    return Promise.reject(
      new APIError(error.message || 'An unexpected error occurred', 0),
    );
  },
);

function validateResponse<T>(
  schema: z.ZodSchema<T>,
  data: unknown,
  endpoint: string,
): T {
  const result = schema.safeParse(data);

  if (!result.success) {
    console.log(result.error);
    throw new APIValidationError(
      `Response validation failed for ${endpoint}`,
      result.error,
      data,
    );
  }

  return result.data;
}

export async function signIn(data: AuthRequest): Promise<AuthResponse> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.AUTH_SIGN_IN,
    AuthRequestSchema.parse(data),
  );
  return validateResponse(
    AuthResponseSchema,
    response.data,
    API_ROUTES.AUTH_SIGN_IN,
  );
}

export async function signUp(data: AuthRequest): Promise<AuthResponse> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.AUTH_SIGN_UP,
    AuthRequestSchema.parse(data),
  );
  return validateResponse(
    AuthResponseSchema,
    response.data,
    API_ROUTES.AUTH_SIGN_UP,
  );
}

export async function getSession(): Promise<GetSessionResponse> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.SESSION);
  return validateResponse(
    GetSessionResponseSchema,
    response.data,
    API_ROUTES.SESSION,
  );
}

export async function signOut(): Promise<SignOutResponse> {
  const response = await axiosInstance.delete<unknown>(
    API_ROUTES.SESSION_SIGN_OUT,
  );
  return validateResponse(
    SignOutResponseSchema,
    response.data,
    API_ROUTES.SESSION_SIGN_OUT,
  );
}

export async function updatePassword(
  data: UpdatePasswordRequest,
): Promise<UpdatePasswordResponse> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.SESSION_UPDATE_PASSWORD,
    UpdatePasswordRequestSchema.parse(data),
  );
  return validateResponse(
    UpdatePasswordResponseSchema,
    response.data,
    API_ROUTES.SESSION_UPDATE_PASSWORD,
  );
}

export async function getStatus(): Promise<StatusResponse> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.STATUS);
  return validateResponse(
    StatusResponseSchema,
    response.data,
    API_ROUTES.STATUS,
  );
}

export async function getFileTree(
  params?: GetFileTreeParams,
): Promise<FileTreeResponse> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.FILE_TREE, {
    params,
  });
  return validateResponse(
    FileTreeResponseSchema,
    response.data,
    API_ROUTES.FILE_TREE,
  );
}

export async function getNPCFile(
  params: GetNPCFileParams,
): Promise<NPCFileAPIData> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.NPC_FILE, {
    params,
  });
  return validateResponse(
    NPCFileAPIDataSchema,
    response.data,
    API_ROUTES.NPC_FILE,
  );
}

export async function updateNPCFile(
  params: GetNPCFileParams,
  data: NPCFileAPIData,
): Promise<UpdateFileResponse> {
  const response = await axiosInstance.put<unknown>(
    API_ROUTES.NPC_FILE,
    NPCFileAPIDataSchema.parse(data),
    {
      params,
    },
  );
  return validateResponse(
    UpdateFileResponseSchema,
    response.data,
    API_ROUTES.NPC_FILE,
  );
}

export async function getTextFile(
  params: GetTextFileParams,
): Promise<TextFileAPIData> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.TEXT_FILE, {
    params,
  });
  return validateResponse(
    TextFileAPIDataSchema,
    response.data,
    API_ROUTES.TEXT_FILE,
  );
}

export async function updateTextFile(
  params: GetTextFileParams,
  data: TextFileAPIData,
): Promise<UpdateFileResponse> {
  const response = await axiosInstance.put<unknown>(
    API_ROUTES.TEXT_FILE,
    TextFileAPIDataSchema.parse(data),
    {
      params,
    },
  );
  return validateResponse(
    UpdateFileResponseSchema,
    response.data,
    API_ROUTES.TEXT_FILE,
  );
}

export async function getSpawnFile(
  params: GetSpawnFileParams,
): Promise<SpawnFileAPIData> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.SPAWN_FILE, {
    params,
  });
  return validateResponse(
    SpawnFileAPIDataSchema,
    response.data,
    API_ROUTES.SPAWN_FILE,
  );
}

export async function updateSpawnFile(
  params: GetSpawnFileParams,
  data: SpawnFileAPIData,
): Promise<UpdateFileResponse> {
  const response = await axiosInstance.put<unknown>(
    API_ROUTES.SPAWN_FILE,
    SpawnFileAPIDataSchema.parse(data),
    {
      params,
    },
  );
  return validateResponse(
    UpdateFileResponseSchema,
    response.data,
    API_ROUTES.SPAWN_FILE,
  );
}

export async function getDropFile(
  params: GetDropFileParams,
): Promise<DropFileAPIData> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.DROP_FILE, {
    params,
  });
  return validateResponse(
    DropFileAPIDataSchema,
    response.data,
    API_ROUTES.DROP_FILE,
  );
}

export async function updateDropFile(
  params: GetDropFileParams,
  data: DropFileAPIData,
): Promise<UpdateFileResponse> {
  const response = await axiosInstance.put<unknown>(
    API_ROUTES.DROP_FILE,
    DropFileAPIDataSchema.parse(data),
    {
      params,
    },
  );
  return validateResponse(
    UpdateFileResponseSchema,
    response.data,
    API_ROUTES.DROP_FILE,
  );
}

export async function getItemFile(
  params: GetItemFileParams,
): Promise<ItemFileAPIData> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.ITEM_FILE, {
    params,
  });
  return validateResponse(
    ItemFileAPIDataSchema,
    response.data,
    API_ROUTES.ITEM_FILE,
  );
}

export async function updateItemFile(
  params: GetItemFileParams,
  data: ItemFileAPIData,
): Promise<UpdateFileResponse> {
  const response = await axiosInstance.put<unknown>(
    API_ROUTES.ITEM_FILE,
    ItemFileAPIDataSchema.parse(data),
    {
      params,
    },
  );
  return validateResponse(
    UpdateFileResponseSchema,
    response.data,
    API_ROUTES.ITEM_FILE,
  );
}

export async function getItemCombinationDataFile(
  params: GetItemCombinationDataFileParams,
): Promise<ItemCombinationDataFileAPIData> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.ITEM_COMBINATION_DATA_FILE,
    {
      params,
    },
  );
  return validateResponse(
    ItemCombinationDataFileAPIDataSchema,
    response.data,
    API_ROUTES.ITEM_COMBINATION_DATA_FILE,
  );
}

export async function updateItemCombinationDataFile(
  params: GetItemCombinationDataFileParams,
  data: ItemCombinationDataFileAPIData,
): Promise<UpdateFileResponse> {
  const response = await axiosInstance.put<unknown>(
    API_ROUTES.ITEM_COMBINATION_DATA_FILE,
    ItemCombinationDataFileUpdateSchema.parse(data),
    {
      params,
    },
  );
  return validateResponse(
    UpdateFileResponseSchema,
    response.data,
    API_ROUTES.ITEM_COMBINATION_DATA_FILE,
  );
}

export async function getQuestFile(
  params: GetQuestFileParams,
): Promise<QuestFileAPIData> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.QUEST_FILE, {
    params,
  });
  return validateResponse(
    QuestFileAPIDataSchema,
    response.data,
    API_ROUTES.QUEST_FILE,
  );
}

export async function updateQuestFile(
  params: GetQuestFileParams,
  data: QuestFileAPIData,
): Promise<UpdateFileResponse> {
  const response = await axiosInstance.put<unknown>(
    API_ROUTES.QUEST_FILE,
    QuestFileAPIDataSchema.parse(data),
    {
      params,
    },
  );
  return validateResponse(
    UpdateFileResponseSchema,
    response.data,
    API_ROUTES.QUEST_FILE,
  );
}

export async function revertFile(
  params: GetTextFileParams,
): Promise<UpdateFileResponse> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.REVERT_FILE,
    undefined,
    {
      params,
    },
  );
  return validateResponse(
    UpdateFileResponseSchema,
    response.data,
    API_ROUTES.REVERT_FILE,
  );
}

export async function getRevisionSummary(
  params: GetTextFileParams,
): Promise<RevisionSummaryResponse> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.REVISION_COUNT, {
    params,
  });
  return validateResponse(
    RevisionSummaryResponseSchema,
    response.data,
    API_ROUTES.REVISION_COUNT,
  );
}

export async function duplicateFile(
  data: DuplicateFileRequest,
): Promise<DuplicateFileResponse> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.DUPLICATE_FILE,
    DuplicateFileRequestSchema.parse(data),
  );
  return validateResponse(
    DuplicateFileResponseSchema,
    response.data,
    API_ROUTES.DUPLICATE_FILE,
  );
}

export async function getMetricsSummary(): Promise<MetricsSummaryResponse> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.METRICS_SUMMARY);
  return validateResponse(
    MetricsSummaryResponseSchema,
    response.data,
    API_ROUTES.METRICS_SUMMARY,
  );
}

export async function getMetricsCharts(params?: {
  range?: string;
  from?: number;
  to?: number;
}): Promise<MetricsChartsResponse> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.METRICS_CHARTS, {
    params,
  });
  return validateResponse(
    MetricsChartsResponseSchema,
    response.data,
    API_ROUTES.METRICS_CHARTS,
  );
}

export async function getMonsters(params?: {
  s?: string;
}): Promise<GameClientDataResponse[]> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.GAME_CLIENT_DATA_MONSTERS,
    {
      params,
    },
  );
  return validateResponse(
    z.array(GameClientDataResponseSchema),
    response.data,
    API_ROUTES.GAME_CLIENT_DATA_MONSTERS,
  );
}

export async function getGameClientDataCounts(): Promise<GameClientDataCountsResponse> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.GAME_CLIENT_DATA_COUNTS,
  );
  return validateResponse(
    GameClientDataCountsResponseSchema,
    response.data,
    API_ROUTES.GAME_CLIENT_DATA_COUNTS,
  );
}

export async function uploadMonFile(file: File): Promise<UploadFileResponse> {
  const formData = new FormData();
  formData.append('file', file);

  const response = await axiosInstance.post<unknown>(
    API_ROUTES.GAME_CLIENT_DATA_UPLOAD_MON_FILE,
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    },
  );
  return validateResponse(
    UploadFileResponseSchema,
    response.data,
    API_ROUTES.GAME_CLIENT_DATA_UPLOAD_MON_FILE,
  );
}

export async function getMaps(params?: {
  s?: string;
}): Promise<GameClientDataResponse[]> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.GAME_CLIENT_DATA_MAPS,
    {
      params,
    },
  );
  return validateResponse(
    z.array(GameClientDataResponseSchema),
    response.data,
    API_ROUTES.GAME_CLIENT_DATA_MAPS,
  );
}

export async function uploadMcFile(file: File): Promise<UploadFileResponse> {
  const formData = new FormData();
  formData.append('file', file);

  const response = await axiosInstance.post<unknown>(
    API_ROUTES.GAME_CLIENT_DATA_UPLOAD_MC_FILE,
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    },
  );
  return validateResponse(
    UploadFileResponseSchema,
    response.data,
    API_ROUTES.GAME_CLIENT_DATA_UPLOAD_MC_FILE,
  );
}

export async function getItems(params?: {
  s?: string;
}): Promise<GameClientDataResponse[]> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.GAME_CLIENT_DATA_ITEMS,
    {
      params,
    },
  );
  return validateResponse(
    z.array(GameClientDataResponseSchema),
    response.data,
    API_ROUTES.GAME_CLIENT_DATA_ITEMS,
  );
}

async function uploadGameClientDataFile(
  endpoint: string,
  file: File,
): Promise<UploadFileResponse> {
  const formData = new FormData();
  formData.append('file', file);

  const response = await axiosInstance.post<unknown>(endpoint, formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
  return validateResponse(UploadFileResponseSchema, response.data, endpoint);
}

export async function uploadIt0File(file: File): Promise<UploadFileResponse> {
  return uploadGameClientDataFile(
    API_ROUTES.GAME_CLIENT_DATA_UPLOAD_IT0_FILE,
    file,
  );
}

export async function uploadIt1File(file: File): Promise<UploadFileResponse> {
  return uploadGameClientDataFile(
    API_ROUTES.GAME_CLIENT_DATA_UPLOAD_IT1_FILE,
    file,
  );
}

export async function uploadIt2File(file: File): Promise<UploadFileResponse> {
  return uploadGameClientDataFile(
    API_ROUTES.GAME_CLIENT_DATA_UPLOAD_IT2_FILE,
    file,
  );
}

export async function uploadIt3File(file: File): Promise<UploadFileResponse> {
  return uploadGameClientDataFile(
    API_ROUTES.GAME_CLIENT_DATA_UPLOAD_IT3_FILE,
    file,
  );
}

export async function getUsers(params?: {
  page?: number;
  pageSize?: number;
  s?: string;
}): Promise<ListUsersResponse> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.USERS, {
    params,
  });
  return validateResponse(
    ListUsersResponseSchema,
    response.data,
    API_ROUTES.USERS,
  );
}

export async function updateUserStatus(
  id: number,
  data: UpdateUserStatusRequest,
): Promise<UpdateUserStatusResponse> {
  const response = await axiosInstance.patch<unknown>(
    API_ROUTES.USER_STATUS(id),
    UpdateUserStatusRequestSchema.parse(data),
  );
  return validateResponse(
    UpdateUserStatusResponseSchema,
    response.data,
    API_ROUTES.USER_STATUS(id),
  );
}

export async function setUserPassword(
  id: number,
  data: SetUserPasswordRequest,
): Promise<SetUserPasswordResponse> {
  const response = await axiosInstance.patch<unknown>(
    API_ROUTES.USER_PASSWORD(id),
    SetUserPasswordRequestSchema.parse(data),
  );
  return validateResponse(
    SetUserPasswordResponseSchema,
    response.data,
    API_ROUTES.USER_PASSWORD(id),
  );
}

export async function getUserStatuses(): Promise<GetUserStatusesResponse> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.USER_STATUSES);
  return validateResponse(
    GetUserStatusesResponseSchema,
    response.data,
    API_ROUTES.USER_STATUSES,
  );
}

const ServerProcessSchema = z.object({
  id: z.number().int(),
  name: z.string(),
  path: z.string(),
  port: z.number().int().nullable(),
  sequence_order: z.number().int(),
  start_time: z.string().nullable(),
  end_time: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string().nullable(),
});

export type ServerProcess = z.infer<typeof ServerProcessSchema>;

const CreateServerProcessRequestSchema = z.object({
  name: z.string().min(1),
  path: z.string().min(1),
  port: z.number().int().positive().optional(),
});

export type CreateServerProcessRequest = z.infer<
  typeof CreateServerProcessRequestSchema
>;

const UpdateServerProcessRequestSchema = z.object({
  name: z.string().min(1),
  path: z.string().min(1),
  port: z.number().int().positive().optional(),
});

export type UpdateServerProcessRequest = z.infer<
  typeof UpdateServerProcessRequestSchema
>;

const ReorderUpdateSchema = z.object({
  id: z.number().int(),
  sequence_order: z.number().int(),
});

const ReorderServerProcessesRequestSchema = z.object({
  updates: z.array(ReorderUpdateSchema),
});

export type ReorderServerProcessesRequest = z.infer<
  typeof ReorderServerProcessesRequestSchema
>;

const ProcessStatusSchema = z.object({
  running: z.boolean(),
  port_open: z.boolean().optional(),
  start_time: z.string().optional(),
  end_time: z.string().optional(),
  current_uptime_seconds: z.number().int().optional(),
  last_uptime_seconds: z.number().int().optional(),
});

export type ProcessStatus = z.infer<typeof ProcessStatusSchema>;

const GetServerProcessesResponseSchema = z.object({
  processes: z.array(ServerProcessSchema),
});

export type GetServerProcessesResponse = z.infer<
  typeof GetServerProcessesResponseSchema
>;

const MessageResponseSchema = z.object({
  message: z.string(),
});

export type MessageResponse = z.infer<typeof MessageResponseSchema>;

const SettingKeySchema = z.enum([
  'DB_HOST',
  'DB_PORT',
  'DB_USER',
  'DB_PASS',
  'ZONE_SERVER_PATH',
  'ACCOUNT_SERVER_PATH',
  'MAIN_SERVER_PATH',
  'BATTLE_SERVER_PATH',
]);

export type SettingKey = z.infer<typeof SettingKeySchema>;

const SettingDefinitionSchema = z.object({
  key: SettingKeySchema,
  label: z.string(),
  description: z.string(),
  value_type: z.string(),
  input_type: z.enum(['text', 'number', 'password', 'directory']),
});

export type SettingDefinition = z.infer<typeof SettingDefinitionSchema>;

const SettingSchema = z.object({
  key: SettingKeySchema,
  value: z.string(),
  created_by: z.number().int().nullable(),
  created_at: z.string(),
  updated_by: z.number().int().nullable(),
  updated_at: z.string().nullable(),
});

export type Setting = z.infer<typeof SettingSchema>;

const SettingsResponseSchema = z.object({
  settings: z.array(SettingSchema),
  definitions: z.array(SettingDefinitionSchema),
});

export type SettingsResponse = z.infer<typeof SettingsResponseSchema>;

const CreateSettingRequestSchema = z.object({
  key: SettingKeySchema,
  value: z.string(),
});

export type CreateSettingRequest = z.infer<typeof CreateSettingRequestSchema>;

const UpdateSettingRequestSchema = z.object({
  value: z.string(),
});

export type UpdateSettingRequest = z.infer<typeof UpdateSettingRequestSchema>;

export async function getSettings(): Promise<SettingsResponse> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.SETTINGS);
  return validateResponse(
    SettingsResponseSchema,
    response.data,
    API_ROUTES.SETTINGS,
  );
}

export async function createSetting(
  data: CreateSettingRequest,
): Promise<Setting> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.SETTINGS,
    CreateSettingRequestSchema.parse(data),
  );
  return validateResponse(SettingSchema, response.data, API_ROUTES.SETTINGS);
}

export async function updateSetting(
  key: SettingKey,
  data: UpdateSettingRequest,
): Promise<Setting> {
  const route = API_ROUTES.SETTING(key);
  const response = await axiosInstance.put<unknown>(
    route,
    UpdateSettingRequestSchema.parse(data),
  );
  return validateResponse(SettingSchema, response.data, route);
}

export async function deleteSetting(key: SettingKey): Promise<MessageResponse> {
  const route = API_ROUTES.SETTING(key);
  const response = await axiosInstance.delete<unknown>(route);
  return validateResponse(MessageResponseSchema, response.data, route);
}

export async function getServerProcesses(): Promise<ServerProcess[]> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.SERVER_PROCESSES,
  );
  const data = validateResponse(
    GetServerProcessesResponseSchema,
    response.data,
    API_ROUTES.SERVER_PROCESSES,
  );
  return data.processes;
}

export async function createServerProcess(
  data: CreateServerProcessRequest,
): Promise<ServerProcess> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.SERVER_PROCESSES,
    CreateServerProcessRequestSchema.parse(data),
  );
  return validateResponse(
    ServerProcessSchema,
    response.data,
    API_ROUTES.SERVER_PROCESSES,
  );
}

export async function getServerProcess(id: number): Promise<ServerProcess> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.SERVER_PROCESS(id),
  );
  return validateResponse(
    ServerProcessSchema,
    response.data,
    API_ROUTES.SERVER_PROCESS(id),
  );
}

export async function updateServerProcess(
  id: number,
  data: UpdateServerProcessRequest,
): Promise<ServerProcess> {
  const response = await axiosInstance.put<unknown>(
    API_ROUTES.SERVER_PROCESS(id),
    UpdateServerProcessRequestSchema.parse(data),
  );
  return validateResponse(
    ServerProcessSchema,
    response.data,
    API_ROUTES.SERVER_PROCESS(id),
  );
}

export async function deleteServerProcess(
  id: number,
): Promise<MessageResponse> {
  const response = await axiosInstance.delete<unknown>(
    API_ROUTES.SERVER_PROCESS(id),
  );
  return validateResponse(
    MessageResponseSchema,
    response.data,
    API_ROUTES.SERVER_PROCESS(id),
  );
}

export async function reorderServerProcesses(
  data: ReorderServerProcessesRequest,
): Promise<MessageResponse> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.SERVER_PROCESSES_REORDER,
    ReorderServerProcessesRequestSchema.parse(data),
  );
  return validateResponse(
    MessageResponseSchema,
    response.data,
    API_ROUTES.SERVER_PROCESSES_REORDER,
  );
}

export async function startFullServer(): Promise<MessageResponse> {
  const response = await axiosInstance.post<unknown>(API_ROUTES.SERVER_START);
  return validateResponse(
    MessageResponseSchema,
    response.data,
    API_ROUTES.SERVER_START,
  );
}

export async function stopFullServer(): Promise<MessageResponse> {
  const response = await axiosInstance.post<unknown>(API_ROUTES.SERVER_STOP);
  return validateResponse(
    MessageResponseSchema,
    response.data,
    API_ROUTES.SERVER_STOP,
  );
}

export async function startProcess(id: number): Promise<MessageResponse> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.SERVER_PROCESS_START(id),
  );
  return validateResponse(
    MessageResponseSchema,
    response.data,
    API_ROUTES.SERVER_PROCESS_START(id),
  );
}

export async function stopProcess(id: number): Promise<MessageResponse> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.SERVER_PROCESS_STOP(id),
  );
  return validateResponse(
    MessageResponseSchema,
    response.data,
    API_ROUTES.SERVER_PROCESS_STOP(id),
  );
}

export async function getProcessStatus(id: number): Promise<ProcessStatus> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.SERVER_PROCESS_STATUS(id),
  );
  return validateResponse(
    ProcessStatusSchema,
    response.data,
    API_ROUTES.SERVER_PROCESS_STATUS(id),
  );
}

const DirectoryShortcutSchema = z.object({
  id: z.number().int(),
  user_id: z.number().int(),
  name: z.string(),
  path: z.string(),
  normalized_path: z.string(),
  created_at: z.string(),
  updated_at: z.string().nullable(),
});

export type DirectoryShortcut = z.infer<typeof DirectoryShortcutSchema>;

const DirectoryShortcutsResponseSchema = z.object({
  shortcuts: z.array(DirectoryShortcutSchema),
  limit: z.number().int(),
  over_limit_by: z.number().int(),
});

export type DirectoryShortcutsResponse = z.infer<
  typeof DirectoryShortcutsResponseSchema
>;

const CreateDirectoryShortcutRequestSchema = z.object({
  name: z.string().min(1),
  path: z.string().min(1),
});

export type CreateDirectoryShortcutRequest = z.infer<
  typeof CreateDirectoryShortcutRequestSchema
>;

const DeleteDirectoryShortcutResponseSchema = z.object({
  message: z.string(),
});

export type DeleteDirectoryShortcutResponse = z.infer<
  typeof DeleteDirectoryShortcutResponseSchema
>;

export async function getDirectoryShortcuts(): Promise<DirectoryShortcutsResponse> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.DIRECTORY_SHORTCUTS,
  );
  return validateResponse(
    DirectoryShortcutsResponseSchema,
    response.data,
    API_ROUTES.DIRECTORY_SHORTCUTS,
  );
}

export async function createDirectoryShortcut(
  data: CreateDirectoryShortcutRequest,
): Promise<DirectoryShortcut> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.DIRECTORY_SHORTCUTS,
    CreateDirectoryShortcutRequestSchema.parse(data),
  );
  return validateResponse(
    DirectoryShortcutSchema,
    response.data,
    API_ROUTES.DIRECTORY_SHORTCUTS,
  );
}

export async function deleteDirectoryShortcut(
  id: number,
): Promise<DeleteDirectoryShortcutResponse> {
  const response = await axiosInstance.delete<unknown>(
    API_ROUTES.DIRECTORY_SHORTCUT(id),
  );
  return validateResponse(
    DeleteDirectoryShortcutResponseSchema,
    response.data,
    API_ROUTES.DIRECTORY_SHORTCUT(id),
  );
}

const ServerViewInfoRowSchema = z.object({
  key: z.string(),
  value: z.string(),
  value_index: z.number().int(),
});

export type ServerViewInfoRow = z.infer<typeof ServerViewInfoRowSchema>;

const ServerViewInfoSectionSchema = z.object({
  name: z.string(),
  rows: z.array(ServerViewInfoRowSchema),
});

export type ServerViewInfoSection = z.infer<typeof ServerViewInfoSectionSchema>;

const ServerViewActionSchema = z.object({
  label: z.string(),
  href: z.string(),
});

export type ServerViewAction = z.infer<typeof ServerViewActionSchema>;

const ServerViewServerInfoSchema = z.object({
  server_type: z.string(),
  label: z.string(),
  setting_key: z.string(),
  configured: z.boolean(),
  path: z.string().nullable(),
  sections: z.array(ServerViewInfoSectionSchema),
  available_actions: z.array(ServerViewActionSchema),
  unavailable_reason: z.string().nullable(),
});

export type ServerViewServerInfo = z.infer<typeof ServerViewServerInfoSchema>;

const ServerViewSyncRunSchema = z.object({
  id: z.number().int(),
  status: z.string(),
  started_at: z.string(),
  finished_at: z.string().nullable(),
  warning_count: z.number().int(),
  error_details: z.string().nullable(),
  created_by: z.number().int().nullable(),
  created_at: z.string(),
  updated_at: z.string().nullable(),
});

export type ServerViewSyncRun = z.infer<typeof ServerViewSyncRunSchema>;

const ServerViewSyncWarningSchema = z.object({
  id: z.number().int(),
  run_id: z.number().int(),
  source: z.string(),
  message: z.string(),
  created_at: z.string(),
});

export type ServerViewSyncWarning = z.infer<typeof ServerViewSyncWarningSchema>;

const ServerViewMissingSettingSchema = z.object({
  server_type: z.string(),
  setting_key: z.string(),
  label: z.string(),
});

const ServerViewConfiguredSettingSchema = z.object({
  server_type: z.string(),
  setting_key: z.string(),
  label: z.string(),
  path: z.string(),
});

const ServerViewSyncStatusSchema = z.object({
  running: z.boolean(),
  latest_run: ServerViewSyncRunSchema.nullable(),
  warnings: z.array(ServerViewSyncWarningSchema),
  missing_settings: z.array(ServerViewMissingSettingSchema),
  configured_settings: z.array(ServerViewConfiguredSettingSchema),
});

export type ServerViewSyncStatus = z.infer<typeof ServerViewSyncStatusSchema>;

const ServerViewOverviewSchema = z.object({
  servers: z.array(ServerViewServerInfoSchema),
  sync: ServerViewSyncStatusSchema,
});

export type ServerViewOverview = z.infer<typeof ServerViewOverviewSchema>;

const ServerViewMapRowSchema = z.object({
  map_id: z.number().int(),
  map_name: z.string().nullable(),
  map_display: z.string(),
  zone_id: z.number().int(),
});

export type ServerViewMapRow = z.infer<typeof ServerViewMapRowSchema>;

const ServerViewMapsResponseSchema = z.object({
  maps: z.array(ServerViewMapRowSchema),
});

const ServerViewSpawnSummaryRowSchema = z.object({
  map_id: z.number().int(),
  map_name: z.string().nullable(),
  map_display: z.string(),
  npc_id: z.number().int(),
  npc_name: z.string().nullable(),
  npc_display: z.string(),
  count: z.number().int(),
});

export type ServerViewSpawnSummaryRow = z.infer<
  typeof ServerViewSpawnSummaryRowSchema
>;

const ServerViewSpawnsResponseSchema = z.object({
  spawns: z.array(ServerViewSpawnSummaryRowSchema),
});

const ServerViewDropSummaryRowSchema = z.object({
  npc_id: z.number().int(),
  npc_name: z.string().nullable(),
  npc_display: z.string(),
  drop_item_count: z.number().int(),
});

export type ServerViewDropSummaryRow = z.infer<
  typeof ServerViewDropSummaryRowSchema
>;

const ServerViewDropsResponseSchema = z.object({
  drops: z.array(ServerViewDropSummaryRowSchema),
});

const ServerViewDropDetailRowSchema = z.object({
  row_index: z.number().int(),
  item_id: z.number().int(),
  item_name: z.string().nullable(),
  item_display: z.string(),
  drop_rate: z.number().int(),
  group_code: z.number().int(),
});

export type ServerViewDropDetailRow = z.infer<
  typeof ServerViewDropDetailRowSchema
>;

const ServerViewDropDetailsResponseSchema = z.object({
  drops: z.array(ServerViewDropDetailRowSchema),
});

const ServerViewShopSummaryRowSchema = z.object({
  npc_id: z.number().int(),
  npc_name: z.string().nullable(),
  npc_display: z.string(),
  item_count: z.number().int(),
});

export type ServerViewShopSummaryRow = z.infer<
  typeof ServerViewShopSummaryRowSchema
>;

const ServerViewShopsResponseSchema = z.object({
  shops: z.array(ServerViewShopSummaryRowSchema),
});

const ServerViewShopDetailRowSchema = z.object({
  line_number: z.number().int(),
  item_id: z.number().int(),
  item_name: z.string().nullable(),
  item_display: z.string(),
});

export type ServerViewShopDetailRow = z.infer<
  typeof ServerViewShopDetailRowSchema
>;

const ServerViewShopDetailsResponseSchema = z.object({
  items: z.array(ServerViewShopDetailRowSchema),
});

export async function getServerView(): Promise<ServerViewOverview> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.SERVER_VIEW);
  return validateResponse(
    ServerViewOverviewSchema,
    response.data,
    API_ROUTES.SERVER_VIEW,
  );
}

export async function getServerViewSyncStatus(): Promise<ServerViewSyncStatus> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.SERVER_VIEW_SYNC_STATUS,
  );
  return validateResponse(
    ServerViewSyncStatusSchema,
    response.data,
    API_ROUTES.SERVER_VIEW_SYNC_STATUS,
  );
}

export async function startServerViewSync(): Promise<ServerViewSyncRun> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.SERVER_VIEW_SYNC,
  );
  return validateResponse(
    ServerViewSyncRunSchema,
    response.data,
    API_ROUTES.SERVER_VIEW_SYNC,
  );
}

export async function getServerViewMainMaps(
  query?: string,
): Promise<ServerViewMapRow[]> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.SERVER_VIEW_MAIN_MAPS,
    { params: query ? { q: query } : undefined },
  );
  const data = validateResponse(
    ServerViewMapsResponseSchema,
    response.data,
    API_ROUTES.SERVER_VIEW_MAIN_MAPS,
  );
  return data.maps;
}

export async function getServerViewZoneMaps(
  query?: string,
): Promise<ServerViewMapRow[]> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.SERVER_VIEW_ZONE_MAPS,
    { params: query ? { q: query } : undefined },
  );
  const data = validateResponse(
    ServerViewMapsResponseSchema,
    response.data,
    API_ROUTES.SERVER_VIEW_ZONE_MAPS,
  );
  return data.maps;
}

export async function getServerViewZoneSpawns(params?: {
  mapQuery?: string;
  npcQuery?: string;
}): Promise<ServerViewSpawnSummaryRow[]> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.SERVER_VIEW_ZONE_SPAWNS,
    {
      params: {
        map_q: params?.mapQuery || undefined,
        npc_q: params?.npcQuery || undefined,
      },
    },
  );
  const data = validateResponse(
    ServerViewSpawnsResponseSchema,
    response.data,
    API_ROUTES.SERVER_VIEW_ZONE_SPAWNS,
  );
  return data.spawns;
}

export async function getServerViewZoneDrops(
  query?: string,
): Promise<ServerViewDropSummaryRow[]> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.SERVER_VIEW_ZONE_DROPS,
    { params: query ? { q: query } : undefined },
  );
  const data = validateResponse(
    ServerViewDropsResponseSchema,
    response.data,
    API_ROUTES.SERVER_VIEW_ZONE_DROPS,
  );
  return data.drops;
}

export async function getServerViewZoneDropDetails(
  npcId: number,
  query?: string,
): Promise<ServerViewDropDetailRow[]> {
  const route = API_ROUTES.SERVER_VIEW_ZONE_DROP_DETAILS(npcId);
  const response = await axiosInstance.get<unknown>(route, {
    params: query ? { q: query } : undefined,
  });
  const data = validateResponse(
    ServerViewDropDetailsResponseSchema,
    response.data,
    route,
  );
  return data.drops;
}

export async function getServerViewZoneShops(
  query?: string,
): Promise<ServerViewShopSummaryRow[]> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.SERVER_VIEW_ZONE_SHOPS,
    { params: query ? { q: query } : undefined },
  );
  const data = validateResponse(
    ServerViewShopsResponseSchema,
    response.data,
    API_ROUTES.SERVER_VIEW_ZONE_SHOPS,
  );
  return data.shops;
}

export async function getServerViewZoneShopDetails(
  npcId: number,
  query?: string,
): Promise<ServerViewShopDetailRow[]> {
  const route = API_ROUTES.SERVER_VIEW_ZONE_SHOP_DETAILS(npcId);
  const response = await axiosInstance.get<unknown>(route, {
    params: query ? { q: query } : undefined,
  });
  const data = validateResponse(
    ServerViewShopDetailsResponseSchema,
    response.data,
    route,
  );
  return data.items;
}

const BackupJobTypeSchema = z.enum(['file', 'sql_server']);

export type BackupJobType = z.infer<typeof BackupJobTypeSchema>;

const BackupJobSchema = z.object({
  id: z.number().int(),
  job_type: BackupJobTypeSchema,
  name: z.string(),
  status: z.string(),
  cron_expression: z.string().nullable(),
  destination_directory: z.string(),
  archive_password: z.string().nullable(),
  source_path: z.string().nullable(),
  sql_host: z.string().nullable(),
  sql_port: z.number().int().nullable(),
  sql_username: z.string().nullable(),
  sql_password: z.string().nullable(),
  sql_database_names: z.string().nullable(),
  last_run_at: z.string().nullable(),
  created_by: z.number().int().nullable(),
  created_at: z.string(),
  updated_by: z.number().int().nullable(),
  updated_at: z.string().nullable(),
  deleted_by: z.number().int().nullable(),
  deleted_at: z.string().nullable(),
});

export type BackupJob = z.infer<typeof BackupJobSchema>;

const BackupRunSchema = z.object({
  id: z.number().int(),
  job_id: z.number().int(),
  trigger_type: z.string(),
  status: z.string(),
  previous_job_status: z.string(),
  started_at: z.string(),
  finished_at: z.string().nullable(),
  cancel_requested_at: z.string().nullable(),
  output: z.string().nullable(),
  error_details: z.string().nullable(),
  created_by: z.number().int().nullable(),
  created_at: z.string(),
  updated_at: z.string().nullable(),
});

export type BackupRun = z.infer<typeof BackupRunSchema>;

const BackupRunFileSchema = z.object({
  id: z.number().int(),
  run_id: z.number().int(),
  item_name: z.string(),
  file_path: z.string(),
  file_size: z.number().int(),
  created_at: z.string(),
});

export type BackupRunFile = z.infer<typeof BackupRunFileSchema>;

const BackupRunDetailsSchema = z.object({
  run: BackupRunSchema,
  files: z.array(BackupRunFileSchema),
});

export type BackupRunDetails = z.infer<typeof BackupRunDetailsSchema>;

const BackupJobsResponseSchema = z.object({
  jobs: z.array(BackupJobSchema),
});

const BackupRunsResponseSchema = z.object({
  runs: z.array(BackupRunSchema),
  pagination: PaginationInfoSchema,
});

export type BackupRunsResponse = z.infer<typeof BackupRunsResponseSchema>;

const BackupJobRequestSchema = z.object({
  job_type: BackupJobTypeSchema,
  name: z.string().min(1),
  status: z.string().min(1),
  cron_expression: z.string().nullable().optional(),
  destination_directory: z.string().min(1),
  archive_password: z.string().nullable().optional(),
  source_path: z.string().nullable().optional(),
  sql_host: z.string().nullable().optional(),
  sql_port: z.number().int().positive().nullable().optional(),
  sql_username: z.string().nullable().optional(),
  sql_password: z.string().nullable().optional(),
  sql_database_names: z.string().nullable().optional(),
});

export type BackupJobRequest = z.infer<typeof BackupJobRequestSchema>;

const BackupPathSearchResultSchema = z.object({
  name: z.string(),
  path: z.string(),
  kind: z.enum(['file', 'directory']),
});

export type BackupPathSearchResult = z.infer<
  typeof BackupPathSearchResultSchema
>;

const BackupPathSearchResponseSchema = z.object({
  results: z.array(BackupPathSearchResultSchema),
});

const BackupSQLServerDefaultsSchema = z.object({
  host: z.string(),
  port: z.number().int(),
  username: z.string(),
  password: z.string(),
  local_server_running: z.boolean(),
});

export type BackupSQLServerDefaults = z.infer<
  typeof BackupSQLServerDefaultsSchema
>;

export async function getBackupJobs(): Promise<BackupJob[]> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.BACKUP_JOBS);
  const data = validateResponse(
    BackupJobsResponseSchema,
    response.data,
    API_ROUTES.BACKUP_JOBS,
  );
  return data.jobs;
}

export async function getBackupJob(id: number): Promise<BackupJob> {
  const route = API_ROUTES.BACKUP_JOB(id);
  const response = await axiosInstance.get<unknown>(route);
  return validateResponse(BackupJobSchema, response.data, route);
}

export async function createBackupJob(
  payload: BackupJobRequest,
): Promise<BackupJob> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.BACKUP_JOBS,
    BackupJobRequestSchema.parse(payload),
  );
  return validateResponse(
    BackupJobSchema,
    response.data,
    API_ROUTES.BACKUP_JOBS,
  );
}

export async function updateBackupJob(
  id: number,
  payload: BackupJobRequest,
): Promise<BackupJob> {
  const route = API_ROUTES.BACKUP_JOB(id);
  const response = await axiosInstance.put<unknown>(
    route,
    BackupJobRequestSchema.parse(payload),
  );
  return validateResponse(BackupJobSchema, response.data, route);
}

export async function deleteBackupJob(id: number): Promise<MessageResponse> {
  const route = API_ROUTES.BACKUP_JOB(id);
  const response = await axiosInstance.delete<unknown>(route);
  return validateResponse(MessageResponseSchema, response.data, route);
}

export async function runBackupJob(id: number): Promise<BackupRun> {
  const route = API_ROUTES.BACKUP_JOB_RUN(id);
  const response = await axiosInstance.post<unknown>(route);
  return validateResponse(BackupRunSchema, response.data, route);
}

export async function cancelBackupJob(id: number): Promise<BackupRun> {
  const route = API_ROUTES.BACKUP_JOB_CANCEL(id);
  const response = await axiosInstance.post<unknown>(route);
  return validateResponse(BackupRunSchema, response.data, route);
}

export async function getBackupRuns(
  id: number,
  params?: { page?: number; pageSize?: number },
): Promise<BackupRunsResponse> {
  const route = API_ROUTES.BACKUP_JOB_RUNS(id);
  const response = await axiosInstance.get<unknown>(route, { params });
  return validateResponse(BackupRunsResponseSchema, response.data, route);
}

export async function getBackupRunDetails(
  id: number,
): Promise<BackupRunDetails> {
  const route = API_ROUTES.BACKUP_RUN(id);
  const response = await axiosInstance.get<unknown>(route);
  return validateResponse(BackupRunDetailsSchema, response.data, route);
}

export function getBackupRunFileDownloadUrl(
  runId: number,
  fileId: number,
): string {
  return API_ROUTES.BACKUP_RUN_FILE_DOWNLOAD(runId, fileId);
}

export async function searchBackupPaths(params: {
  query?: string;
  kind?: 'input' | 'directory';
}): Promise<BackupPathSearchResult[]> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.BACKUP_PATH_SEARCH,
    { params },
  );
  const data = validateResponse(
    BackupPathSearchResponseSchema,
    response.data,
    API_ROUTES.BACKUP_PATH_SEARCH,
  );
  return data.results;
}

export async function getBackupSQLServerDefaults(): Promise<BackupSQLServerDefaults> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.BACKUP_SQL_SERVER_DEFAULTS,
  );
  return validateResponse(
    BackupSQLServerDefaultsSchema,
    response.data,
    API_ROUTES.BACKUP_SQL_SERVER_DEFAULTS,
  );
}

const SQLServerODBCDSNSchema = z.object({
  name: z.string(),
  server: z.string(),
  database: z.string(),
  login_id: z.string(),
  last_user: z.string().optional(),
});

export type SQLServerODBCDSN = z.infer<typeof SQLServerODBCDSNSchema>;

const SQLServerODBCDSNsResponseSchema = z.object({
  dsns: z.array(SQLServerODBCDSNSchema),
});

const SQLServerODBCDSNRequestSchema = z.object({
  name: z.string().min(1),
  server: z.string().min(1),
  database: z.string().min(1),
  login_id: z.string().min(1),
  password: z.string().optional(),
});

export type SQLServerODBCDSNRequest = z.infer<
  typeof SQLServerODBCDSNRequestSchema
>;

const SQLServerODBCDSNTestRequestSchema = SQLServerODBCDSNRequestSchema.extend({
  password: z.string().min(1),
});

const SQLServerODBCDefaultDSNRequestSchema = z.object({
  server: z.string().min(1),
  login_id: z.string().min(1),
});

export type SQLServerODBCDefaultDSNRequest = z.infer<
  typeof SQLServerODBCDefaultDSNRequestSchema
>;

const SQLServerODBCDefaultDSNResponseSchema = z.object({
  created: z.array(z.string()),
  skipped: z.array(z.string()),
  dsns: z.array(SQLServerODBCDSNSchema),
});

export type SQLServerODBCDefaultDSNResponse = z.infer<
  typeof SQLServerODBCDefaultDSNResponseSchema
>;

const SQLServerODBCDSNTestResponseSchema = z.object({
  message: z.string(),
});

export async function getSQLServerODBCDSNs(): Promise<SQLServerODBCDSN[]> {
  const response = await axiosInstance.get<unknown>(
    API_ROUTES.ODBC_SQLSERVER_DSNS,
  );
  const data = validateResponse(
    SQLServerODBCDSNsResponseSchema,
    response.data,
    API_ROUTES.ODBC_SQLSERVER_DSNS,
  );
  return data.dsns;
}

export async function getSQLServerODBCDSN(
  name: string,
): Promise<SQLServerODBCDSN> {
  const route = API_ROUTES.ODBC_SQLSERVER_DSN(name);
  const response = await axiosInstance.get<unknown>(route);
  return validateResponse(SQLServerODBCDSNSchema, response.data, route);
}

export async function createSQLServerODBCDSN(
  payload: SQLServerODBCDSNRequest,
): Promise<SQLServerODBCDSN> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.ODBC_SQLSERVER_DSNS,
    SQLServerODBCDSNRequestSchema.parse(payload),
  );
  return validateResponse(
    SQLServerODBCDSNSchema,
    response.data,
    API_ROUTES.ODBC_SQLSERVER_DSNS,
  );
}

export async function updateSQLServerODBCDSN(
  name: string,
  payload: SQLServerODBCDSNRequest,
): Promise<SQLServerODBCDSN> {
  const route = API_ROUTES.ODBC_SQLSERVER_DSN(name);
  const response = await axiosInstance.put<unknown>(
    route,
    SQLServerODBCDSNRequestSchema.parse(payload),
  );
  return validateResponse(SQLServerODBCDSNSchema, response.data, route);
}

export async function createDefaultSQLServerODBCDSNs(
  payload: SQLServerODBCDefaultDSNRequest,
): Promise<SQLServerODBCDefaultDSNResponse> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.ODBC_SQLSERVER_DEFAULT_DSNS,
    SQLServerODBCDefaultDSNRequestSchema.parse(payload),
  );
  return validateResponse(
    SQLServerODBCDefaultDSNResponseSchema,
    response.data,
    API_ROUTES.ODBC_SQLSERVER_DEFAULT_DSNS,
  );
}

export async function deleteSQLServerODBCDSN(
  name: string,
): Promise<MessageResponse> {
  const route = API_ROUTES.ODBC_SQLSERVER_DSN(name);
  const response = await axiosInstance.delete<unknown>(route);
  return validateResponse(MessageResponseSchema, response.data, route);
}

export async function testSQLServerODBCDSNConnection(
  payload: SQLServerODBCDSNRequest,
): Promise<MessageResponse> {
  const response = await axiosInstance.post<unknown>(
    API_ROUTES.ODBC_SQLSERVER_DSN_TEST,
    SQLServerODBCDSNTestRequestSchema.parse(payload),
  );
  return validateResponse(
    SQLServerODBCDSNTestResponseSchema,
    response.data,
    API_ROUTES.ODBC_SQLSERVER_DSN_TEST,
  );
}
