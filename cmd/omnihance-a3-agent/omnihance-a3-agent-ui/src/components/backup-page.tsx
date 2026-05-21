import type React from 'react';
import { useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Archive,
  ChevronLeft,
  ChevronRight,
  Clock,
  Database,
  Download,
  Edit,
  Eye,
  EyeOff,
  FileArchive,
  Folder,
  Loader2,
  Play,
  Plus,
  RefreshCw,
  Square,
  Trash2,
} from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  cn,
  emptyToNull,
  formatBytes,
  formatExactDateTime,
  formatRelativeDateTime,
  formatStatusLabel,
  maskSecret,
} from '@/lib/util';
import { PathAutocomplete } from '@/components/path-autocomplete';
import { queryKeys } from '@/constants';
import {
  APIError,
  cancelBackupJob,
  createBackupJob,
  deleteBackupJob,
  getBackupJob,
  getBackupJobs,
  getBackupRunDetails,
  getBackupRunFileDownloadUrl,
  getBackupRuns,
  getBackupSQLServerDefaults,
  runBackupJob,
  updateBackupJob,
  type BackupJob,
  type BackupJobRequest,
  type BackupJobType,
  type BackupRun,
  type BackupRunDetails,
  type BackupSQLServerDefaults,
} from '@/lib/api';

type BackupFormState = {
  job_type: BackupJobType;
  name: string;
  status: string;
  cron_expression: string;
  destination_directory: string;
  archive_password: string;
  source_path: string;
  sql_host: string;
  sql_port: string;
  sql_username: string;
  sql_password: string;
  sql_database_names: string;
};

const emptyForm: BackupFormState = {
  job_type: 'file',
  name: '',
  status: 'active',
  cron_expression: '',
  destination_directory: '',
  archive_password: '',
  source_path: '',
  sql_host: '',
  sql_port: '1433',
  sql_username: '',
  sql_password: '',
  sql_database_names: '',
};

const backupRunPageSize = 10;
const emptyBackupRuns: BackupRun[] = [];

