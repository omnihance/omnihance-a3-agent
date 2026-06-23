import { getUploadSizeError } from '@/lib/upload-validation';

export function validateGameClientUploadFile(
  file: File,
  fileLabel: string,
  maxFileUploadSizeBytes?: number,
) {
  if (!file.name.toLowerCase().endsWith('.ull')) {
    return `Please select a valid ${fileLabel}.ull file.`;
  }

  return getUploadSizeError(file.name, file.size, maxFileUploadSizeBytes);
}
