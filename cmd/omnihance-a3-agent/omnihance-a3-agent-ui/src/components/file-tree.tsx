import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import {
  Folder,
  File,
  ChevronRight,
  Loader2,
  AlertCircle,
  Eye,
} from 'lucide-react';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from '@/components/ui/context-menu';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  getFileTree,
  createServerProcess,
  getServerProcesses,
  deleteServerProcess,
  getProcessStatus,
  APIError,
  type FileNode,
  type FileTreeResponse,
  type ServerProcess,
  type ProcessStatus,
} from '@/lib/api';
import { formatBytes, formatDate, cn } from '@/lib/utils';
import { queryKeys } from '@/constants';
import { toast } from 'sonner';
import { usePermissions } from '@/hooks/use-permissions';

interface FileTreeProps {
  initialPath?: string;
}

interface ProcessContextMenuWrapperProps {
  existingProcess: ServerProcess | undefined;
  fileItem: React.ReactElement;
  onAdd: () => void;
  onRemove: () => void;
}

function ProcessContextMenuWrapper({
  existingProcess,
  fileItem,
  onAdd,
  onRemove,
}: ProcessContextMenuWrapperProps) {
  const { data: processStatus } = useQuery<ProcessStatus>({
    queryKey: [...queryKeys.serverProcesses, 'status', existingProcess?.id],
    queryFn: () => getProcessStatus(existingProcess!.id),
    enabled: !!existingProcess,
    refetchInterval: 3000,
  });

  const isRunning = processStatus?.running === true;

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>{fileItem}</ContextMenuTrigger>
      <ContextMenuContent>
        {existingProcess ? (
          <ContextMenuItem
            onClick={onRemove}
            variant="destructive"
            disabled={isRunning}
          >
            {isRunning
              ? 'Remove from Server Startup List (Stop process first)'
              : 'Remove from Server Startup List'}
          </ContextMenuItem>
        ) : (
          <ContextMenuItem onClick={onAdd}>
            Add to Server Startup List
          </ContextMenuItem>
        )}
      </ContextMenuContent>
    </ContextMenu>
  );
}

