import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  AlertCircle,
  CheckCircle2,
  FolderUp,
  Loader2,
  Upload,
  X,
} from 'lucide-react';
import { toast } from 'sonner';
import {
  cancelFileUpload,
  completeFileUpload,
  createFileUpload,
  heartbeatFileUpload,
  uploadFileChunk,
  APIError,
  type CreateFileUploadResponse,
} from '@/lib/api';
import { formatBytes, cn } from '@/lib/util';
import { getUploadSizeError } from '@/lib/upload-validation';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

const uploadChunkSize = 4 * 1024 * 1024;
const heartbeatIntervalMs = 30 * 1000;
const maxChunkAttempts = 4;

type UploadTaskStatus =
  | 'queued'
  | 'uploading'
  | 'verifying'
  | 'complete'
  | 'failed'
  | 'cancelled';

type UploadSource = {
  file: File;
  relativePath: string;
};

type UploadTask = {
  id: string;
  destinationPath: string;
  files: UploadSource[];
  status: UploadTaskStatus;
  uploadedBytes: number;
  totalBytes: number;
  currentFileName: string | null;
  uploadId: string | null;
  error: string | null;
  createdAt: number;
};

type UseFileBrowserUploaderParams = {
  destinationPath: string;
  canUpload: boolean;
  maxFileUploadSizeBytes?: number;
  onUploaded: () => void;
};

type BrowserFileEntry = FileSystemFileEntry & {
  file: (
    successCallback: (file: File) => void,
    errorCallback?: (error: DOMException) => void,
  ) => void;
};

type BrowserDirectoryEntry = FileSystemDirectoryEntry & {
  createReader: () => FileSystemDirectoryReader;
};

type BrowserEntry = BrowserFileEntry | BrowserDirectoryEntry;

type DataTransferItemWithEntry = DataTransferItem & {
  webkitGetAsEntry?: () => BrowserEntry | null;
};

