import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import {
  ArrowLeft,
  File as FileIcon,
  Calendar,
  Shield,
  Loader2,
  HardDrive,
  X,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  APIError,
  type FileNode,
  getFileTree,
  getTextFile,
  getNPCFile,
  getSpawnFile,
} from '@/lib/api';
import { formatBytes, formatDate } from '@/lib/utils';
import { TextFileEdit } from './text-file-edit';
import { NPCFileEdit } from './npc-file-edit';
import { SpawnFileEdit } from './spawn-file-edit';

interface FileEditProps {
  filePath: string;
}

export function FileEdit({ filePath }: FileEditProps) {
  const {
    data: fileTreeResponse,
    isLoading: fileTreeLoading,
    error: fileTreeError,
  } = useQuery({
    queryKey: ['file-tree', filePath],
    queryFn: () => {
      return getFileTree({ path: filePath });
    },
    enabled: !!filePath,
  });

  const fileNode: FileNode | undefined = fileTreeResponse?.file_tree;
  const fileType = fileNode?.file_type;

  const {
    data: textFileData,
    isLoading: textFileLoading,
    error: textFileError,
  } = useQuery({
    queryKey: ['text-file', filePath],
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
    queryKey: ['npc-file', filePath],
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
    queryKey: ['spawn-file', filePath],
    queryFn: () => {
      return getSpawnFile({ path: filePath });
    },
    enabled: !!filePath && fileType === 'a3_spawn_file',
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

  return (
    <div className="p-4 lg:p-6">
      <div className="mb-6">
        <Link
          to="/file/view"
          search={{ path: filePath }}
          className="mb-4 inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to File View
        </Link>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {fileNode?.name || 'Loading...'}
            </h1>
            <p className="text-muted-foreground mt-1">{filePath}</p>
          </div>
          <Button variant="outline" asChild>
            <Link to="/file/view" search={{ path: filePath }}>
              <span className="flex items-center gap-1.5">
                <X className="h-4 w-4" />
                Cancel
              </span>
            </Link>
          </Button>
        </div>
      </div>

      {fileTreeError && (
        <Alert variant="destructive" className="mb-6">
          <AlertDescription>{fileTreeErrorMessage}</AlertDescription>
        </Alert>
      )}

      {(textFileError || npcFileError || spawnFileError) && (
        <Alert variant="destructive" className="mb-6">
          <AlertDescription>{fileContentErrorMessage}</AlertDescription>
        </Alert>
      )}

      {fileNode && !fileTreeError && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileIcon className="h-5 w-5" />
              File Information
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
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
            </div>
          </CardContent>
        </Card>
      )}

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
                <TextFileEdit
                  filePath={filePath}
                  defaultContent={textFileData.content}
                />
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
                <NPCFileEdit filePath={filePath} defaultData={npcFileData} />
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
                <SpawnFileEdit
                  filePath={filePath}
                  defaultData={spawnFileData}
                />
              )}
            </>
          )}
          {fileType &&
            fileType !== 'text_file' &&
            fileType !== 'a3_npc_file' &&
            fileType !== 'a3_spawn_file' && (
              <Card>
                <CardContent className="p-6 text-center text-muted-foreground">
                  File type "{fileType}" is not yet supported for editing.
                </CardContent>
              </Card>
            )}
        </>
      )}
    </div>
  );
}