export function FileTree({ initialPath }: FileTreeProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { hasPermission } = usePermissions();
  const canManageServer = hasPermission('manage_server');
  const [internalPath, setInternalPath] = useState<string | null>(null);
  const [showDotfiles, setShowDotfiles] = useState(false);
  const [contextMenuFile, setContextMenuFile] = useState<FileNode | null>(null);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [addDialogName, setAddDialogName] = useState('');
  const [addDialogPort, setAddDialogPort] = useState('');

  const rawCurrentPath =
    internalPath !== null ? internalPath : (initialPath ?? '');

  const normalizePathForQuery = (path: string): string => {
    if (!path || path === '') {
      return '';
    }

    const looksLikeWindowsPath = /^[A-Za-z]:/.test(path);

    if (looksLikeWindowsPath) {
      return path;
    }

    if (!path.startsWith('/')) {
      return '/' + path;
    }

    return path;
  };

  const queryPath = normalizePathForQuery(rawCurrentPath);

  const {
    data: fileTreeResponse,
    isLoading,
    error,
  } = useQuery<FileTreeResponse>({
    queryKey: queryKeys.fileTree(queryPath, showDotfiles),
    queryFn: async () => {
      let pathToUse = queryPath;

      if (pathToUse && /^[A-Za-z]:$/.test(pathToUse)) {
        pathToUse = `${pathToUse}\\`;
      }

      return getFileTree(
        pathToUse
          ? { path: pathToUse, show_dotfiles: showDotfiles }
          : { show_dotfiles: showDotfiles },
      );
    },
  });

  const { data: serverProcesses } = useQuery<ServerProcess[]>({
    queryKey: queryKeys.serverProcesses,
    queryFn: getServerProcesses,
  });

  const fileNode = fileTreeResponse?.file_tree;
  const files = fileNode?.children || [];
  const os = fileTreeResponse?.os?.toLowerCase() || '';
  const isWindows = os === 'windows';

  const isWindowsPath = (path: string): boolean => {
    return isWindows && /^[A-Za-z]:/.test(path);
  };

  const normalizePath = (path: string, isWindowsSystem: boolean): string => {
    if (!path || path === '') {
      return '';
    }

    const looksLikeWindowsPath = /^[A-Za-z]:/.test(path);

    if (looksLikeWindowsPath) {
      return path;
    }

    if (!isWindowsSystem && !path.startsWith('/')) {
      return '/' + path;
    }

    return path;
  };

  const currentPath = normalizePath(rawCurrentPath, isWindows);
  const effectivePath = currentPath;
  const pathParts = effectivePath
    ? isWindowsPath(effectivePath)
      ? effectivePath.split(/[/\\]/).filter(Boolean)
      : effectivePath.split('/').filter(Boolean)
    : [];

  const navigateTo = (path: string) => {
    const normalizedPath = normalizePath(path, isWindows);
    setInternalPath(normalizedPath);
    navigate({
      to: '/file',
      search: normalizedPath ? { path: normalizedPath } : {},
      replace: true,
      resetScroll: false,
    });
  };

  const handleItemClick = (item: FileNode) => {
    if (item.kind === 'directory') {
      let newPath: string;
      if (!currentPath || currentPath === '') {
        if (isWindows && /^[A-Za-z]:$/.test(item.name)) {
          newPath = `${item.name}\\`;
        } else {
          newPath = '/' + item.name;
        }
      } else {
        const isWindowsCurrentPath = isWindowsPath(currentPath);
        const separator = isWindowsCurrentPath ? '\\' : '/';
        let basePath = currentPath;

        if (isWindowsCurrentPath) {
          if (basePath.endsWith('\\')) {
            basePath = basePath.slice(0, -1);
          }
          if (basePath.match(/^[A-Za-z]:$/)) {
            newPath = `${basePath}${separator}${item.name}`;
          } else {
            newPath = `${basePath}${separator}${item.name}`;
          }
        } else {
          if (basePath.endsWith('/')) {
            basePath = basePath.slice(0, -1);
          }
          newPath = `${basePath}${separator}${item.name}`;
        }
      }
      navigateTo(normalizePath(newPath, isWindows));
    }
  };

  const sortedFiles = [...(files || [])].sort((a, b) => {
    if (a.kind === 'directory' && b.kind === 'file') return -1;
    if (a.kind === 'file' && b.kind === 'directory') return 1;
    return a.name.localeCompare(b.name);
  });

  const getFullPath = (item: FileNode): string => {
    if (!currentPath || currentPath === '') {
      const path =
        isWindows && /^[A-Za-z]:$/.test(item.name)
          ? `${item.name}\\`
          : '/' + item.name;
      return normalizePath(path, isWindows);
    }

    const isWindowsCurrentPath = isWindowsPath(currentPath);
    const separator = isWindowsCurrentPath ? '\\' : '/';
    let basePath = currentPath;

    if (isWindowsCurrentPath) {
      if (basePath.endsWith('\\')) {
        basePath = basePath.slice(0, -1);
      }
    } else {
      if (basePath.endsWith('/')) {
        basePath = basePath.slice(0, -1);
      }
    }

    const fullPath = `${basePath}${separator}${item.name}`;
    return normalizePath(fullPath, isWindows);
  };

  const isExecutableOrBatch = (item: FileNode): boolean => {
    if (item.kind !== 'file') {
      return false;
    }
    const ext = item.file_extension?.toLowerCase() || '';
    return ext === '.exe' || ext === '.bat' || ext === '.cmd';
  };

  const normalizePathForComparison = (path: string): string => {
    return path.replace(/\\/g, '/').toLowerCase().trim();
  };

  const findProcessByPath = (filePath: string): ServerProcess | undefined => {
    if (!serverProcesses) {
      return undefined;
    }
    const normalizedFilePath = normalizePathForComparison(filePath);
    return serverProcesses.find(
      (proc) => normalizePathForComparison(proc.path) === normalizedFilePath,
    );
  };

  const handleAddToServer = (item: FileNode) => {
    setContextMenuFile(item);
    setAddDialogName(item.name.replace(/\.(exe|bat|cmd)$/i, ''));
    setAddDialogPort('');
    setShowAddDialog(true);
  };

  const handleRemoveFromServer = (item: FileNode) => {
    const fullPath = getFullPath(item);
    const process = findProcessByPath(fullPath);
    if (process) {
      removeFromServerMutation.mutate(process.id);
    }
  };

  const addToServerMutation = useMutation({
    mutationFn: async (data: { name: string; path: string; port?: number }) => {
      return createServerProcess(data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [queryKeys.serverProcesses] });
      toast.success('Process added to server startup list');
      setShowAddDialog(false);
      setContextMenuFile(null);
      setAddDialogName('');
      setAddDialogPort('');
    },
    onError: (error: APIError) => {
      toast.error(error.getErrorMessage());
    },
  });

  const removeFromServerMutation = useMutation({
    mutationFn: async (id: number) => {
      return deleteServerProcess(id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [queryKeys.serverProcesses] });
      toast.success('Process removed from server startup list');
    },
    onError: (error: APIError) => {
      toast.error(error.getErrorMessage());
    },
  });

  const handleAddDialogSubmit = () => {
    if (!contextMenuFile || !addDialogName.trim()) {
      toast.error('Name is required');
      return;
    }

    const fullPath = getFullPath(contextMenuFile);
    const port = addDialogPort.trim()
      ? parseInt(addDialogPort.trim(), 10)
      : undefined;

    if (port !== undefined && (isNaN(port) || port < 1 || port > 65535)) {
      toast.error('Port must be a number between 1 and 65535');
      return;
    }

    addToServerMutation.mutate({
      name: addDialogName.trim(),
      path: fullPath,
      port,
    });
  };

  if (error) {
    const errorMessage =
      error instanceof APIError
        ? error.getErrorMessage()
        : error instanceof Error
          ? error.message
          : 'Failed to load file tree';
    return (
      <Alert variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertDescription>{errorMessage}</AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      {/* Breadcrumb Navigation and Show Dotfiles Checkbox */}
      <div className="mb-4 flex shrink-0 items-center justify-between gap-4">
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem>
              <BreadcrumbLink
                className="cursor-pointer"
                onClick={() => navigateTo('')}
              >
                Root
              </BreadcrumbLink>
            </BreadcrumbItem>
            {pathParts.map((part, index) => {
              const isWindowsCurrentPath = isWindowsPath(effectivePath);
              const separator = isWindowsCurrentPath ? '\\' : '/';
              const pathPartsToJoin = pathParts.slice(0, index + 1);
              const rawPath = pathPartsToJoin.join(separator);
              const path = normalizePath(rawPath, isWindows);
              const isLast = index === pathParts.length - 1;

              return (
                <span key={path} className="flex items-center">
                  <BreadcrumbSeparator />
                  <BreadcrumbItem>
                    {isLast ? (
                      <BreadcrumbPage>{part}</BreadcrumbPage>
                    ) : (
                      <BreadcrumbLink
                        className="cursor-pointer"
                        onClick={() => navigateTo(path)}
                      >
                        {part}
                      </BreadcrumbLink>
                    )}
                  </BreadcrumbItem>
                </span>
              );
            })}
          </BreadcrumbList>
        </Breadcrumb>
        <div className="flex items-center gap-2 shrink-0">
          <Checkbox
            id="show-dotfiles"
            checked={showDotfiles}
            onCheckedChange={(checked) => setShowDotfiles(checked === true)}
          />
          <Label
            htmlFor="show-dotfiles"
            className="text-sm font-normal cursor-pointer whitespace-nowrap"
          >
            Show dotfiles
          </Label>
        </div>
      </div>

      {/* File List */}
      <ScrollArea className="min-h-0 flex-1 rounded-lg border">
        {isLoading ? (
          <div className="flex h-full items-center justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : sortedFiles.length === 0 ? (
          <div className="flex h-full items-center justify-center text-muted-foreground">
            This folder is empty
          </div>
        ) : (
          <div className="divide-y">
            {/* Header */}
            <div className="grid grid-cols-12 gap-4 bg-muted/50 px-4 py-2 text-xs font-medium text-muted-foreground">
              <div className="col-span-5 sm:col-span-4">Name</div>
              <div className="col-span-2 sm:col-span-2 text-right">Size</div>
              <div className="col-span-3 hidden sm:block">Modified</div>
              <div className="col-span-2 sm:col-span-3 text-right">Actions</div>
            </div>

            {/* Files */}
            {sortedFiles.map((item) => {
              const isExecutable = isExecutableOrBatch(item);
              const fullPath = isExecutable ? getFullPath(item) : '';
              const existingProcess = isExecutable
                ? findProcessByPath(fullPath)
                : undefined;
              const fileItem = (
                <div
                  key={item.id}
                  className={cn(
                    'grid grid-cols-12 gap-4 px-4 py-3 text-sm transition-colors',
                    (item.kind === 'directory' ||
                      (item.kind === 'file' && item.is_viewable)) &&
                      'cursor-pointer hover:bg-accent',
                  )}
                  onClick={() => {
                    if (item.kind === 'directory') {
                      handleItemClick(item);
                    } else if (item.kind === 'file' && item.is_viewable) {
                      navigate({
                        to: '/file/view',
                        search: { path: getFullPath(item) },
                      });
                    }
                  }}
                  onKeyDown={(e) => {
                    if (item.kind === 'file' && item.is_viewable) {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        navigate({
                          to: '/file/view',
                          search: { path: getFullPath(item) },
                        });
                      }
                    }
                  }}
                  tabIndex={
                    item.kind === 'file' && item.is_viewable ? 0 : undefined
                  }
                  role={
                    item.kind === 'file' && item.is_viewable
                      ? 'button'
                      : undefined
                  }
                  aria-label={
                    item.kind === 'file' && item.is_viewable
                      ? `View ${item.name}`
                      : undefined
                  }
                >
                  <div className="col-span-5 sm:col-span-4 flex items-center gap-3 truncate">
                    {item.kind === 'directory' ? (
                      <Folder className="h-4 w-4 shrink-0 text-primary" />
                    ) : (
                      <File className="h-4 w-4 shrink-0 text-muted-foreground" />
                    )}
                    <span className="truncate">{item.name}</span>
                    {item.kind === 'directory' && (
                      <ChevronRight className="ml-auto h-4 w-4 shrink-0 text-muted-foreground" />
                    )}
                  </div>
                  <div className="col-span-2 sm:col-span-2 text-right text-muted-foreground">
                    {item.kind === 'file'
                      ? item.file_size
                        ? formatBytes(item.file_size)
                        : 'Unknown'
                      : ''}
                  </div>
                  <div className="col-span-3 hidden text-muted-foreground sm:block">
                    {item.last_modified
                      ? formatDate(item.last_modified)
                      : 'Unknown'}
                  </div>
                  <div className="col-span-2 sm:col-span-3 flex items-center justify-end gap-2">
                    {item.kind === 'file' && item.is_viewable && (
                      <Link
                        to="/file/view"
                        search={{ path: getFullPath(item) }}
                        className="p-1.5 rounded-md hover:bg-accent transition-colors"
                        aria-label="View file"
                        tabIndex={0}
                        onClick={(e) => {
                          e.stopPropagation();
                        }}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault();
                            e.stopPropagation();
                          }
                        }}
                      >
                        <Eye className="h-4 w-4 text-muted-foreground hover:text-foreground" />
                      </Link>
                    )}
                  </div>
                </div>
              );

              if (isExecutable && canManageServer) {
                return (
                  <ProcessContextMenuWrapper
                    key={item.id}
                    existingProcess={existingProcess}
                    fileItem={fileItem}
                    onAdd={() => handleAddToServer(item)}
                    onRemove={() => handleRemoveFromServer(item)}
                  />
                );
              }

              return fileItem;
            })}
          </div>
        )}
      </ScrollArea>

      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add to Server Startup List</DialogTitle>
            <DialogDescription>
              Add this executable to the server startup sequence
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                value={addDialogName}
                onChange={(e) => setAddDialogName(e.target.value)}
                placeholder="Friendly name for this process"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="port">Port (Optional)</Label>
              <Input
                id="port"
                type="number"
                value={addDialogPort}
                onChange={(e) => setAddDialogPort(e.target.value)}
                placeholder="Port number to check"
                min={1}
                max={65535}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowAddDialog(false);
                setContextMenuFile(null);
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={handleAddDialogSubmit}
              disabled={addToServerMutation.isPending || !addDialogName.trim()}
            >
              {addToServerMutation.isPending ? (
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
    </div>
  );
}