export function useFileBrowserUploader({
  destinationPath,
  canUpload,
  maxFileUploadSizeBytes,
  onUploaded,
}: UseFileBrowserUploaderParams) {
  const [tasks, setTasks] = useState<UploadTask[]>([]);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const directoryInputRef = useRef<HTMLInputElement>(null);
  const workerRunningRef = useRef(false);
  const cancelledTaskIdsRef = useRef(new Set<string>());
  const activeAbortRef = useRef<AbortController | null>(null);
  const activeUploadIdsRef = useRef(new Map<string, string>());
  const currentTaskRef = useRef<UploadTask | null>(null);

  useEffect(() => {
    directoryInputRef.current?.setAttribute('webkitdirectory', '');
  }, []);

  const enqueueUpload = useCallback(
    (sources: UploadSource[]) => {
      if (!canUpload || !destinationPath) {
        toast.error('Open a folder before uploading');
        return;
      }

      const uniqueSources = dedupeUploadSources(sources);
      if (uniqueSources.length === 0) {
        toast.error('No files found to upload');
        return;
      }

      const oversizedSource = findOversizedUploadSource(
        uniqueSources,
        maxFileUploadSizeBytes,
      );
      if (oversizedSource) {
        toast.error(oversizedSource);
        return;
      }

      const task: UploadTask = {
        id: crypto.randomUUID(),
        destinationPath,
        files: uniqueSources,
        status: 'queued',
        uploadedBytes: 0,
        totalBytes: uniqueSources.reduce((total, source) => {
          return total + source.file.size;
        }, 0),
        currentFileName: null,
        uploadId: null,
        error: null,
        createdAt: Date.now(),
      };

      setTasks((current) => [...current, task]);
      setIsDialogOpen(false);
      toast.success(
        uniqueSources.length === 1
          ? `Queued ${uniqueSources[0].file.name}`
          : `Queued ${uniqueSources.length} files`,
      );
    },
    [canUpload, destinationPath, maxFileUploadSizeBytes],
  );

  const cancelTask = useCallback((taskId: string) => {
    cancelledTaskIdsRef.current.add(taskId);
    if (currentTaskRef.current?.id === taskId) {
      activeAbortRef.current?.abort();
    }

    const uploadId = activeUploadIdsRef.current.get(taskId);
    if (uploadId) {
      void cancelFileUpload(uploadId).catch(() => undefined);
    }

    setTasks((current) =>
      current.map((task) => {
        if (task.id !== taskId) {
          return task;
        }

        return {
          ...task,
          status: 'cancelled',
          error: null,
        };
      }),
    );
  }, []);

  const clearTask = useCallback((taskId: string) => {
    setTasks((current) => current.filter((task) => task.id !== taskId));
    cancelledTaskIdsRef.current.delete(taskId);
    activeUploadIdsRef.current.delete(taskId);
  }, []);

  const openUploadDialog = useCallback(() => {
    if (!canUpload) {
      toast.error('Open a folder before uploading');
      return;
    }

    setIsDialogOpen(true);
  }, [canUpload]);

  const handleFilesSelected = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const selectedFiles = Array.from(event.target.files ?? []);
      enqueueUpload(
        selectedFiles.map((file) => ({
          file,
          relativePath: file.name,
        })),
      );
      event.target.value = '';
    },
    [enqueueUpload],
  );

  const handleDirectorySelected = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const selectedFiles = Array.from(event.target.files ?? []);
      enqueueUpload(
        selectedFiles.map((file) => ({
          file,
          relativePath:
            (file as File & { webkitRelativePath?: string })
              .webkitRelativePath || file.name,
        })),
      );
      event.target.value = '';
    },
    [enqueueUpload],
  );

  const enqueueDroppedDataTransfer = useCallback(
    async (dataTransfer: DataTransfer) => {
      try {
        const sources = await uploadSourcesFromDataTransfer(dataTransfer);
        enqueueUpload(sources);
      } catch (error) {
        toast.error(
          error instanceof Error
            ? error.message
            : 'Failed to read dropped files',
        );
      }
    },
    [enqueueUpload],
  );

  const handleDrop = useCallback(
    async (event: React.DragEvent<HTMLDivElement>) => {
      if (!canUpload) {
        return;
      }

      event.preventDefault();
      event.stopPropagation();
      setIsDragging(false);

      await enqueueDroppedDataTransfer(event.dataTransfer);
    },
    [canUpload, enqueueDroppedDataTransfer],
  );

  const dropHandlers = useMemo(
    () => ({
      onDragOver: (event: React.DragEvent<HTMLDivElement>) => {
        if (!canUpload || !hasFileDrag(event.dataTransfer)) {
          return;
        }

        event.preventDefault();
        event.stopPropagation();
        setIsDragging(true);
      },
      onDragLeave: (event: React.DragEvent<HTMLDivElement>) => {
        if (!canUpload) {
          return;
        }

        event.preventDefault();
        event.stopPropagation();
        if (event.currentTarget.contains(event.relatedTarget as Node | null)) {
          return;
        }

        setIsDragging(false);
      },
      onDrop: handleDrop,
    }),
    [canUpload, handleDrop],
  );

  const runNextQueuedTask = useCallback(async () => {
    if (workerRunningRef.current) {
      return;
    }

    const nextTask = tasks.find((task) => task.status === 'queued');
    if (!nextTask) {
      return;
    }

    workerRunningRef.current = true;
    currentTaskRef.current = nextTask;

    try {
      await uploadTask(
        nextTask,
        setTasks,
        cancelledTaskIdsRef,
        activeAbortRef,
        activeUploadIdsRef,
      );
      onUploaded();
    } catch (error) {
      if (!cancelledTaskIdsRef.current.has(nextTask.id)) {
        setTasks((current) =>
          current.map((task) => {
            if (task.id !== nextTask.id) {
              return task;
            }

            return {
              ...task,
              status: 'failed',
              error: uploadErrorMessage(error),
            };
          }),
        );
      }
    } finally {
      activeAbortRef.current = null;
      currentTaskRef.current = null;
      workerRunningRef.current = false;
    }
  }, [onUploaded, tasks]);

  useEffect(() => {
    void runNextQueuedTask();
  }, [runNextQueuedTask, tasks]);

  const activeTasks = tasks.filter((task) => task.status !== 'complete');
  const completedTasks = tasks.filter((task) => task.status === 'complete');
  const visibleTasks = [...activeTasks, ...completedTasks].slice(-6);

  const dialog = (
    <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Upload</DialogTitle>
          <DialogDescription>
            Add files or a folder to the current directory.
          </DialogDescription>
        </DialogHeader>
        <div
          className={cn(
            'flex min-h-52 flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 text-center transition-colors',
            'border-muted-foreground/25 hover:border-muted-foreground/50',
          )}
          onDragOver={(event) => {
            event.preventDefault();
            event.stopPropagation();
          }}
          onDrop={async (event) => {
            event.preventDefault();
            event.stopPropagation();
            await enqueueDroppedDataTransfer(event.dataTransfer);
          }}
        >
          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={handleFilesSelected}
            aria-label="Select files"
            tabIndex={-1}
          />
          <input
            ref={directoryInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={handleDirectorySelected}
            aria-label="Select folder"
            tabIndex={-1}
          />
          <FolderUp className="mb-4 h-11 w-11 text-muted-foreground" />
          <p className="text-sm font-medium">Drop files here</p>
          <div className="mt-5 flex flex-wrap justify-center gap-2">
            <Button type="button" onClick={() => fileInputRef.current?.click()}>
              <Upload className="h-4 w-4" />
              Select files
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => directoryInputRef.current?.click()}
            >
              <FolderUp className="h-4 w-4" />
              Select folder
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );

  const progressPanel =
    visibleTasks.length > 0 ? (
      <div className="fixed right-4 bottom-4 z-50 w-[min(24rem,calc(100vw-2rem))] overflow-hidden rounded-lg border bg-background shadow-lg">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <div>
            <p className="text-sm font-medium">Uploads</p>
            <p className="text-xs text-muted-foreground">
              {activeTasks.length} active, {completedTasks.length} complete
            </p>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => {
              setTasks((current) =>
                current.filter((task) => task.status !== 'complete'),
              );
            }}
            aria-label="Clear completed uploads"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
        <div className="max-h-80 divide-y overflow-y-auto">
          {visibleTasks.map((task) => (
            <UploadTaskRow
              key={task.id}
              task={task}
              onCancel={() => cancelTask(task.id)}
              onClear={() => clearTask(task.id)}
            />
          ))}
        </div>
      </div>
    ) : null;

  return {
    canUpload,
    isDragging,
    openUploadDialog,
    dropHandlers,
    dialog,
    progressPanel,
  };
}

