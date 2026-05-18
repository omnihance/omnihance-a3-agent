import { useMemo, useState } from 'react';
import { useForm, type UseFormReturn } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  AlertTriangle,
  Edit,
  Folder,
  Loader2,
  Plus,
  Trash2,
  X,
} from 'lucide-react';
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  getSession,
  updatePassword,
  getDirectoryShortcuts,
  deleteDirectoryShortcut,
  getSettings,
  createSetting,
  updateSetting,
  deleteSetting,
  APIError,
  type DirectoryShortcutsResponse,
  type Setting,
  type SettingDefinition,
  type SettingKey,
} from '@/lib/api';
import { queryKeys } from '@/constants';
import { toast } from 'sonner';
import { usePermissions } from '@/hooks/use-permissions';

const settingKeySchema = z.enum(['DB_HOST', 'DB_PORT', 'DB_USER', 'DB_PASS']);

const settingFormSchema = z
  .object({
    key: settingKeySchema,
    value: z.string(),
  })
  .superRefine((data, ctx) => {
    const errorMessage = getSettingValueError(data.key, data.value);
    if (errorMessage) {
      ctx.addIssue({
        code: 'custom',
        message: errorMessage,
        path: ['value'],
      });
    }
  });

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

type SettingFormData = z.infer<typeof settingFormSchema>;
type ChangePasswordFormData = z.infer<typeof changePasswordSchema>;

