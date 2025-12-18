import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { useState } from 'react';
import {
  ArrowLeft,
  Loader2,
  AlertCircle,
  File as FileIcon,
  Calendar,
  Shield,
  HardDrive,
  Edit,
  RotateCcw,
  History,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
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
import {
  APIError,
  type FileNode,
  getFileTree,
  getTextFile,
  getNPCFile,
  getSpawnFile,
  getRevisionSummary,
  revertFile,
  getMaps,
} from '@/lib/api';
import { formatBytes, formatDate } from '@/lib/utils';
import { TextFileView } from '@/components/text-file-view';
import { NPCFileView } from '@/components/npc-file-view';
import { SpawnFileView } from '@/components/spawn-file-view';
import { toast } from 'sonner';
import { queryKeys } from '@/constants';
import { useMemo } from 'react';

interface FileViewProps {
  filePath: string;
}

export function FileView({ filePath }: FileViewProps) {
  const {
    data: fileTreeResponse,
    isLoading: fileTreeLoading,
    error: fileTreeError,
  } = useQuery({
    queryKey: queryKeys.fileTree(filePath),
    queryFn: () => {
      return getFileTree({ path: filePath });
    },
    enabled: !!filePath,
  });

  const fileNode: FileNode | undefined = fileTreeResponse?.file_tree;
  const fileType = fileNode?.file_type;
  const isEditable = fileNode?.is_editable ?? false;

  const {
    data: textFileData,
    isLoading: textFileLoading,
    error: textFileError,
  } = useQuery({
    queryKey: queryKeys.textFile(filePath),
    queryFn: () => {
      return getTextFile({ path: filePath });
    },
    enabled: !!filePath && fileType === 'text_file',
  });

  const {
    data: npcFileData,
    isLoading: npcFileLoading,
    error: npcFileError,
  } = useQuery({
    queryKey: queryKeys.npcFile(filePath),
    queryFn: () => {
      return getNPCFile({ path: filePath });
    },
    enabled: !!filePath && fileType === 'a3_npc_file',
  });

  const {
    data: spawnFileData,
    isLoading: spawnFileLoading,
    error: spawnFileError,
  } = useQuery({
    queryKey: queryKeys.spawnFile(filePath),
    queryFn: () => {
      return getSpawnFile({ path: filePath });
    },
    enabled: !!filePath && fileType === 'a3_spawn_file',
  });

  const { data: revisionSummary, isLoading: revisionSummaryLoading } = useQuery(
    {
      queryKey: queryKeys.revisionSummary(filePath),
      queryFn: () => {
        return getRevisionSummary({ path: filePath });
      },
      enabled: !!filePath && isEditable,
    },
  );

  const { data: maps } = useQuery({
    queryKey: queryKeys.maps,
    queryFn: () => getMaps(),
    enabled: fileType === 'a3_spawn_file',
  });

  const mapName = useMemo(() => {
    if (fileType !== 'a3_spawn_file' || !maps || !fileNode?.name) {
      return null;
    }

    const fileName = fileNode.name;
    if (!fileName.endsWith('.n_ndt')) {
      return null;
    }

    const mapId = parseInt(fileName.replace(/\.n_ndt$/, ''), 10);
    if (isNaN(mapId)) {
      return null;
    }

    const map = maps.find((m) => m.id === mapId);
    return map?.name || null;
  }, [fileType, maps, fileNode?.name]);

  const queryClient = useQueryClient();
  const [showRevertDialog, setShowRevertDialog] = useState(false);

  const revertMutation = useMutation({
    mutationFn: () => {
      return revertFile({ path: filePath });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.textFile(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.npcFile(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.spawnFile(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.fileTree(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.revisionSummary(filePath),
      });
      toast.success('File reverted to previous version');
      setShowRevertDialog(false);
    },
    onError: (error) => {
      const errorMessage =
        error instanceof APIError
          ? error.getErrorMessage()
          : error instanceof Error
            ? error.message
            : 'Failed to revert file';
      toast.error(errorMessage);
    },
  });

  const fileTreeErrorMessage =
    fileTreeError instanceof APIError
      ? fileTreeError.getErrorMessage()
      : fileTreeError instanceof Error
        ? fileTreeError.message
        : 'Failed to load file information';

  const fileContentErrorMessage =
    textFileError instanceof APIError
      ? textFileError.getErrorMessage()
      : npcFileError instanceof APIError
        ? npcFileError.getErrorMessage()
        : spawnFileError instanceof APIError
          ? spawnFileError.getErrorMessage()
          : textFileError instanceof Error
            ? textFileError.message
            : npcFileError instanceof Error
              ? npcFileError.message
              : spawnFileError instanceof Error
                ? spawnFileError.message
                : 'Failed to load file content';

  const getDirectoryPath = (filePath: string): string => {
    const isWindowsPath = /^[A-Za-z]:/.test(filePath);
    const separator = isWindowsPath ? '\\' : '/';
    const lastSeparatorIndex = Math.max(
      filePath.lastIndexOf(separator),
      filePath.lastIndexOf(isWindowsPath ? '/' : '\\'),
    );

    if (lastSeparatorIndex === -1) {
      return '';
    }

    return filePath.substring(0, lastSeparatorIndex);
  };

  const directoryPath = getDirectoryPath(filePath);

  return (
    <div className="p-4 lg:p-6">
      {/* Header */}
      <div className="mb-6">
        <Link
          to="/file"
          search={directoryPath ? { path: directoryPath } : {}}
          className="mb-4 inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to File Browser
        </Link>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {fileNode?.name || 'Loading...'}
              {mapName && (
                <span className="text-muted-foreground font-normal ml-2">
                  ({mapName})
                </span>
              )}
            </h1>
            <p className="text-muted-foreground mt-1">{filePath}</p>
          </div>
          <div className="flex items-center gap-2">
            {isEditable && revisionSummary && revisionSummary.count > 0 && (
              <Button
                variant="outline"
                onClick={() => setShowRevertDialog(true)}
                disabled={revertMutation.isPending}
              >
                {revertMutation.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <RotateCcw className="mr-2 h-4 w-4" />
                )}
                Revert
              </Button>
            )}
            {isEditable && (
              <Button variant="outline" asChild>
                <Link to="/file/edit" search={{ path: filePath }}>
                  <Edit className="mr-2 h-4 w-4" />
                  Edit
                </Link>
              </Button>
            )}
          </div>
        </div>
      </div>

      {/* File Tree Error */}
      {fileTreeError && (
        <Alert variant="destructive" className="mb-6">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{fileTreeErrorMessage}</AlertDescription>
        </Alert>
      )}

      {/* File Content Error */}
      {(textFileError || npcFileError || spawnFileError) && (
        <Alert variant="destructive" className="mb-6">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{fileContentErrorMessage}</AlertDescription>
        </Alert>
      )}

      {/* File Details Cards */}
      {fileNode && !fileTreeError && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileIcon className="h-5 w-5" />
              File Information
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <HardDrive className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium text-muted-foreground">
                    File Size
                  </span>
                </div>
                <div className="text-lg font-semibold">
                  {fileNode.file_size
                    ? formatBytes(fileNode.file_size)
                    : 'Unknown'}
                </div>
              </div>

              <div>
                <div className="flex items-center gap-2 mb-1">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium text-muted-foreground">
                    Last Modified
                  </span>
                </div>
                <div className="text-lg font-semibold">
                  {fileNode.last_modified
                    ? formatDate(fileNode.last_modified)
                    : 'Unknown'}
                </div>
              </div>

              <div>
                <div className="flex items-center gap-2 mb-1">
                  <Shield className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium text-muted-foreground">
                    Permissions
                  </span>
                </div>
                <div className="text-lg font-semibold">
                  {fileNode.permissions}
                </div>
              </div>

              <div>
                <div className="flex items-center gap-2 mb-1">
                  <FileIcon className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium text-muted-foreground">
                    File Type
                  </span>
                </div>
                <div className="text-lg font-semibold">
                  {fileNode.file_type || 'Unknown'}
                </div>
              </div>

              {isEditable && (
                <>
                  <div>
                    <div className="flex items-center gap-2 mb-1">
                      <History className="h-4 w-4 text-muted-foreground" />
                      <span className="text-sm font-medium text-muted-foreground">
                        Revision Count
                      </span>
                    </div>
                    <div className="text-lg font-semibold">
                      {revisionSummaryLoading ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        (revisionSummary?.count ?? 0)
                      )}
                    </div>
                  </div>

                  {revisionSummary && revisionSummary.last_revision_at && (
                    <div>
                      <div className="flex items-center gap-2 mb-1">
                        <History className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm font-medium text-muted-foreground">
                          Last Revision
                        </span>
                      </div>
                      <div className="text-lg font-semibold">
                        {formatDate(
                          new Date(revisionSummary.last_revision_at * 1000),
                        )}
                      </div>
                    </div>
                  )}
                </>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* File Content */}
      {fileTreeLoading && (
        <div className="flex h-96 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      )}

      {!fileTreeLoading && fileNode && !fileTreeError && (
        <>
          {fileType === 'text_file' && (
            <>
              {textFileLoading && (
                <div className="flex h-96 items-center justify-center">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              )}
              {textFileData && !textFileError && (
                <div>
                  <h2 className="text-xl font-semibold mb-4">File Content</h2>
                  <TextFileView content={textFileData.content} />
                </div>
              )}
            </>
          )}

          {fileType === 'a3_npc_file' && (
            <>
              {npcFileLoading && (
                <div className="flex h-96 items-center justify-center">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              )}
              {npcFileData && !npcFileError && (
                <div>
                  <h2 className="text-xl font-semibold mb-4">NPC Data</h2>
                  <NPCFileView data={npcFileData} />
                </div>
              )}
            </>
          )}

          {fileType === 'a3_spawn_file' && (
            <>
              {spawnFileLoading && (
                <div className="flex h-96 items-center justify-center">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              )}
              {spawnFileData && !spawnFileError && (
                <div>
                  <h2 className="text-xl font-semibold mb-4">Spawn Data</h2>
                  <SpawnFileView data={spawnFileData} />
                </div>
              )}
            </>
          )}

          {fileType &&
            fileType !== 'text_file' &&
            fileType !== 'a3_npc_file' &&
            fileType !== 'a3_spawn_file' &&
            !textFileLoading &&
            !npcFileLoading &&
            !spawnFileLoading && (
              <Card>
                <CardContent className="p-6 text-center text-muted-foreground">
                  File type &quot;{fileType}&quot; is not yet supported for
                  viewing.
                </CardContent>
              </Card>
            )}
        </>
      )}

      {/* Revert Confirmation Dialog */}
      <AlertDialog open={showRevertDialog} onOpenChange={setShowRevertDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revert File</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to revert this file to the previous version?
              This action cannot be undone and will replace the current file
              content with the previous revision.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={revertMutation.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => revertMutation.mutate()}
              disabled={revertMutation.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {revertMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Revert
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