function UploadTaskRow({
  task,
  onCancel,
  onClear,
}: {
  task: UploadTask;
  onCancel: () => void;
  onClear: () => void;
}) {
  const progress =
    task.totalBytes > 0
      ? Math.min(100, Math.round((task.uploadedBytes / task.totalBytes) * 100))
      : task.status === 'complete'
        ? 100
        : 0;
  const label =
    task.files.length === 1
      ? task.files[0].file.name
      : `${task.files.length} files`;
  const canCancel =
    task.status === 'queued' ||
    task.status === 'uploading' ||
    task.status === 'verifying';
  const canClear =
    task.status === 'complete' ||
    task.status === 'failed' ||
    task.status === 'cancelled';

  return (
    <div className="space-y-2 px-4 py-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">{label}</p>
          <p className="truncate text-xs text-muted-foreground">
            {task.status === 'queued'
              ? 'Queued'
              : task.status === 'uploading'
                ? task.currentFileName || 'Uploading'
                : task.status === 'verifying'
                  ? 'Verifying'
                  : task.status === 'failed'
                    ? task.error || 'Upload failed'
                    : task.status === 'cancelled'
                      ? 'Cancelled'
                      : 'Complete'}
          </p>
        </div>
        {task.status === 'complete' && (
          <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-green-600" />
        )}
        {task.status === 'failed' && (
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
        )}
        {(task.status === 'queued' ||
          task.status === 'uploading' ||
          task.status === 'verifying') && (
          <Loader2 className="mt-0.5 h-4 w-4 shrink-0 animate-spin text-muted-foreground" />
        )}
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-muted">
        <div
          className={cn(
            'h-full rounded-full transition-all',
            task.status === 'failed'
              ? 'bg-destructive'
              : task.status === 'cancelled'
                ? 'bg-muted-foreground'
                : 'bg-primary',
          )}
          style={{ width: `${progress}%` }}
          role="progressbar"
          aria-valuenow={progress}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label={`Upload progress for ${label}`}
        />
      </div>
      <div className="flex items-center justify-between gap-3 text-xs text-muted-foreground">
        <span>
          {formatBytes(task.uploadedBytes)} / {formatBytes(task.totalBytes)}
        </span>
        {canCancel && (
          <button
            type="button"
            className="font-medium text-foreground hover:underline"
            onClick={onCancel}
          >
            Cancel
          </button>
        )}
        {canClear && (
          <button
            type="button"
            className="font-medium text-foreground hover:underline"
            onClick={onClear}
          >
            Clear
          </button>
        )}
      </div>
    </div>
  );
}