export function SettingsPage() {
  const queryClient = useQueryClient();
  const { hasPermission } = usePermissions();
  const canManageServer = hasPermission('manage_server');

  const { data: session } = useQuery({
    queryKey: queryKeys.session,
    queryFn: getSession,
    retry: false,
  });

  const { data: settingsData, isLoading: isSettingsLoading } = useQuery({
    queryKey: queryKeys.settings,
    queryFn: getSettings,
    enabled: canManageServer,
  });

  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [passwordSuccess, setPasswordSuccess] = useState(false);
  const [isUpdatingPassword, setIsUpdatingPassword] = useState(false);
  const [deleteShortcutId, setDeleteShortcutId] = useState<number | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [updateDialogOpen, setUpdateDialogOpen] = useState(false);
  const [deleteSettingKey, setDeleteSettingKey] = useState<SettingKey | null>(
    null,
  );
  const [selectedSetting, setSelectedSetting] = useState<Setting | null>(null);

  const { data: directoryShortcutsData } = useQuery({
    queryKey: queryKeys.directoryShortcuts,
    queryFn: getDirectoryShortcuts,
  });

  const settingsByKey = useMemo(() => {
    const map = new Map<SettingKey, Setting>();
    for (const setting of settingsData?.settings || []) {
      map.set(setting.key, setting);
    }

    return map;
  }, [settingsData?.settings]);

  const definitionsByKey = useMemo(() => {
    const map = new Map<SettingKey, SettingDefinition>();
    for (const definition of settingsData?.definitions || []) {
      map.set(definition.key, definition);
    }

    return map;
  }, [settingsData?.definitions]);

  const availableDefinitions = useMemo(
    () =>
      (settingsData?.definitions || []).filter(
        (definition) => !settingsByKey.has(definition.key),
      ),
    [settingsData?.definitions, settingsByKey],
  );

  const createForm = useForm<SettingFormData>({
    resolver: zodResolver(settingFormSchema),
    defaultValues: {
      key: 'DB_HOST',
      value: '',
    },
  });

  const updateForm = useForm<SettingFormData>({
    resolver: zodResolver(settingFormSchema),
    defaultValues: {
      key: 'DB_HOST',
      value: '',
    },
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

  const createSettingMutation = useMutation({
    mutationFn: createSetting,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.settings });
      toast.success('Setting created');
      setCreateDialogOpen(false);
      createForm.reset({ key: 'DB_HOST', value: '' });
    },
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
  });

  const updateSettingMutation = useMutation({
    mutationFn: ({ key, value }: { key: SettingKey; value: string }) =>
      updateSetting(key, { value }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.settings });
      toast.success('Setting updated');
      setUpdateDialogOpen(false);
      setSelectedSetting(null);
    },
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
  });

  const deleteSettingMutation = useMutation({
    mutationFn: deleteSetting,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.settings });
      toast.success('Setting deleted');
      setDeleteSettingKey(null);
    },
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
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

  const openCreateDialog = () => {
    const firstDefinition = availableDefinitions[0];
    if (!firstDefinition) {
      return;
    }

    createForm.reset({ key: firstDefinition.key, value: '' });
    setCreateDialogOpen(true);
  };

  const openUpdateDialog = (setting: Setting) => {
    setSelectedSetting(setting);
    updateForm.reset({ key: setting.key, value: setting.value });
    setUpdateDialogOpen(true);
  };

  const onCreateSetting = (data: SettingFormData) => {
    createSettingMutation.mutate({
      key: data.key,
      value: data.value,
    });
  };

  const onUpdateSetting = (data: SettingFormData) => {
    updateSettingMutation.mutate({
      key: data.key,
      value: data.value,
    });
  };

  return (
    <div className="p-4 lg:p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">
          Manage account and server settings
        </p>
      </div>

      <div className="space-y-6">
        {canManageServer && (
          <Card>
            <CardHeader className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <CardTitle>Game Server Database</CardTitle>
                <CardDescription>SQL Server connection values</CardDescription>
              </div>
              <Button
                onClick={openCreateDialog}
                disabled={availableDefinitions.length === 0}
                aria-label="Create setting"
              >
                <Plus className="mr-2 h-4 w-4" />
                Create Setting
              </Button>
            </CardHeader>
            <CardContent>
              {isSettingsLoading ? (
                <div className="flex justify-center py-10">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              ) : !settingsData || settingsData.settings.length === 0 ? (
                <div className="py-8 text-center text-muted-foreground">
                  No settings configured.
                </div>
              ) : (
                <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Setting</TableHead>
                        <TableHead>Key</TableHead>
                        <TableHead>Value</TableHead>
                        <TableHead className="text-right">Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {settingsData.settings.map((setting) => {
                        const definition = definitionsByKey.get(setting.key);
                        return (
                          <TableRow key={setting.key}>
                            <TableCell className="font-medium">
                              {definition?.label || setting.key}
                            </TableCell>
                            <TableCell className="font-mono text-xs">
                              {setting.key}
                            </TableCell>
                            <TableCell className="max-w-[22rem] truncate font-mono text-xs">
                              {setting.value}
                            </TableCell>
                            <TableCell className="text-right">
                              <div className="flex items-center justify-end gap-2">
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => openUpdateDialog(setting)}
                                  aria-label={`Update ${setting.key}`}
                                >
                                  <Edit className="h-4 w-4" />
                                </Button>
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() =>
                                    setDeleteSettingKey(setting.key)
                                  }
                                  aria-label={`Delete ${setting.key}`}
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </div>
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        )}

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
                    <div className="flex min-w-0 flex-1 items-center gap-3">
                      <Folder className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <div className="min-w-0 flex-1">
                        <div className="truncate font-medium">
                          {shortcut.name}
                        </div>
                        <div className="truncate text-sm text-muted-foreground">
                          {shortcut.path}
                        </div>
                      </div>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
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
              <div className="py-8 text-center text-muted-foreground">
                No shortcuts yet. Pin directories from the File Browser to add
                them here.
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create Setting</DialogTitle>
            <DialogDescription>
              Add a game server database setting
            </DialogDescription>
          </DialogHeader>
          <form
            id="create-setting-form"
            onSubmit={createForm.handleSubmit(onCreateSetting)}
            className="space-y-4"
          >
            <SettingForm
              form={createForm}
              definitions={availableDefinitions}
              disabled={createSettingMutation.isPending}
              lockKey={false}
            />
          </form>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCreateDialogOpen(false)}
              disabled={createSettingMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              form="create-setting-form"
              disabled={createSettingMutation.isPending}
            >
              {createSettingMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={updateDialogOpen} onOpenChange={setUpdateDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Update Setting</DialogTitle>
            <DialogDescription>
              Update {selectedSetting?.key || 'setting'} value
            </DialogDescription>
          </DialogHeader>
          <form
            id="update-setting-form"
            onSubmit={updateForm.handleSubmit(onUpdateSetting)}
            className="space-y-4"
          >
            <SettingForm
              form={updateForm}
              definitions={settingsData?.definitions || []}
              disabled={updateSettingMutation.isPending}
              lockKey
            />
          </form>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setUpdateDialogOpen(false)}
              disabled={updateSettingMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              form="update-setting-form"
              disabled={updateSettingMutation.isPending}
            >
              {updateSettingMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Update
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={deleteSettingKey !== null}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteSettingKey(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Setting</AlertDialogTitle>
            <AlertDialogDescription>
              Delete {deleteSettingKey}? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteSettingMutation.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deleteSettingKey) {
                  deleteSettingMutation.mutate(deleteSettingKey);
                }
              }}
              disabled={deleteSettingMutation.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteSettingMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

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

function SettingForm({
  form,
  definitions,
  disabled,
  lockKey,
}: {
  form: UseFormReturn<SettingFormData>;
  definitions: SettingDefinition[];
  disabled: boolean;
  lockKey: boolean;
}) {
  const selectedKey = form.watch('key');
  const selectedDefinition = definitions.find(
    (definition) => definition.key === selectedKey,
  );
  const inputType = selectedDefinition?.input_type || 'text';

  return (
    <>
      <div className="space-y-2">
        <Label htmlFor={lockKey ? 'update-setting-key' : 'create-setting-key'}>
          Setting
        </Label>
        {lockKey ? (
          <Input
            id="update-setting-key"
            value={selectedDefinition?.label || selectedKey}
            disabled
          />
        ) : (
          <Select
            value={selectedKey}
            onValueChange={(value) => {
              form.setValue('key', value as SettingKey, {
                shouldValidate: true,
              });
              form.setValue('value', '', { shouldValidate: true });
            }}
            disabled={disabled}
          >
            <SelectTrigger id="create-setting-key" className="w-full">
              <SelectValue placeholder="Select setting" />
            </SelectTrigger>
            <SelectContent>
              {definitions.map((definition) => (
                <SelectItem key={definition.key} value={definition.key}>
                  {definition.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        {form.formState.errors.key && (
          <p className="text-sm text-destructive" role="alert">
            {form.formState.errors.key.message}
          </p>
        )}
      </div>
      <div className="space-y-2">
        <Label
          htmlFor={lockKey ? 'update-setting-value' : 'create-setting-value'}
        >
          Value
        </Label>
        <Input
          id={lockKey ? 'update-setting-value' : 'create-setting-value'}
          type={inputType}
          min={selectedKey === 'DB_PORT' ? 1 : undefined}
          max={selectedKey === 'DB_PORT' ? 65535 : undefined}
          {...form.register('value')}
          disabled={disabled}
        />
        {form.formState.errors.value && (
          <p className="text-sm text-destructive" role="alert">
            {form.formState.errors.value.message}
          </p>
        )}
      </div>
    </>
  );
}

function getSettingValueError(key: SettingKey, value: string): string | null {
  if (key === 'DB_HOST') {
    const normalizedValue = value.trim();
    if (!normalizedValue) {
      return 'Game server DB host is required';
    }
    if (normalizedValue.length > 255) {
      return 'Game server DB host must be at most 255 characters';
    }
    if (hasControlCharacter(normalizedValue)) {
      return 'Game server DB host cannot contain control characters';
    }
  }

  if (key === 'DB_PORT') {
    const normalizedValue = value.trim();
    if (!/^\d+$/.test(normalizedValue)) {
      return 'Game server DB port must be between 1 and 65535';
    }

    const port = Number(normalizedValue);
    if (!Number.isInteger(port) || port < 1 || port > 65535) {
      return 'Game server DB port must be between 1 and 65535';
    }
  }

  if (key === 'DB_USER') {
    const normalizedValue = value.trim();
    if (!normalizedValue) {
      return 'Game server DB username is required';
    }
    if (normalizedValue.length > 128) {
      return 'Game server DB username must be at most 128 characters';
    }
  }

  if (key === 'DB_PASS') {
    if (!value) {
      return 'Game server DB password is required';
    }
    if (value.length > 512) {
      return 'Game server DB password must be at most 512 characters';
    }
  }

  return null;
}

function hasControlCharacter(value: string): boolean {
  for (const char of value) {
    const code = char.charCodeAt(0);
    if (code < 32 || code === 127) {
      return true;
    }
  }

  return false;
}
