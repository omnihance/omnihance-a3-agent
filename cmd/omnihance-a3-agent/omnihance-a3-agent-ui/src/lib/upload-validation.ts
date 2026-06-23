import { formatBytes } from '@/lib/util';

export function getUploadSizeError(
  fileName: string,
  fileSize: number,
  maxFileUploadSizeBytes?: number | null,
) {
  if (!isValidUploadSizeLimit(maxFileUploadSizeBytes)) {
    return null;
  }

  if (fileSize <= maxFileUploadSizeBytes) {
    return null;
  }

  return `${fileName} exceeds the maximum upload size of ${formatBytes(maxFileUploadSizeBytes)}.`;
}

function isValidUploadSizeLimit(
  maxFileUploadSizeBytes?: number | null,
): maxFileUploadSizeBytes is number {
  return (
    typeof maxFileUploadSizeBytes === 'number' &&
    Number.isFinite(maxFileUploadSizeBytes) &&
    maxFileUploadSizeBytes > 0
  );
}