export function BackupPage() {
  const queryClient = useQueryClient();
  const [selectedJobId, setSelectedJobId] = useState<number | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [editingJob, setEditingJob] = useState<BackupJob | null>(null);
  const [form, setForm] = useState<BackupFormState>(emptyForm);
  const sqlDefaultsAppliedRef = useRef(false);
  const sqlDefaultsRequestIdRef = useRef(0);
  const [deleteJob, setDeleteJob] = useState<BackupJob | null>(null);
  const [selectedRunId, setSelectedRunId] = useState<number | null>(null);
  const [runPaging, setRunPaging] = useState<{
    jobId: number | null;
    page: number;
  }>({
    jobId: null,
    page: 1,
  });

  const { data: jobs = [], isLoading } = useQuery({
    queryKey: queryKeys.backupJobs,
    queryFn: getBackupJobs,
    refetchInterval: (query) => {
      const data = query.state.data as BackupJob[] | undefined;
      return data?.some((job) => job.status === 'running') ? 3000 : false;
    },
  });

  const activeJobId = selectedJobId ?? jobs[0]?.id ?? null;
  const runPage = runPaging.jobId === activeJobId ? runPaging.page : 1;
  const runPageSize = backupRunPageSize;

  const { data: selectedJob } = useQuery({
    queryKey: activeJobId
      ? queryKeys.backupJob(activeJobId)
      : ['backup-job', 'none'],
    queryFn: () => getBackupJob(activeJobId as number),
    enabled: activeJobId !== null,
    refetchInterval: (query) => {
      const data = query.state.data as BackupJob | undefined;
      return data?.status === 'running' ? 3000 : false;
    },
  });

  const { data: runsData, isLoading: runsLoading } = useQuery({
    queryKey: activeJobId
      ? queryKeys.backupRuns(activeJobId, runPage, runPageSize)
      : ['backup-runs', 'none'],
    queryFn: () =>
      getBackupRuns(activeJobId as number, {
        page: runPage,
        pageSize: runPageSize,
      }),
    enabled: activeJobId !== null,
    refetchInterval: selectedJob?.status === 'running' ? 3000 : false,
  });

  const runs = runsData?.runs ?? emptyBackupRuns;
  const runsPagination = runsData?.pagination;
  const totalRunPages = runsPagination
    ? Math.max(
        1,
        Math.ceil(runsPagination.totalCount / runsPagination.pageSize),
      )
    : 1;

  const { data: runDetails } = useQuery({
    queryKey: selectedRunId
      ? queryKeys.backupRun(selectedRunId)
      : ['backup-run', 'none'],
    queryFn: () => getBackupRunDetails(selectedRunId as number),
    enabled: selectedRunId !== null,
  });

  const { data: sqlDefaults } = useQuery({
    queryKey: queryKeys.backupSqlServerDefaults,
    queryFn: getBackupSQLServerDefaults,
    enabled: formOpen && form.job_type === 'sql_server' && editingJob === null,
  });

  const prefillCreateSQLDefaults = () => {
    if (sqlDefaultsAppliedRef.current) {
      return;
    }

    const requestId = sqlDefaultsRequestIdRef.current;
    void queryClient
      .fetchQuery({
        queryKey: queryKeys.backupSqlServerDefaults,
        queryFn: getBackupSQLServerDefaults,
      })
      .then((defaults) => {
        if (
          requestId !== sqlDefaultsRequestIdRef.current ||
          sqlDefaultsAppliedRef.current
        ) {
          return;
        }

        sqlDefaultsAppliedRef.current = true;
        setForm((current) => ({
          ...current,
          sql_host: defaults.host,
          sql_port: defaults.port.toString(),
          sql_username: defaults.username,
          sql_password: defaults.password,
        }));
      })
      .catch(() => undefined);
  };

  const createMutation = useMutation({
    mutationFn: createBackupJob,
    onSuccess: async (job) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.backupJobs });
      setSelectedJobId(job.id);
      setFormOpen(false);
      toast.success('Backup job created');
    },
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: BackupJobRequest }) =>
      updateBackupJob(id, payload),
    onSuccess: async (job) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.backupJobs }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.backupJob(job.id),
        }),
      ]);
      setSelectedJobId(job.id);
      setFormOpen(false);
      setEditingJob(null);
      toast.success('Backup job updated');
    },
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteBackupJob,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.backupJobs });
      if (deleteJob?.id === activeJobId) {
        setSelectedJobId(null);
      }
      setDeleteJob(null);
      toast.success('Backup job deleted');
    },
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
  });

  const runMutation = useMutation({
    mutationFn: runBackupJob,
    onSuccess: async (run) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.backupJobs }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.backupJob(run.job_id),
        }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.backupRuns(run.job_id),
        }),
      ]);
      toast.success('Backup started');
    },
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
  });

  const cancelMutation = useMutation({
    mutationFn: cancelBackupJob,
    onSuccess: async (run) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.backupJobs }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.backupJob(run.job_id),
        }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.backupRuns(run.job_id),
        }),
      ]);
      toast.success('Cancellation requested');
    },
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
  });

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const selectedIsRunning = selectedJob?.status === 'running';

  const openCreate = () => {
    setEditingJob(null);
    setForm(emptyForm);
    sqlDefaultsRequestIdRef.current += 1;
    sqlDefaultsAppliedRef.current = false;
    setFormOpen(true);
    prefillCreateSQLDefaults();
  };

  const openEdit = (job: BackupJob) => {
    setEditingJob(job);
    setForm(jobToForm(job));
    sqlDefaultsRequestIdRef.current += 1;
    sqlDefaultsAppliedRef.current = true;
    setFormOpen(true);
  };

  const handleSave = () => {
    const validationError = validateForm(form);
    if (validationError) {
      toast.error(validationError);
      return;
    }

    const payload = formToPayload(form);
    if (editingJob) {
      updateMutation.mutate({ id: editingJob.id, payload });
      return;
    }

    createMutation.mutate(payload);
  };

  const handleRunPageChange = (newPage: number) => {
    if (activeJobId !== null && newPage >= 1 && newPage <= totalRunPages) {
      setRunPaging({ jobId: activeJobId, page: newPage });
    }
  };

  const latestRun = runPage === 1 ? runs[0] : undefined;

  return (
    <div className="space-y-6 p-4 lg:p-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Backups</h1>
          <p className="text-muted-foreground">
            Schedule and run server file and SQL Server backups
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            onClick={() =>
              queryClient.invalidateQueries({ queryKey: queryKeys.backupJobs })
            }
            aria-label="Refresh backup jobs"
          >
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <Button onClick={openCreate} aria-label="Create backup job">
            <Plus className="mr-2 h-4 w-4" />
            New Job
          </Button>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_minmax(24rem,0.9fr)]">
        <Card>
          <CardHeader>
            <CardTitle>Backup Jobs</CardTitle>
            <CardDescription>{jobs.length} configured job(s)</CardDescription>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="flex justify-center py-12">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            ) : jobs.length === 0 ? (
              <div className="flex flex-col items-center justify-center gap-3 py-12 text-center">
                <FileArchive className="h-10 w-10 text-muted-foreground" />
                <div className="font-medium">No backup jobs configured</div>
                <Button onClick={openCreate}>
                  <Plus className="mr-2 h-4 w-4" />
                  Create Job
                </Button>
              </div>
            ) : (
              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Schedule</TableHead>
                      <TableHead>Last Run</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {jobs.map((job) => (
                      <TableRow
                        key={job.id}
                        className={cn(
                          'cursor-pointer',
                          activeJobId === job.id && 'bg-muted/60',
                        )}
                        onClick={() => setSelectedJobId(job.id)}
                        tabIndex={0}
                        aria-label={`View ${job.name}`}
                        onKeyDown={(event) => {
                          if (event.key === 'Enter' || event.key === ' ') {
                            setSelectedJobId(job.id);
                          }
                        }}
                      >
                        <TableCell className="font-medium">
                          {job.name}
                        </TableCell>
                        <TableCell>
                          <BackupTypeBadge jobType={job.job_type} />
                        </TableCell>
                        <TableCell>
                          <StatusBadge status={job.status} />
                        </TableCell>
                        <TableCell className="font-mono text-xs">
                          {job.cron_expression || 'Manual'}
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          <RelativeDateTime value={job.last_run_at} />
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-2">
                            {job.status === 'running' ? (
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={(event) => {
                                  event.stopPropagation();
                                  cancelMutation.mutate(job.id);
                                }}
                                disabled={cancelMutation.isPending}
                                aria-label={`Cancel ${job.name}`}
                              >
                                <Square className="h-4 w-4" />
                              </Button>
                            ) : (
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={(event) => {
                                  event.stopPropagation();
                                  runMutation.mutate(job.id);
                                }}
                                disabled={runMutation.isPending}
                                aria-label={`Run ${job.name}`}
                              >
                                <Play className="h-4 w-4" />
                              </Button>
                            )}
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={(event) => {
                                event.stopPropagation();
                                openEdit(job);
                              }}
                              disabled={job.status === 'running'}
                              aria-label={`Edit ${job.name}`}
                            >
                              <Edit className="h-4 w-4" />
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={(event) => {
                                event.stopPropagation();
                                setDeleteJob(job);
                              }}
                              disabled={job.status === 'running'}
                              aria-label={`Delete ${job.name}`}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Job Details</CardTitle>
            <CardDescription>
              {selectedJob ? (
                <RunSummary
                  latestRun={latestRun}
                  lastRunAt={selectedJob.last_run_at}
                />
              ) : (
                'Select a job'
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {selectedJob ? (
              <div className="space-y-6">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="flex items-center gap-2">
                    <BackupTypeBadge jobType={selectedJob.job_type} />
                    <StatusBadge status={selectedJob.status} />
                  </div>
                  <div className="flex gap-2">
                    {selectedIsRunning ? (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => cancelMutation.mutate(selectedJob.id)}
                        disabled={cancelMutation.isPending}
                      >
                        <Square className="mr-2 h-4 w-4" />
                        Cancel
                      </Button>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => runMutation.mutate(selectedJob.id)}
                        disabled={runMutation.isPending}
                      >
                        <Play className="mr-2 h-4 w-4" />
                        Run
                      </Button>
                    )}
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => openEdit(selectedJob)}
                      disabled={selectedIsRunning}
                    >
                      <Edit className="mr-2 h-4 w-4" />
                      Edit
                    </Button>
                  </div>
                </div>

                <DetailGrid job={selectedJob} />

                <div className="space-y-3">
                  <div className="flex items-center gap-2 font-medium">
                    <Clock className="h-4 w-4" />
                    Runs
                  </div>
                  <div className="rounded-md border">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Started</TableHead>
                          <TableHead>Status</TableHead>
                          <TableHead>Trigger</TableHead>
                          <TableHead className="text-right">Details</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {runsLoading ? (
                          <TableRow>
                            <TableCell colSpan={4} className="py-8 text-center">
                              <Loader2 className="mx-auto h-5 w-5 animate-spin text-muted-foreground" />
                            </TableCell>
                          </TableRow>
                        ) : runs.length === 0 ? (
                          <TableRow>
                            <TableCell
                              colSpan={4}
                              className="py-8 text-center text-muted-foreground"
                            >
                              No runs recorded
                            </TableCell>
                          </TableRow>
                        ) : (
                          runs.map((run) => (
                            <TableRow key={run.id}>
                              <TableCell>
                                <RelativeDateTime value={run.started_at} />
                              </TableCell>
                              <TableCell>
                                <StatusBadge status={run.status} />
                              </TableCell>
                              <TableCell className="capitalize">
                                {run.trigger_type}
                              </TableCell>
                              <TableCell className="text-right">
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => setSelectedRunId(run.id)}
                                >
                                  View
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))
                        )}
                      </TableBody>
                    </Table>
                  </div>
                  {runsPagination && runsPagination.totalCount > 0 && (
                    <div className="flex flex-col gap-3 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
                      <div>
                        Showing {(runPage - 1) * runPageSize + 1} to{' '}
                        {Math.min(
                          runPage * runPageSize,
                          runsPagination.totalCount,
                        )}{' '}
                        of {runsPagination.totalCount} runs
                      </div>
                      <div className="flex items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleRunPageChange(runPage - 1)}
                          disabled={runPage === 1 || runsLoading}
                        >
                          <ChevronLeft className="h-4 w-4" />
                          Previous
                        </Button>
                        <div>
                          Page {runPage} of {totalRunPages}
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleRunPageChange(runPage + 1)}
                          disabled={runPage >= totalRunPages || runsLoading}
                        >
                          Next
                          <ChevronRight className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center gap-3 py-12 text-center text-muted-foreground">
                <Archive className="h-10 w-10" />
                <div>Select a backup job to view details</div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <BackupJobDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        form={form}
        setForm={setForm}
        editingJob={editingJob}
        onSave={handleSave}
        isSaving={isSaving}
        sqlDefaults={sqlDefaults}
        onSqlServerSelected={prefillCreateSQLDefaults}
      />

      <AlertDialog
        open={deleteJob !== null}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteJob(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Backup Job</AlertDialogTitle>
            <AlertDialogDescription>
              Delete {deleteJob?.name}? Runs and output history remain in the
              database, but the job will no longer appear in the active list.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deleteJob) {
                  deleteMutation.mutate(deleteJob.id);
                }
              }}
              disabled={deleteMutation.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <RunDetailsDialog
        details={runDetails}
        open={selectedRunId !== null}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedRunId(null);
          }
        }}
      />
    </div>
  );
}