async function uploadTask(
  task: UploadTask,
  setTasks: React.Dispatch<React.SetStateAction<UploadTask[]>>,
  cancelledTaskIdsRef: React.MutableRefObject<Set<string>>,
  activeAbortRef: React.MutableRefObject<AbortController | null>,
  activeUploadIdsRef: React.MutableRefObject<Map<string, string>>,
) {
  updateTask(setTasks, task.id, {
    status: 'uploading',
    uploadedBytes: 0,
    currentFileName: null,
    error: null,
  });

  const session = await createFileUpload({
    destination_path: task.destinationPath,
    chunk_size: uploadChunkSize,
    files: task.files.map((source, index) => ({
      client_file_id: `${task.id}-${index}`,
      relative_path: source.relativePath,
      size: source.file.size,
    })),
  });

  activeUploadIdsRef.current.set(task.id, session.upload_id);
  updateTask(setTasks, task.id, { uploadId: session.upload_id });

  if (cancelledTaskIdsRef.current.has(task.id)) {
    await cancelFileUpload(session.upload_id).catch(() => undefined);
    throw new Error('Upload cancelled');
  }

  const heartbeatId = window.setInterval(() => {
    void heartbeatFileUpload(session.upload_id).catch(() => undefined);
  }, heartbeatIntervalMs);

  try {
    await uploadSessionFiles(
      task,
      session,
      setTasks,
      cancelledTaskIdsRef,
      activeAbortRef,
    );

    updateTask(setTasks, task.id, {
      status: 'complete',
      uploadedBytes: task.totalBytes,
      currentFileName: null,
    });
  } catch (error) {
    if (cancelledTaskIdsRef.current.has(task.id)) {
      updateTask(setTasks, task.id, {
        status: 'cancelled',
        currentFileName: null,
      });
      throw error;
    }

    await cancelFileUpload(session.upload_id).catch(() => undefined);
    throw error;
  } finally {
    window.clearInterval(heartbeatId);
    activeUploadIdsRef.current.delete(task.id);
  }
}

