import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Loader2, Folder, X, AlertTriangle } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
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
  getSession,
  updatePassword,
  getDirectoryShortcuts,
  deleteDirectoryShortcut,
  APIError,
  type DirectoryShortcutsResponse,
} from '@/lib/api';
import { queryKeys } from '@/constants';
import { toast } from 'sonner';

const changePasswordSchema = z
  .object({
    currentPassword: z.string().min(1, 'Current password is required'),
    newPassword: z.string().min(6, 'Password must be at least 6 characters'),
    confirmPassword: z.string(),
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    message: "Passwords don't match",
    path: ['confirmPassword'],
  })
  .refine((data) => data.currentPassword !== data.newPassword, {
    message: 'New password must be different from current password',
    path: ['newPassword'],
  });

type ChangePasswordFormData = z.infer<typeof changePasswordSchema>;

export function SettingsPage() {
  const { data: session } = useQuery({
    queryKey: queryKeys.session,
    queryFn: getSession,
    retry: false,
  });

  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [passwordSuccess, setPasswordSuccess] = useState(false);
  const [isUpdatingPassword, setIsUpdatingPassword] = useState(false);
  const [deleteShortcutId, setDeleteShortcutId] = useState<number | null>(null);
  const queryClient = useQueryClient();

  const { data: directoryShortcutsData } = useQuery({
    queryKey: queryKeys.directoryShortcuts,
    queryFn: getDirectoryShortcuts,
  });

  const deleteShortcutMutation = useMutation({
    mutationFn: async (id: number) => {
      return deleteDirectoryShortcut(id);
    },
    onMutate: async (id) => {
      await queryClient.cancelQueries({
        queryKey: queryKeys.directoryShortcuts,
      });
      const previousData = queryClient.getQueryData<DirectoryShortcutsResponse>(
        queryKeys.directoryShortcuts,
      );
      if (previousData) {
        queryClient.setQueryData<DirectoryShortcutsResponse>(
          queryKeys.directoryShortcuts,
          {
            ...previousData,
            shortcuts: previousData.shortcuts.filter((s) => s.id !== id),
            over_limit_by: Math.max(0, previousData.over_limit_by - 1),
          },
        );
      }
      return { previousData };
    },
    onError: (error: APIError, _id, context) => {
      if (context?.previousData) {
        queryClient.setQueryData(
          queryKeys.directoryShortcuts,
          context.previousData,
        );
      }
      toast.error(error.getErrorMessage());
    },
    onSuccess: () => {
      toast.success('Shortcut removed');
      setDeleteShortcutId(null);
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.directoryShortcuts });
    },
  });

  const passwordForm = useForm<ChangePasswordFormData>({
    resolver: zodResolver(changePasswordSchema),
    defaultValues: {
      currentPassword: '',
      newPassword: '',
      confirmPassword: '',
    },
  });

  const onPasswordChange = async (data: ChangePasswordFormData) => {
    setPasswordError(null);
    setPasswordSuccess(false);
    try {
      setIsUpdatingPassword(true);
      await updatePassword({
        current_password: data.currentPassword,
        new_password: data.newPassword,
      });

      setPasswordSuccess(true);
      passwordForm.reset();
      setTimeout(() => setPasswordSuccess(false), 5000);
    } catch (err) {
      if (err instanceof APIError) {
        setPasswordError(err.getErrorMessage());
      } else {
        setPasswordError(
          err instanceof Error ? err.message : 'Failed to update password',
        );
      }
    } finally {
      setIsUpdatingPassword(false);
    }
  };

  return (
    <div className="p-4 lg:p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">Manage your account settings</p>
      </div>

      <div className="space-y-6">
        {/* Profile Settings */}
        <Card>
          <CardHeader>
            <CardTitle>Profile</CardTitle>
            <CardDescription>Your account information</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                value={session?.email || ''}
                disabled
              />
            </div>
          </CardContent>
        </Card>

        {/* Security Settings */}
        <Card>
          <CardHeader>
            <CardTitle>Security</CardTitle>
            <CardDescription>
              Manage your password and security settings
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <form
              onSubmit={passwordForm.handleSubmit(onPasswordChange)}
              className="space-y-4"
            >
              {passwordError && (
                <Alert variant="destructive">
                  <AlertDescription>{passwordError}</AlertDescription>
                </Alert>
              )}
              {passwordSuccess && (
                <Alert>
                  <AlertDescription>
                    Password updated successfully
                  </AlertDescription>
                </Alert>
              )}
              <div className="space-y-2">
                <Label htmlFor="current-password">Current Password</Label>
                <Input
                  id="current-password"
                  type="password"
                  {...passwordForm.register('currentPassword')}
                  disabled={isUpdatingPassword}
                />
                {passwordForm.formState.errors.currentPassword && (
                  <p className="text-sm text-destructive">
                    {passwordForm.formState.errors.currentPassword.message}
                  </p>
                )}
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="new-password">New Password</Label>
                  <Input
                    id="new-password"
                    type="password"
                    {...passwordForm.register('newPassword')}
                    disabled={isUpdatingPassword}
                  />
                  {passwordForm.formState.errors.newPassword && (
                    <p className="text-sm text-destructive">
                      {passwordForm.formState.errors.newPassword.message}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="confirm-password">Confirm New Password</Label>
                  <Input
                    id="confirm-password"
                    type="password"
                    {...passwordForm.register('confirmPassword')}
                    disabled={isUpdatingPassword}
                  />
                  {passwordForm.formState.errors.confirmPassword && (
                    <p className="text-sm text-destructive">
                      {passwordForm.formState.errors.confirmPassword.message}
                    </p>
                  )}
                </div>
              </div>
              <Button type="submit" disabled={isUpdatingPassword}>
                {isUpdatingPassword && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Update Password
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* Directory Shortcuts */}
        <Card>
          <CardHeader>
            <CardTitle>Directory Shortcuts</CardTitle>
            <CardDescription>
              Manage your pinned directory shortcuts
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {directoryShortcutsData?.over_limit_by &&
              directoryShortcutsData.over_limit_by > 0 && (
                <Alert>
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription>
                    You have {directoryShortcutsData.over_limit_by} shortcut(s)
                    over the limit. Only the first{' '}
                    {directoryShortcutsData.limit} shortcuts appear in the
                    sidebar.
                  </AlertDescription>
                </Alert>
              )}
            {directoryShortcutsData?.shortcuts &&
            directoryShortcutsData.shortcuts.length > 0 ? (
              <div className="space-y-2">
                {directoryShortcutsData.shortcuts.map((shortcut) => (
                  <div
                    key={shortcut.id}
                    className="flex items-center justify-between rounded-lg border p-3"
                  >
                    <div className="flex flex-1 items-center gap-3 min-w-0">
                      <Folder className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <div className="flex-1 min-w-0">
                        <div className="font-medium truncate">
                          {shortcut.name}
                        </div>
                        <div className="text-sm text-muted-foreground truncate">
                          {shortcut.path}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <Button variant="ghost" size="sm" asChild className="h-8">
                        <Link to="/file" search={{ path: shortcut.path }}>
                          Open
                        </Link>
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                        onClick={() => setDeleteShortcutId(shortcut.id)}
                        aria-label={`Remove ${shortcut.name}`}
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8 text-muted-foreground">
                No shortcuts yet. Pin directories from the File Browser to add
                them here.
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <AlertDialog
        open={deleteShortcutId !== null}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteShortcutId(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove Shortcut</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to remove this shortcut? This action cannot
              be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteShortcutMutation.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deleteShortcutId !== null) {
                  deleteShortcutMutation.mutate(deleteShortcutId);
                }
              }}
              disabled={deleteShortcutMutation.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
