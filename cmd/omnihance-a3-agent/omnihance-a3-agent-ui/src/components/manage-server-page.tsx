import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Loader2,
  Play,
  Square,
  Trash2,
  Edit,
  Plus,
  ChevronUp,
  ChevronDown,
  AlertCircle,
} from 'lucide-react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Button } from '@/components/ui/button';
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
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  getServerProcesses,
  createServerProcess,
  updateServerProcess,
  deleteServerProcess,
  reorderServerProcesses,
  startFullServer,
  stopFullServer,
  startProcess,
  stopProcess,
  getProcessStatus,
  type ServerProcess,
  type CreateServerProcessRequest,
  type UpdateServerProcessRequest,
  type ReorderServerProcessesRequest,
  type ProcessStatus,
  APIError,
} from '@/lib/api';
import { queryKeys } from '@/constants';
import { toast } from 'sonner';
import { usePermissions } from '@/hooks/use-permissions';

function formatUptime(seconds: number): string {
  if (seconds < 60) {
    return `${seconds}s`;
  }
  if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60);
    return `${minutes}m`;
  }
  if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
  }
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  return hours > 0 ? `${days}d ${hours}h` : `${days}d`;
}

export function ManageServerPage() {
  const queryClient = useQueryClient();
  const { hasPermission } = usePermissions();
  const canManageServer = hasPermission('manage_server');
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedProcess, setSelectedProcess] = useState<ServerProcess | null>(
    null,
  );
  const [addFormName, setAddFormName] = useState('');
  const [addFormPath, setAddFormPath] = useState('');
  const [addFormPort, setAddFormPort] = useState('');
  const [editFormName, setEditFormName] = useState('');
  const [editFormPath, setEditFormPath] = useState('');
  const [editFormPort, setEditFormPort] = useState('');

  const { data: processes, isLoading } = useQuery({
    queryKey: queryKeys.serverProcesses,
    queryFn: getServerProcesses,
    refetchInterval: (query) => {
      const data = query.state.data as ServerProcess[] | undefined;
      if (data && data.some((p) => p.start_time !== null)) {
        return 3000;
      }
      return false;
    },
  });

  const [processStatuses, setProcessStatuses] = useState<
    Map<number, ProcessStatus>
  >(new Map());

  useEffect(() => {
    if (!processes) {
      return;
    }

    const fetchStatuses = async () => {
      const statusMap = new Map<number, ProcessStatus>();
      for (const proc of processes) {
        try {
          const status = await getProcessStatus(proc.id);
          statusMap.set(proc.id, status);
        } catch (error) {
          console.error(
            `Failed to fetch status for process ${proc.id}:`,
            error,
          );
        }
      }
      setProcessStatuses(statusMap);
    };

    fetchStatuses();

    const hasRunningProcesses = processes.some((p) => p.start_time !== null);
    if (!hasRunningProcesses) {
      return;
    }

    const interval = setInterval(() => {
      fetchStatuses();
    }, 3000);

    return () => clearInterval(interval);
  }, [processes]);

  useEffect(() => {
    if (!processes) {
      return;
    }

    const runningProcesses = processes.filter((p) => {
      const status = processStatuses.get(p.id);
      return status?.running === true;
    });

    if (runningProcesses.length === 0) {
      return;
    }

    const interval = setInterval(() => {
      setProcessStatuses((prev) => {
        const updated = new Map(prev);
        runningProcesses.forEach((proc) => {
          const status = updated.get(proc.id);
          if (status && proc.start_time) {
            const startTime = new Date(proc.start_time).getTime();
            const now = Date.now();
            const currentUptime = Math.floor((now - startTime) / 1000);
            updated.set(proc.id, {
              ...status,
              current_uptime_seconds: currentUptime,
            });
          }
        });
        return updated;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [processes, processStatuses]);

  const createMutation = useMutation({
    mutationFn: (data: CreateServerProcessRequest) => createServerProcess(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.serverProcesses });
      toast.success('Process added successfully');
      setAddDialogOpen(false);
      setAddFormName('');
      setAddFormPath('');
      setAddFormPort('');
    },
    onError: (error: APIError) => {
      toast.error(error.getErrorMessage());
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: number;
      data: UpdateServerProcessRequest;
    }) => updateServerProcess(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.serverProcesses });
      toast.success('Process updated successfully');
      setEditDialogOpen(false);
      setSelectedProcess(null);
    },
    onError: (error: APIError) => {
      toast.error(error.getErrorMessage());
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteServerProcess(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.serverProcesses });
      toast.success('Process deleted successfully');
      setDeleteDialogOpen(false);
      setSelectedProcess(null);
    },
    onError: (error: APIError) => {
      toast.error(error.getErrorMessage());
    },
  });

  const reorderMutation = useMutation({
    mutationFn: (data: ReorderServerProcessesRequest) =>
      reorderServerProcesses(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.serverProcesses });
      toast.success('Processes reordered successfully');
    },
    onError: (error: APIError) => {
      toast.error(error.getErrorMessage());
    },
  });

  const startFullMutation = useMutation({
    mutationFn: () => startFullServer(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.serverProcesses });
      toast.success('Server started successfully');
    },
    onError: (error: APIError) => {
      toast.error(error.getErrorMessage());
    },
  });

  const stopFullMutation = useMutation({
    mutationFn: () => stopFullServer(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.serverProcesses });
      toast.success('Server stopped successfully');
    },
    onError: (error: APIError) => {
      toast.error(error.getErrorMessage());
    },
  });

  const startProcessMutation = useMutation({
    mutationFn: (id: number) => startProcess(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.serverProcesses });
      toast.success('Process started successfully');
    },
    onError: (error: APIError) => {
      toast.error(error.getErrorMessage());
    },
  });

  const stopProcessMutation = useMutation({
    mutationFn: (id: number) => stopProcess(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.serverProcesses });
      toast.success('Process stopped successfully');
    },
    onError: (error: APIError) => {
      toast.error(error.getErrorMessage());
    },
  });

  const handleAdd = () => {
    if (!addFormName.trim() || !addFormPath.trim()) {
      toast.error('Name and path are required');
      return;
    }

    const port = addFormPort.trim()
      ? parseInt(addFormPort.trim(), 10)
      : undefined;

    if (port !== undefined && (isNaN(port) || port < 1 || port > 65535)) {
      toast.error('Port must be a number between 1 and 65535');
      return;
    }

    createMutation.mutate({
      name: addFormName.trim(),
      path: addFormPath.trim(),
      port,
    });
  };

  const handleEdit = () => {
    if (!selectedProcess) {
      return;
    }

    if (!editFormName.trim() || !editFormPath.trim()) {
      toast.error('Name and path are required');
      return;
    }

    const port = editFormPort.trim()
      ? parseInt(editFormPort.trim(), 10)
      : undefined;

    if (port !== undefined && (isNaN(port) || port < 1 || port > 65535)) {
      toast.error('Port must be a number between 1 and 65535');
      return;
    }

    updateMutation.mutate({
      id: selectedProcess.id,
      data: {
        name: editFormName.trim(),
        path: editFormPath.trim(),
        port,
      },
    });
  };

  const handleDelete = () => {
    if (!selectedProcess) {
      return;
    }
    deleteMutation.mutate(selectedProcess.id);
  };

  const handleMoveUp = (index: number) => {
    if (!processes || index === 0) {
      return;
    }

    const updates = processes.map((proc, i) => {
      if (i === index) {
        return { id: proc.id, sequence_order: proc.sequence_order - 1 };
      }
      if (i === index - 1) {
        return { id: proc.id, sequence_order: proc.sequence_order + 1 };
      }
      return { id: proc.id, sequence_order: proc.sequence_order };
    });

    reorderMutation.mutate({ updates });
  };

  const handleMoveDown = (index: number) => {
    if (!processes || index === processes.length - 1) {
      return;
    }

    const updates = processes.map((proc, i) => {
      if (i === index) {
        return { id: proc.id, sequence_order: proc.sequence_order + 1 };
      }
      if (i === index + 1) {
        return { id: proc.id, sequence_order: proc.sequence_order - 1 };
      }
      return { id: proc.id, sequence_order: proc.sequence_order };
    });

    reorderMutation.mutate({ updates });
  };

  const openEditDialog = (process: ServerProcess) => {
    setSelectedProcess(process);
    setEditFormName(process.name);
    setEditFormPath(process.path);
    setEditFormPort(process.port?.toString() || '');
    setEditDialogOpen(true);
  };

  const openDeleteDialog = (process: ServerProcess) => {
    setSelectedProcess(process);
    setDeleteDialogOpen(true);
  };

  const anyRunning = processes?.some((p) => {
    const status = processStatuses.get(p.id);
    return status?.running === true;
  });

  const allStopped = !processes?.some((p) => {
    const status = processStatuses.get(p.id);
    return status?.running === true;
  });

  const isEmpty = !processes || processes.length === 0;

  return (
    <div className="p-4 lg:p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            Game Server Management
          </h1>
          <p className="text-muted-foreground">
            Manage server processes and startup sequence
          </p>
        </div>
        {canManageServer && (
          <div className="flex items-center gap-2">
            <Button
              onClick={() => startFullMutation.mutate()}
              disabled={startFullMutation.isPending || anyRunning || isEmpty}
              variant="default"
            >
              {startFullMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Starting...
                </>
              ) : (
                <>
                  <Play className="mr-2 h-4 w-4" />
                  Start Full Server
                </>
              )}
            </Button>
            <Button
              onClick={() => stopFullMutation.mutate()}
              disabled={stopFullMutation.isPending || allStopped || isEmpty}
              variant="destructive"
            >
              {stopFullMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Stopping...
                </>
              ) : (
                <>
                  <Square className="mr-2 h-4 w-4" />
                  Stop Full Server
                </>
              )}
            </Button>
            <Button
              onClick={() => {
                setAddFormName('');
                setAddFormPath('');
                setAddFormPort('');
                setAddDialogOpen(true);
              }}
            >
              <Plus className="mr-2 h-4 w-4" />
              Add Process
            </Button>
          </div>
        )}
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : !processes || processes.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <AlertCircle className="mb-4 h-12 w-12 text-muted-foreground" />
            <p className="text-lg font-medium">No processes configured</p>
            {canManageServer ? (
              <>
                <p className="text-muted-foreground mb-4">
                  Click &apos;Add Process&apos; to get started
                </p>
                <Button
                  onClick={() => {
                    setAddFormName('');
                    setAddFormPath('');
                    setAddFormPort('');
                    setAddDialogOpen(true);
                  }}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Add Process
                </Button>
              </>
            ) : (
              <p className="text-muted-foreground">
                No processes configured. Contact an administrator to add
                processes.
              </p>
            )}
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Server Processes</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    {canManageServer && (
                      <TableHead className="w-12">Order</TableHead>
                    )}
                    <TableHead>Name</TableHead>
                    <TableHead>Path</TableHead>
                    <TableHead className="w-24">Port</TableHead>
                    <TableHead className="w-32">Status</TableHead>
                    <TableHead className="w-40">Uptime</TableHead>
                    {canManageServer && (
                      <TableHead className="w-64 text-right">Actions</TableHead>
                    )}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {processes.map((process, index) => {
                    const status = processStatuses.get(process.id);
                    const isRunning = status?.running === true;
                    const currentUptime =
                      status?.current_uptime_seconds !== undefined
                        ? status.current_uptime_seconds
                        : null;
                    const lastUptime =
                      status?.last_uptime_seconds !== undefined
                        ? status.last_uptime_seconds
                        : null;

                    return (
                      <TableRow key={process.id}>
                        {canManageServer && (
                          <TableCell>
                            <div className="flex flex-col gap-1">
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-6 w-6"
                                onClick={() => handleMoveUp(index)}
                                disabled={
                                  index === 0 || reorderMutation.isPending
                                }
                              >
                                <ChevronUp className="h-3 w-3" />
                              </Button>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-6 w-6"
                                onClick={() => handleMoveDown(index)}
                                disabled={
                                  index === processes.length - 1 ||
                                  reorderMutation.isPending
                                }
                              >
                                <ChevronDown className="h-3 w-3" />
                              </Button>
                            </div>
                          </TableCell>
                        )}
                        <TableCell className="font-medium">
                          {process.name}
                        </TableCell>
                        <TableCell className="font-mono text-xs">
                          {process.path}
                        </TableCell>
                        <TableCell>
                          {process.port ? (
                            <Badge variant="outline">{process.port}</Badge>
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </TableCell>
                        <TableCell>
                          {isRunning ? (
                            <Badge className="bg-green-600">Running</Badge>
                          ) : (
                            <Badge variant="destructive">Stopped</Badge>
                          )}
                        </TableCell>
                        <TableCell>
                          {isRunning && currentUptime !== null ? (
                            <span className="text-sm">
                              {formatUptime(currentUptime)}
                            </span>
                          ) : !isRunning && lastUptime !== null ? (
                            <span className="text-sm text-muted-foreground">
                              Last: {formatUptime(lastUptime)}
                            </span>
                          ) : (
                            <span className="text-sm text-muted-foreground">
                              -
                            </span>
                          )}
                        </TableCell>
                        {canManageServer && (
                          <TableCell className="text-right">
                            <div className="flex items-center justify-end gap-2">
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => openEditDialog(process)}
                              >
                                <Edit className="h-4 w-4" />
                              </Button>
                              {isRunning ? (
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() =>
                                    stopProcessMutation.mutate(process.id)
                                  }
                                  disabled={stopProcessMutation.isPending}
                                >
                                  <Square className="h-4 w-4" />
                                </Button>
                              ) : (
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() =>
                                    startProcessMutation.mutate(process.id)
                                  }
                                  disabled={startProcessMutation.isPending}
                                >
                                  <Play className="h-4 w-4" />
                                </Button>
                              )}
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => openDeleteDialog(process)}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </div>
                          </TableCell>
                        )}
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      )}

      <Dialog open={addDialogOpen} onOpenChange={setAddDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Process</DialogTitle>
            <DialogDescription>
              Add a new process to the server startup sequence
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="add-name">Name</Label>
              <Input
                id="add-name"
                value={addFormName}
                onChange={(e) => setAddFormName(e.target.value)}
                placeholder="Friendly name for this process"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="add-path">Path</Label>
              <Input
                id="add-path"
                value={addFormPath}
                onChange={(e) => setAddFormPath(e.target.value)}
                placeholder="Full path to executable or batch file"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="add-port">Port (Optional)</Label>
              <Input
                id="add-port"
                type="number"
                value={addFormPort}
                onChange={(e) => setAddFormPort(e.target.value)}
                placeholder="Port number to check"
                min={1}
                max={65535}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAddDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleAdd} disabled={createMutation.isPending}>
              {createMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Adding...
                </>
              ) : (
                'Add'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Process</DialogTitle>
            <DialogDescription>Update process configuration</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="edit-name">Name</Label>
              <Input
                id="edit-name"
                value={editFormName}
                onChange={(e) => setEditFormName(e.target.value)}
                placeholder="Friendly name for this process"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-path">Path</Label>
              <Input
                id="edit-path"
                value={editFormPath}
                onChange={(e) => setEditFormPath(e.target.value)}
                placeholder="Full path to executable or batch file"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-port">Port (Optional)</Label>
              <Input
                id="edit-port"
                type="number"
                value={editFormPort}
                onChange={(e) => setEditFormPort(e.target.value)}
                placeholder="Port number to check"
                min={1}
                max={65535}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleEdit} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Updating...
                </>
              ) : (
                'Update'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Process</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete &quot;{selectedProcess?.name}
              &quot;? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Deleting...
                </>
              ) : (
                'Delete'
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