async function uploadSessionFiles(
  task: UploadTask,
  session: CreateFileUploadResponse,
  setTasks: React.Dispatch<React.SetStateAction<UploadTask[]>>,
  cancelledTaskIdsRef: React.MutableRefObject<Set<string>>,
  activeAbortRef: React.MutableRefObject<AbortController | null>,
) {
  let completedBytes = 0;

  for (let index = 0; index < task.files.length; index++) {
    throwIfCancelled(task.id, cancelledTaskIdsRef);

    const source = task.files[index];
    const serverFile = session.files.find(
      (file) => file.client_file_id === `${task.id}-${index}`,
    );
    if (!serverFile) {
      throw new Error('Upload server did not return file metadata');
    }

    updateTask(setTasks, task.id, {
      status: 'uploading',
      currentFileName: source.relativePath,
    });

    const hasher = new IncrementalSha256();
    let fileUploadedBytes = 0;

    for (
      let chunkIndex = 0;
      chunkIndex < serverFile.total_chunks;
      chunkIndex++
    ) {
      throwIfCancelled(task.id, cancelledTaskIdsRef);

      const start = chunkIndex * serverFile.chunk_size;
      const end = Math.min(source.file.size, start + serverFile.chunk_size);
      const chunk = source.file.slice(start, end);
      const bytes = new Uint8Array(await chunk.arrayBuffer());
      hasher.update(bytes);

      await uploadChunkWithRetry({
        taskId: task.id,
        uploadId: session.upload_id,
        fileId: serverFile.file_id,
        chunkIndex,
        chunk,
        completedBytes,
        fileUploadedBytes,
        setTasks,
        cancelledTaskIdsRef,
        activeAbortRef,
      });

      fileUploadedBytes += chunk.size;
      updateTask(setTasks, task.id, {
        uploadedBytes: completedBytes + fileUploadedBytes,
      });
    }

    updateTask(setTasks, task.id, {
      status: 'verifying',
      currentFileName: source.relativePath,
    });
    const sha256 = hasher.digestHex();
    throwIfCancelled(task.id, cancelledTaskIdsRef);

    const controller = new AbortController();
    activeAbortRef.current = controller;
    try {
      await completeFileUpload({
        uploadId: session.upload_id,
        fileId: serverFile.file_id,
        sha256,
        signal: controller.signal,
      });
    } finally {
      if (activeAbortRef.current === controller) {
        activeAbortRef.current = null;
      }
    }

    completedBytes += source.file.size;
    updateTask(setTasks, task.id, {
      uploadedBytes: completedBytes,
    });
  }
}

async function uploadChunkWithRetry({
  taskId,
  uploadId,
  fileId,
  chunkIndex,
  chunk,
  completedBytes,
  fileUploadedBytes,
  setTasks,
  cancelledTaskIdsRef,
  activeAbortRef,
}: {
  taskId: string;
  uploadId: string;
  fileId: string;
  chunkIndex: number;
  chunk: Blob;
  completedBytes: number;
  fileUploadedBytes: number;
  setTasks: React.Dispatch<React.SetStateAction<UploadTask[]>>;
  cancelledTaskIdsRef: React.MutableRefObject<Set<string>>;
  activeAbortRef: React.MutableRefObject<AbortController | null>;
}) {
  let lastError: unknown;
  for (let attempt = 1; attempt <= maxChunkAttempts; attempt++) {
    throwIfCancelled(taskId, cancelledTaskIdsRef);

    const controller = new AbortController();
    activeAbortRef.current = controller;

    try {
      await uploadFileChunk({
        uploadId,
        fileId,
        chunkIndex,
        chunk,
        signal: controller.signal,
        onUploadProgress: (loaded) => {
          updateTask(setTasks, taskId, {
            uploadedBytes: completedBytes + fileUploadedBytes + loaded,
          });
        },
      });
      return;
    } catch (error) {
      lastError = error;
      if (
        cancelledTaskIdsRef.current.has(taskId) ||
        !shouldRetryUploadError(error) ||
        attempt === maxChunkAttempts
      ) {
        break;
      }

      await wait(500 * 2 ** (attempt - 1));
    } finally {
      activeAbortRef.current = null;
    }
  }

  throw lastError;
}

function updateTask(
  setTasks: React.Dispatch<React.SetStateAction<UploadTask[]>>,
  taskId: string,
  patch: Partial<UploadTask>,
) {
  setTasks((current) =>
    current.map((task) => {
      if (task.id !== taskId) {
        return task;
      }

      return {
        ...task,
        ...patch,
      };
    }),
  );
}

function throwIfCancelled(
  taskId: string,
  cancelledTaskIdsRef: React.MutableRefObject<Set<string>>,
) {
  if (cancelledTaskIdsRef.current.has(taskId)) {
    throw new Error('Upload cancelled');
  }
}

function shouldRetryUploadError(error: unknown) {
  if (error instanceof APIError) {
    if (error.status === 0) {
      return true;
    }

    return error.status >= 500 && error.status !== 507;
  }

  return false;
}

function uploadErrorMessage(error: unknown) {
  if (error instanceof APIError) {
    if (error.status === 0) {
      return 'Unable to reach upload server after multiple retries';
    }

    return error.getErrorMessage();
  }

  if (error instanceof Error) {
    return error.message;
  }

  return 'Upload failed';
}

