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
  REVERT_FILE: '/api/file-tree/revert-file',
  REVISION_COUNT: '/api/file-tree/revision-summary',
  SETTINGS: '/api/settings',
  SETTING: (key: string) => `/api/settings/${key}`,
  METRICS_SUMMARY: '/api/metrics/summary',
  METRICS_CHARTS: '/api/metrics/charts',
  GAME_CLIENT_DATA_MONSTERS: '/api/game-client-data/monsters',
  GAME_CLIENT_DATA_UPLOAD_MON_FILE: '/api/game-client-data/upload-mon-file',
  GAME_CLIENT_DATA_MAPS: '/api/game-client-data/maps',
  GAME_CLIENT_DATA_UPLOAD_MC_FILE: '/api/game-client-data/upload-mc-file',
  GAME_CLIENT_DATA_ITEMS: '/api/game-client-data/items',
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
  metrics_enabled: z.boolean(),
});

export type StatusResponse = z.infer<typeof StatusResponseSchema>;

export interface SetupResponse {
  access_token: string;
}

const SettingsSchema = z.object({
  key: z.string(),
  value: z.string(),
  updated_at: z.string(),
});

export type Settings = z.infer<typeof SettingsSchema>;

const UpsertSettingRequestSchema = z.object({
  value: z.string(),
});

export type UpsertSettingRequest = z.infer<typeof UpsertSettingRequestSchema>;

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
    | 'a3_map_file'
    | 'a3_spawn_file'
    | 'a3_unknown_file'
    | 'text_file';
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
        'a3_map_file',
        'a3_spawn_file',
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
});

export type GameClientDataResponse = z.infer<
  typeof GameClientDataResponseSchema
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

export async function getAllSettings(): Promise<Settings[]> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.SETTINGS);
  return validateResponse(
    z.array(SettingsSchema),
    response.data,
    API_ROUTES.SETTINGS,
  );
}

export async function getSetting(key: string): Promise<Settings> {
  const response = await axiosInstance.get<unknown>(API_ROUTES.SETTING(key));
  return validateResponse(
    SettingsSchema,
    response.data,
    API_ROUTES.SETTING(key),
  );
}

export async function upsertSetting(
  key: string,
  data: UpsertSettingRequest,
): Promise<Settings> {
  const response = await axiosInstance.put<unknown>(
    API_ROUTES.SETTING(key),
    UpsertSettingRequestSchema.parse(data),
  );
  return validateResponse(
    SettingsSchema,
    response.data,
    API_ROUTES.SETTING(key),
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
  range?: '1h' | '6h' | '1d' | '7d';
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