function RunSummary({
  latestRun,
  lastRunAt,
}: {
  latestRun?: BackupRun;
  lastRunAt?: string | null;
}) {
  if (latestRun) {
    return (
      <>
        {formatStatusLabel(latestRun.status)}{' '}
        <RelativeDateTime value={latestRun.started_at} />
      </>
    );
  }

  if (lastRunAt) {
    return (
      <>
        Last run <RelativeDateTime value={lastRunAt} />
      </>
    );
  }

  return 'No runs yet';
}

function RelativeDateTime({
  value,
  className,
}: {
  value?: string | null;
  className?: string;
}) {
  return (
    <time
      className={className}
      dateTime={value || undefined}
      title={formatExactDateTime(value)}
    >
      {formatRelativeDateTime(value)}
    </time>
  );
}

function BackupJobDialog({
  open,
  onOpenChange,
  form,
  setForm,
  editingJob,
  onSave,
  isSaving,
  sqlDefaults,
  onSqlServerSelected,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  form: BackupFormState;
  setForm: React.Dispatch<React.SetStateAction<BackupFormState>>;
  editingJob: BackupJob | null;
  onSave: () => void;
  isSaving: boolean;
  sqlDefaults?: BackupSQLServerDefaults;
  onSqlServerSelected: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {editingJob ? 'Edit Backup Job' : 'New Backup Job'}
          </DialogTitle>
          <DialogDescription>
            Configure manual or scheduled backup creation
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-2">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="backup-name">Name</Label>
              <Input
                id="backup-name"
                value={form.name}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    name: event.target.value,
                  }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="backup-type">Type</Label>
              <Select
                value={form.job_type}
                onValueChange={(value) => {
                  const jobType = value as BackupJobType;
                  setForm((current) => ({
                    ...current,
                    job_type: jobType,
                  }));
                  if (jobType === 'sql_server' && !editingJob) {
                    onSqlServerSelected();
                  }
                }}
              >
                <SelectTrigger id="backup-type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="file">File or Directory</SelectItem>
                  <SelectItem value="sql_server">SQL Server</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="backup-status">Status</Label>
              <Select
                value={form.status}
                onValueChange={(value) =>
                  setForm((current) => ({ ...current, status: value }))
                }
              >
                <SelectTrigger id="backup-status">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="inactive">Inactive</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="backup-cron">Cron Expression</Label>
              <Input
                id="backup-cron"
                value={form.cron_expression}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    cron_expression: event.target.value,
                  }))
                }
                placeholder="0 2 * * *"
              />
            </div>
          </div>

          <PathAutocomplete
            id="backup-destination"
            label="Destination Directory"
            value={form.destination_directory}
            kind="directory"
            onChange={(value) =>
              setForm((current) => ({
                ...current,
                destination_directory: value,
              }))
            }
          />

          <div className="space-y-2">
            <Label htmlFor="backup-archive-password">Archive Password</Label>
            <PasswordInput
              id="backup-archive-password"
              value={form.archive_password}
              onChange={(value) =>
                setForm((current) => ({
                  ...current,
                  archive_password: value,
                }))
              }
            />
          </div>

          {form.job_type === 'file' ? (
            <PathAutocomplete
              id="backup-source"
              label="Input File or Directory"
              value={form.source_path}
              kind="input"
              onChange={(value) =>
                setForm((current) => ({ ...current, source_path: value }))
              }
            />
          ) : (
            <div className="space-y-4">
              {sqlDefaults?.local_server_running === false && (
                <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
                  Local SQL Server service was not detected as running.
                </div>
              )}
              <div className="grid gap-4 sm:grid-cols-[minmax(0,1fr)_8rem]">
                <div className="space-y-2">
                  <Label htmlFor="backup-sql-host">SQL Server Host</Label>
                  <Input
                    id="backup-sql-host"
                    value={form.sql_host}
                    onChange={(event) =>
                      setForm((current) => ({
                        ...current,
                        sql_host: event.target.value,
                      }))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="backup-sql-port">Port</Label>
                  <Input
                    id="backup-sql-port"
                    type="number"
                    value={form.sql_port}
                    onChange={(event) =>
                      setForm((current) => ({
                        ...current,
                        sql_port: event.target.value,
                      }))
                    }
                    min={1}
                    max={65535}
                  />
                </div>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="backup-sql-user">SQL Username</Label>
                  <Input
                    id="backup-sql-user"
                    value={form.sql_username}
                    onChange={(event) =>
                      setForm((current) => ({
                        ...current,
                        sql_username: event.target.value,
                      }))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="backup-sql-password">SQL Password</Label>
                  <PasswordInput
                    id="backup-sql-password"
                    value={form.sql_password}
                    onChange={(value) =>
                      setForm((current) => ({
                        ...current,
                        sql_password: value,
                      }))
                    }
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="backup-sql-dbs">Database Names</Label>
                <Input
                  id="backup-sql-dbs"
                  value={form.sql_database_names}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      sql_database_names: event.target.value,
                    }))
                  }
                  placeholder="ASD, FriendDB, HSDB"
                />
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isSaving}
          >
            Cancel
          </Button>
          <Button onClick={onSave} disabled={isSaving}>
            {isSaving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {editingJob ? 'Update' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function PasswordInput({
  id,
  value,
  onChange,
}: {
  id: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const [visible, setVisible] = useState(false);

  return (
    <div className="relative">
      <Input
        id={id}
        type={visible ? 'text' : 'password'}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="pr-10"
        autoComplete="new-password"
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="absolute right-1 top-1/2 h-8 w-8 -translate-y-1/2"
        onClick={() => setVisible((current) => !current)}
        aria-label={visible ? 'Hide password' : 'Show password'}
      >
        {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
      </Button>
    </div>
  );
}

function DetailGrid({ job }: { job: BackupJob }) {
  const items: DetailGridItem[] = [
    { label: 'Name', value: job.name },
    { label: 'Destination', value: job.destination_directory },
    { label: 'Cron', value: job.cron_expression || 'Manual only' },
    {
      label: 'Archive Password',
      value: job.archive_password || 'Not set',
      secret: Boolean(job.archive_password),
    },
    { label: 'Last Run', value: job.last_run_at, dateTime: true },
    { label: 'Last Updated', value: job.updated_at, dateTime: true },
  ];

  if (job.job_type === 'file') {
    items.push({ label: 'Input', value: job.source_path || '-' });
  } else {
    items.push(
      {
        label: 'SQL Host',
        value: `${job.sql_host || '-'}:${job.sql_port || '-'}`,
      },
      { label: 'SQL Username', value: job.sql_username || '-' },
      {
        label: 'SQL Password',
        value: job.sql_password || 'Not set',
        secret: Boolean(job.sql_password),
      },
      { label: 'Databases', value: job.sql_database_names || '-' },
    );
  }

  return (
    <div className="grid gap-3">
      {items.map((item) => (
        <div key={item.label} className="grid gap-1">
          <div className="text-xs font-medium uppercase text-muted-foreground">
            {item.label}
          </div>
          <DetailValue
            value={item.value}
            secret={item.secret}
            dateTime={item.dateTime}
          />
        </div>
      ))}
    </div>
  );
}

function DetailValue({
  value,
  secret,
  dateTime,
}: {
  value?: string | null;
  secret?: boolean;
  dateTime?: boolean;
}) {
  const [visible, setVisible] = useState(false);
  const displayValue = value || '-';

  if (dateTime) {
    return (
      <div className="break-words rounded-md border bg-muted/30 px-3 py-2 font-mono text-sm">
        <RelativeDateTime value={value} />
      </div>
    );
  }

  if (!secret) {
    return (
      <div className="break-words rounded-md border bg-muted/30 px-3 py-2 font-mono text-sm">
        {displayValue}
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2 rounded-md border bg-muted/30 px-3 py-2 font-mono text-sm">
      <span className="min-w-0 flex-1 break-words">
        {visible ? displayValue : maskSecret(displayValue)}
      </span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="h-7 w-7 shrink-0"
        onClick={() => setVisible((current) => !current)}
        aria-label={visible ? 'Hide password' : 'Show password'}
      >
        {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
      </Button>
    </div>
  );
}

type DetailGridItem = {
  label: string;
  value?: string | null;
  secret?: boolean;
  dateTime?: boolean;
};

function RunDetailsDialog({
  details,
  open,
  onOpenChange,
}: {
  details?: BackupRunDetails;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] w-[calc(100vw-2rem)] max-w-3xl overflow-hidden">
        <DialogHeader>
          <DialogTitle>Backup Run</DialogTitle>
          <DialogDescription>
            {details ? (
              <RelativeDateTime value={details.run.started_at} />
            ) : (
              'Loading'
            )}
          </DialogDescription>
        </DialogHeader>
        {details ? (
          <div className="min-w-0 space-y-4 overflow-y-auto pr-1">
            <div className="flex flex-wrap gap-2">
              <StatusBadge status={details.run.status} />
              <Badge variant="outline" className="capitalize">
                {details.run.trigger_type}
              </Badge>
            </div>
            {details.files.length > 0 && (
              <div className="overflow-hidden rounded-md border">
                <div className="border-b bg-muted/30 px-3 py-2 text-xs font-medium uppercase text-muted-foreground">
                  Output Files
                </div>
                <div className="divide-y">
                  {details.files.map((file) => (
                    <div
                      key={file.id}
                      className="grid min-w-0 gap-3 px-3 py-3 sm:grid-cols-[minmax(0,1fr)_5rem_3rem] sm:items-center"
                    >
                      <div className="min-w-0">
                        <div
                          className="truncate font-medium"
                          title={file.item_name}
                        >
                          {file.item_name}
                        </div>
                        <div
                          className="mt-1 min-w-0 font-mono text-xs text-muted-foreground [overflow-wrap:anywhere]"
                          title={file.file_path}
                        >
                          {file.file_path}
                        </div>
                      </div>
                      <div className="text-sm text-muted-foreground">
                        {formatBytes(file.file_size, 1)}
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        className="w-fit sm:justify-self-end"
                        asChild
                      >
                        <a
                          href={getBackupRunFileDownloadUrl(
                            details.run.id,
                            file.id,
                          )}
                          aria-label={`Download ${file.item_name}`}
                        >
                          <Download className="h-4 w-4" />
                        </a>
                      </Button>
                    </div>
                  ))}
                </div>
              </div>
            )}
            <ScrollArea className="h-64 rounded-md border bg-muted/30 p-3">
              <pre className="whitespace-pre-wrap text-xs [overflow-wrap:anywhere]">
                {details.run.output || details.run.error_details || 'No output'}
              </pre>
            </ScrollArea>
          </div>
        ) : (
          <div className="flex justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function BackupTypeBadge({ jobType }: { jobType: BackupJobType }) {
  if (jobType === 'sql_server') {
    return (
      <Badge variant="secondary">
        <Database className="h-3 w-3" />
        SQL Server
      </Badge>
    );
  }

  return (
    <Badge variant="outline">
      <Folder className="h-3 w-3" />
      Files
    </Badge>
  );
}

function StatusBadge({ status }: { status: string }) {
  const normalized = status.toLowerCase();
  const className =
    normalized === 'running'
      ? 'bg-blue-600 text-white'
      : normalized === 'succeeded' || normalized === 'active'
        ? 'bg-green-600 text-white'
        : normalized === 'failed' || normalized === 'deleted'
          ? 'bg-destructive text-destructive-foreground'
          : normalized === 'cancelled' || normalized === 'inactive'
            ? 'bg-amber-600 text-white'
            : normalized === 'skipped'
              ? 'bg-muted text-muted-foreground'
              : '';

  return <Badge className={className}>{formatStatusLabel(status)}</Badge>;
}

function jobToForm(job: BackupJob): BackupFormState {
  return {
    job_type: job.job_type,
    name: job.name,
    status: job.status === 'running' ? 'active' : job.status,
    cron_expression: job.cron_expression || '',
    destination_directory: job.destination_directory,
    archive_password: job.archive_password || '',
    source_path: job.source_path || '',
    sql_host: job.sql_host || '',
    sql_port: job.sql_port?.toString() || '1433',
    sql_username: job.sql_username || '',
    sql_password: job.sql_password || '',
    sql_database_names: job.sql_database_names || '',
  };
}

function formToPayload(form: BackupFormState): BackupJobRequest {
  const payload: BackupJobRequest = {
    job_type: form.job_type,
    name: form.name.trim(),
    status: form.status,
    cron_expression: emptyToNull(form.cron_expression),
    destination_directory: form.destination_directory.trim(),
    archive_password: emptyToNull(form.archive_password),
  };

  if (form.job_type === 'file') {
    payload.source_path = emptyToNull(form.source_path);
    return payload;
  }

  payload.sql_host = emptyToNull(form.sql_host);
  payload.sql_port = form.sql_port.trim() ? Number(form.sql_port) : null;
  payload.sql_username = emptyToNull(form.sql_username);
  payload.sql_password = form.sql_password;
  payload.sql_database_names = emptyToNull(form.sql_database_names);
  return payload;
}

function validateForm(form: BackupFormState): string | null {
  if (!form.name.trim()) {
    return 'Job name is required';
  }

  if (!form.destination_directory.trim()) {
    return 'Destination directory is required';
  }

  if (form.job_type === 'file' && !form.source_path.trim()) {
    return 'Input file or directory is required';
  }

  if (form.job_type === 'sql_server') {
    if (!form.sql_host.trim()) {
      return 'SQL Server host is required';
    }

    const port = Number(form.sql_port);
    if (!Number.isInteger(port) || port < 1 || port > 65535) {
      return 'SQL Server port must be between 1 and 65535';
    }

    if (!form.sql_username.trim()) {
      return 'SQL Server username is required';
    }

    if (!form.sql_database_names.trim()) {
      return 'At least one database name is required';
    }
  }

  return null;
}