function wait(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

function dedupeUploadSources(sources: UploadSource[]) {
  const seen = new Set<string>();
  const result: UploadSource[] = [];
  for (const source of sources) {
    const relativePath = normalizeRelativeUploadPath(source.relativePath);
    if (!relativePath || seen.has(relativePath)) {
      continue;
    }

    seen.add(relativePath);
    result.push({
      file: source.file,
      relativePath,
    });
  }

  return result;
}

function findOversizedUploadSource(
  sources: UploadSource[],
  maxFileUploadSizeBytes?: number,
) {
  for (const source of sources) {
    const error = getUploadSizeError(
      source.relativePath || source.file.name,
      source.file.size,
      maxFileUploadSizeBytes,
    );
    if (error) {
      return error;
    }
  }

  return null;
}

function normalizeRelativeUploadPath(path: string) {
  return path
    .replace(/\\/g, '/')
    .split('/')
    .map((part) => part.trim())
    .filter(Boolean)
    .join('/');
}

function hasFileDrag(dataTransfer: DataTransfer) {
  return Array.from(dataTransfer.types).includes('Files');
}

async function uploadSourcesFromDataTransfer(dataTransfer: DataTransfer) {
  const itemEntries = Array.from(dataTransfer.items)
    .map((item) => (item as DataTransferItemWithEntry).webkitGetAsEntry?.())
    .filter((entry): entry is BrowserEntry => Boolean(entry));

  if (itemEntries.length > 0) {
    const sources: UploadSource[] = [];
    for (const entry of itemEntries) {
      sources.push(...(await uploadSourcesFromEntry(entry, '')));
    }

    return sources;
  }

  return Array.from(dataTransfer.files).map((file) => ({
    file,
    relativePath: file.name,
  }));
}

async function uploadSourcesFromEntry(
  entry: BrowserEntry,
  parentPath: string,
): Promise<UploadSource[]> {
  if (entry.isFile) {
    const file = await readEntryFile(entry as BrowserFileEntry);
    return [
      {
        file,
        relativePath: parentPath ? `${parentPath}/${file.name}` : file.name,
      },
    ];
  }

  const directory = entry as BrowserDirectoryEntry;
  const nextParentPath = parentPath
    ? `${parentPath}/${directory.name}`
    : directory.name;
  const entries = await readDirectoryEntries(directory);
  const sources: UploadSource[] = [];
  for (const child of entries) {
    sources.push(
      ...(await uploadSourcesFromEntry(child as BrowserEntry, nextParentPath)),
    );
  }

  return sources;
}

function readEntryFile(entry: BrowserFileEntry) {
  return new Promise<File>((resolve, reject) => {
    entry.file(resolve, reject);
  });
}

async function readDirectoryEntries(directory: BrowserDirectoryEntry) {
  const reader = directory.createReader();
  const entries: FileSystemEntry[] = [];

  for (;;) {
    const batch = await new Promise<FileSystemEntry[]>((resolve, reject) => {
      reader.readEntries(resolve, reject);
    });
    if (batch.length === 0) {
      break;
    }

    entries.push(...batch);
  }

  return entries;
}

class IncrementalSha256 {
  private hash = new Uint32Array([
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c,
    0x1f83d9ab, 0x5be0cd19,
  ]);

  private buffer = new Uint8Array(64);
  private bufferLength = 0;
  private bytesHashed = 0;
  private finished = false;
  private temp = new Uint32Array(64);

  update(data: Uint8Array) {
    if (this.finished) {
      throw new Error('Hash is already finalized');
    }

    let position = 0;
    this.bytesHashed += data.length;

    while (position < data.length) {
      const take = Math.min(data.length - position, 64 - this.bufferLength);
      this.buffer.set(
        data.subarray(position, position + take),
        this.bufferLength,
      );
      this.bufferLength += take;
      position += take;

      if (this.bufferLength === 64) {
        this.processBlock(this.buffer);
        this.bufferLength = 0;
      }
    }
  }

  digestHex() {
    this.finish();
    const bytes = new Uint8Array(32);
    for (let i = 0; i < this.hash.length; i++) {
      bytes[i * 4] = this.hash[i] >>> 24;
      bytes[i * 4 + 1] = this.hash[i] >>> 16;
      bytes[i * 4 + 2] = this.hash[i] >>> 8;
      bytes[i * 4 + 3] = this.hash[i];
    }

    return Array.from(bytes)
      .map((byte) => byte.toString(16).padStart(2, '0'))
      .join('');
  }

  private finish() {
    if (this.finished) {
      return;
    }

    const bytesHashed = this.bytesHashed;
    this.buffer[this.bufferLength++] = 0x80;

    if (this.bufferLength > 56) {
      this.buffer.fill(0, this.bufferLength, 64);
      this.processBlock(this.buffer);
      this.bufferLength = 0;
    }

    this.buffer.fill(0, this.bufferLength, 56);
    const bitsHigh = Math.floor(bytesHashed / 0x20000000);
    const bitsLow = (bytesHashed << 3) >>> 0;
    this.buffer[56] = bitsHigh >>> 24;
    this.buffer[57] = bitsHigh >>> 16;
    this.buffer[58] = bitsHigh >>> 8;
    this.buffer[59] = bitsHigh;
    this.buffer[60] = bitsLow >>> 24;
    this.buffer[61] = bitsLow >>> 16;
    this.buffer[62] = bitsLow >>> 8;
    this.buffer[63] = bitsLow;
    this.processBlock(this.buffer);
    this.finished = true;
  }

  private processBlock(block: Uint8Array) {
    const words = this.temp;
    for (let i = 0; i < 16; i++) {
      const offset = i * 4;
      words[i] =
        (block[offset] << 24) |
        (block[offset + 1] << 16) |
        (block[offset + 2] << 8) |
        block[offset + 3];
    }

    for (let i = 16; i < 64; i++) {
      const s0 =
        rotateRight(words[i - 15], 7) ^
        rotateRight(words[i - 15], 18) ^
        (words[i - 15] >>> 3);
      const s1 =
        rotateRight(words[i - 2], 17) ^
        rotateRight(words[i - 2], 19) ^
        (words[i - 2] >>> 10);
      words[i] = (words[i - 16] + s0 + words[i - 7] + s1) >>> 0;
    }

    let a = this.hash[0];
    let b = this.hash[1];
    let c = this.hash[2];
    let d = this.hash[3];
    let e = this.hash[4];
    let f = this.hash[5];
    let g = this.hash[6];
    let h = this.hash[7];

    for (let i = 0; i < 64; i++) {
      const s1 = rotateRight(e, 6) ^ rotateRight(e, 11) ^ rotateRight(e, 25);
      const ch = (e & f) ^ (~e & g);
      const temp1 = (h + s1 + ch + sha256K[i] + words[i]) >>> 0;
      const s0 = rotateRight(a, 2) ^ rotateRight(a, 13) ^ rotateRight(a, 22);
      const maj = (a & b) ^ (a & c) ^ (b & c);
      const temp2 = (s0 + maj) >>> 0;

      h = g;
      g = f;
      f = e;
      e = (d + temp1) >>> 0;
      d = c;
      c = b;
      b = a;
      a = (temp1 + temp2) >>> 0;
    }

    this.hash[0] = (this.hash[0] + a) >>> 0;
    this.hash[1] = (this.hash[1] + b) >>> 0;
    this.hash[2] = (this.hash[2] + c) >>> 0;
    this.hash[3] = (this.hash[3] + d) >>> 0;
    this.hash[4] = (this.hash[4] + e) >>> 0;
    this.hash[5] = (this.hash[5] + f) >>> 0;
    this.hash[6] = (this.hash[6] + g) >>> 0;
    this.hash[7] = (this.hash[7] + h) >>> 0;
  }
}

function rotateRight(value: number, bits: number) {
  return (value >>> bits) | (value << (32 - bits));
}

const sha256K = new Uint32Array([
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1,
  0x923f82a4, 0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
  0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786,
  0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147,
  0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
  0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b,
  0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a,
  0x5b9cca4f, 0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
  0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
]);
