import { useState, useRef } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Upload, FileText, Loader2, CheckCircle2, XCircle } from 'lucide-react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { uploadMonFile, APIError } from '@/lib/api';
import { cn } from '@/lib/utils';

export function MonsterFileUpload() {
  const [file, setFile] = useState<File | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const uploadMutation = useMutation({
    mutationFn: uploadMonFile,
    onSuccess: () => {
      setFile(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    },
  });

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);

    const droppedFile = e.dataTransfer.files[0];
    if (droppedFile && droppedFile.name.toLowerCase().endsWith('.ull')) {
      setFile(droppedFile);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (selectedFile && selectedFile.name.toLowerCase().endsWith('.ull')) {
      setFile(selectedFile);
    }
  };

  const handleUpload = () => {
    if (file) {
      uploadMutation.mutate(file);
    }
  };

  const handleClickUpload = () => {
    fileInputRef.current?.click();
  };

  const isUploading = uploadMutation.isPending;
  const success = uploadMutation.isSuccess ? uploadMutation.data : null;
  const error = uploadMutation.isError
    ? uploadMutation.error instanceof APIError
      ? uploadMutation.error.getErrorMessage()
      : 'Failed to upload file'
    : null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <FileText className="h-5 w-5" />
          Monster Client Data
        </CardTitle>
        <CardDescription>
          Upload MON.ull file to populate monster client data in the database
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          className={cn(
            'relative flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 transition-colors',
            isDragging
              ? 'border-primary bg-primary/5'
              : 'border-muted-foreground/25 hover:border-muted-foreground/50',
            file && 'border-primary bg-primary/5',
          )}
        >
          <input
            ref={fileInputRef}
            type="file"
            accept=".ull"
            onChange={handleFileSelect}
            className="hidden"
            tabIndex={-1}
            aria-label="Select MON.ull file"
          />
          <Upload className="mb-4 h-12 w-12 text-muted-foreground" />
          <div className="text-center">
            <p className="text-sm font-medium">
              {file ? file.name : 'Drag and drop MON.ull file here'}
            </p>
            <p className="mt-2 text-xs text-muted-foreground">
              or{' '}
              <button
                type="button"
                onClick={handleClickUpload}
                className="text-primary hover:underline"
                tabIndex={0}
                aria-label="Click to select file"
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    handleClickUpload();
                  }
                }}
              >
                click to browse
              </button>
            </p>
          </div>
        </div>

        {file && (
          <div className="flex items-center justify-between rounded-lg bg-muted p-3">
            <div className="flex items-center gap-2">
              <FileText className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-medium">{file.name}</span>
              <span className="text-xs text-muted-foreground">
                ({(file.size / 1024).toFixed(2)} KB)
              </span>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setFile(null);
                if (fileInputRef.current) {
                  fileInputRef.current.value = '';
                }
              }}
              disabled={isUploading}
              tabIndex={0}
              aria-label="Remove file"
            >
              <XCircle className="h-4 w-4" />
            </Button>
          </div>
        )}

        {error && (
          <Alert variant="destructive">
            <XCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {success && (
          <Alert>
            <CheckCircle2 className="h-4 w-4" />
            <AlertDescription>
              {success.message}. {success.count} records imported successfully.
            </AlertDescription>
          </Alert>
        )}

        <Button
          onClick={handleUpload}
          disabled={!file || isUploading}
          className="w-full"
          tabIndex={0}
          aria-label="Upload MON.ull file"
        >
          {isUploading ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Uploading...
            </>
          ) : (
            <>
              <Upload className="mr-2 h-4 w-4" />
              Upload MON.ull
            </>
          )}
        </Button>
      </CardContent>
    </Card>
  );
}
