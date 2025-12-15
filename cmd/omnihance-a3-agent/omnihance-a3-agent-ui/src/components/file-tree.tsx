import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
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
  getFileTree,
  APIError,
  type FileNode,
  type FileTreeResponse,
} from '@/lib/api';
import { formatBytes, formatDate, cn } from '@/lib/utils';

interface FileTreeProps {
  initialPath?: string;
}

export function FileTree({ initialPath }: FileTreeProps) {
  const navigate = useNavigate();
  const [internalPath, setInternalPath] = useState<string | null>(null);
  const [showDotfiles, setShowDotfiles] = useState(false);

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
    queryKey: ['file-tree', queryPath, showDotfiles],
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
            {sortedFiles.map((item) => (
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
            ))}
          </div>
        )}
      </ScrollArea>
    </div>
  );
}
