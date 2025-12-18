import { useEffect, useRef } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from '@tanstack/react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Loader2, X, Save } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Link } from '@tanstack/react-router';
import { toast } from 'sonner';
import { APIError, updateTextFile } from '@/lib/api';
import { queryKeys } from '@/constants';

const textFileSchema = z.object({
  content: z.string(),
});

type TextFileFormData = z.infer<typeof textFileSchema>;

interface TextFileEditProps {
  filePath: string;
  defaultContent: string;
}

export function TextFileEdit({ filePath, defaultContent }: TextFileEditProps) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const previousFilePathRef = useRef<string>(filePath);

  const form = useForm<TextFileFormData>({
    resolver: zodResolver(textFileSchema),
    defaultValues: {
      content: defaultContent,
    },
  });

  useEffect(() => {
    if (previousFilePathRef.current !== filePath) {
      previousFilePathRef.current = filePath;
      form.reset({
        content: defaultContent,
      });
    }
  }, [filePath, defaultContent, form]);

  const mutation = useMutation({
    mutationFn: (data: TextFileFormData) =>
      updateTextFile({ path: filePath }, { content: data.content }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.textFile(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.fileTree(filePath),
      });
      toast.success('File saved');
      router.navigate({
        to: '/file/view',
        search: { path: filePath },
      });
    },
    onError: (error) => {
      const errorMessage =
        error instanceof APIError
          ? error.getErrorMessage()
          : error instanceof Error
            ? error.message
            : 'Failed to save file';
      toast.error(errorMessage);
    },
  });

  const mutationErrorMessage =
    mutation.error instanceof APIError
      ? mutation.error.getErrorMessage()
      : mutation.error instanceof Error
        ? mutation.error.message
        : 'Failed to save file';

  const isSaving = mutation.status === 'pending';

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Text File Editor</CardTitle>
        </CardHeader>
        <CardContent>
          <form
            onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
            className="space-y-4"
          >
            {mutation.isError && (
              <Alert variant="destructive">
                <AlertDescription>{mutationErrorMessage}</AlertDescription>
              </Alert>
            )}
            <div className="space-y-2">
              <Label htmlFor="content">Content</Label>
              <Textarea
                id="content"
                className="h-[600px] font-mono"
                {...form.register('content')}
              />
              {form.formState.errors.content && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.content.message}
                </p>
              )}
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Button type="submit" disabled={isSaving}>
                <span className="flex items-center gap-1.5">
                  {isSaving ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4" />
                  )}
                  Save Changes
                </span>
              </Button>
              <Button variant="outline" asChild>
                <Link to="/file/view" search={{ path: filePath }}>
                  <span className="flex items-center gap-1.5">
                    <X className="h-4 w-4" />
                    Cancel
                  </span>
                </Link>
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
