import { useState } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Loader2, Edit, Key, ChevronLeft, ChevronRight } from 'lucide-react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import {
  getUsers,
  getUserStatuses,
  updateUserStatus,
  setUserPassword,
  type UserListItem,
  type SetUserPasswordRequest,
  APIError,
} from '@/lib/api';
import { queryKeys } from '@/constants';
import { toast } from 'sonner';

const createUpdateStatusSchema = (allowedStatuses: string[]) =>
  z.object({
    status: z.enum(allowedStatuses as [string, ...string[]]),
  });

const setPasswordSchema = z
  .object({
    password: z.string().min(6, 'Password must be at least 6 characters'),
    repeatPassword: z.string().min(6, 'Password must be at least 6 characters'),
  })
  .refine((data) => data.password === data.repeatPassword, {
    message: 'Passwords do not match',
    path: ['repeatPassword'],
  });

type SetPasswordFormData = z.infer<typeof setPasswordSchema>;

function getStatusBadgeVariant(status: string) {
  switch (status) {
    case 'active':
      return 'default';
    case 'pending':
      return 'secondary';
    case 'inactive':
      return 'outline';
    case 'banned':
      return 'destructive';
    default:
      return 'secondary';
  }
}

export function UsersPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [statusModalOpen, setStatusModalOpen] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<UserListItem | null>(null);

  const { data: statusesData } = useQuery({
    queryKey: queryKeys.userStatuses,
    queryFn: getUserStatuses,
  });

  const allowedStatuses = statusesData?.statuses || [];

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.users(page, pageSize, search),
    queryFn: () => getUsers({ page, pageSize, s: search || undefined }),
  });

  const updateStatusSchema = createUpdateStatusSchema(
    allowedStatuses.length > 0 ? allowedStatuses : ['active'],
  );
  type UpdateStatusFormData = z.infer<typeof updateStatusSchema>;

  const statusForm = useForm<UpdateStatusFormData>({
    resolver: zodResolver(updateStatusSchema),
    defaultValues: {
      status: (allowedStatuses.length > 0
        ? allowedStatuses[0]
        : 'active') as UpdateStatusFormData['status'],
    },
  });

  const passwordForm = useForm<SetPasswordFormData>({
    resolver: zodResolver(setPasswordSchema),
    defaultValues: {
      password: '',
      repeatPassword: '',
    },
  });

  const updateStatusMutation = useMutation({
    mutationFn: (data: { status: string }) =>
      updateUserStatus(selectedUser!.id, {
        status: data.status as 'pending' | 'active' | 'inactive' | 'banned',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.users(page, pageSize, search),
      });
      setStatusModalOpen(false);
      setSelectedUser(null);
      statusForm.reset();
      toast.success('User status updated successfully');
    },
    onError: (error: unknown) => {
      if (error instanceof APIError) {
        toast.error(error.getErrorMessage());
      } else {
        toast.error('Failed to update user status');
      }
    },
  });

  const setPasswordMutation = useMutation({
    mutationFn: (data: SetUserPasswordRequest) =>
      setUserPassword(selectedUser!.id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.users(page, pageSize, search),
      });
      setPasswordModalOpen(false);
      setSelectedUser(null);
      passwordForm.reset();
      toast.success('Password set successfully');
    },
    onError: (error: unknown) => {
      if (error instanceof APIError) {
        toast.error(error.getErrorMessage());
      } else {
        toast.error('Failed to set password');
      }
    },
  });

  const handleOpenStatusModal = (user: UserListItem) => {
    setSelectedUser(user);
    statusForm.reset({ status: user.status as UpdateStatusFormData['status'] });
    setStatusModalOpen(true);
  };

  const handleOpenPasswordModal = (user: UserListItem) => {
    setSelectedUser(user);
    passwordForm.reset();
    setPasswordModalOpen(true);
  };

  const handleStatusSubmit = (data: UpdateStatusFormData) => {
    if (selectedUser && data.status !== selectedUser.status) {
      updateStatusMutation.mutate({ status: data.status as string });
    } else {
      toast.error('New status must be different from current status');
    }
  };

  const handlePasswordSubmit = (data: SetPasswordFormData) => {
    if (selectedUser) {
      setPasswordMutation.mutate({ password: data.password });
    }
  };

  const totalPages = data
    ? Math.ceil(data.pagination.totalCount / data.pagination.pageSize)
    : 0;

  const handlePageChange = (newPage: number) => {
    if (newPage >= 1 && newPage <= totalPages) {
      setPage(newPage);
    }
  };

  return (
    <div className="p-4 lg:p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Users</h1>
        <p className="text-muted-foreground">
          Manage user accounts and permissions
        </p>
      </div>

      <div className="mb-4">
        <Input
          placeholder="Search by email..."
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(1);
          }}
          className="max-w-sm"
        />
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Roles</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created At</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin mx-auto" />
                </TableCell>
              </TableRow>
            ) : isError ? (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className="text-center py-8 text-destructive"
                >
                  Failed to load users
                </TableCell>
              </TableRow>
            ) : data && data.data.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className="text-center py-8 text-muted-foreground"
                >
                  No users found
                </TableCell>
              </TableRow>
            ) : (
              data?.data.map((user) => (
                <TableRow key={user.id}>
                  <TableCell>{user.id}</TableCell>
                  <TableCell>{user.email}</TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {user.roles.map((role) => (
                        <Badge key={role} variant="outline">
                          {role}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getStatusBadgeVariant(user.status)}>
                      {user.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {new Date(user.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-2">
                      {!user.roles.includes('super_admin') && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleOpenStatusModal(user)}
                          aria-label="Update status"
                          tabIndex={0}
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleOpenPasswordModal(user)}
                        aria-label="Set password"
                        tabIndex={0}
                      >
                        <Key className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {data && data.pagination.totalCount > 0 && (
        <div className="mt-4 flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            Showing {(page - 1) * pageSize + 1} to{' '}
            {Math.min(page * pageSize, data.pagination.totalCount)} of{' '}
            {data.pagination.totalCount} users
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => handlePageChange(page - 1)}
              disabled={page === 1 || isLoading}
            >
              <ChevronLeft className="h-4 w-4" />
              Previous
            </Button>
            <div className="text-sm">
              Page {page} of {totalPages}
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => handlePageChange(page + 1)}
              disabled={page >= totalPages || isLoading}
            >
              Next
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      <Dialog open={statusModalOpen} onOpenChange={setStatusModalOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Update User Status</DialogTitle>
            <DialogDescription>
              Update the status for {selectedUser?.email}
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={statusForm.handleSubmit(handleStatusSubmit)}>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="status">Status</Label>
                <Controller
                  name="status"
                  control={statusForm.control}
                  render={({ field }) => (
                    <Select
                      value={field.value}
                      onValueChange={field.onChange}
                      disabled={allowedStatuses.length === 0}
                    >
                      <SelectTrigger id="status">
                        <SelectValue placeholder="Select status" />
                      </SelectTrigger>
                      <SelectContent>
                        {allowedStatuses.map((status) => (
                          <SelectItem key={status} value={status}>
                            {status.charAt(0).toUpperCase() + status.slice(1)}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                />
                {statusForm.formState.errors.status && (
                  <p className="text-sm text-destructive">
                    {statusForm.formState.errors.status.message}
                  </p>
                )}
              </div>
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setStatusModalOpen(false);
                  setSelectedUser(null);
                  statusForm.reset();
                }}
                disabled={updateStatusMutation.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={updateStatusMutation.isPending}>
                {updateStatusMutation.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Save
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={passwordModalOpen} onOpenChange={setPasswordModalOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Set User Password</DialogTitle>
            <DialogDescription>
              Set a new password for {selectedUser?.email}
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={passwordForm.handleSubmit(handlePasswordSubmit)}>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="password">New Password</Label>
                <Input
                  id="password"
                  type="password"
                  {...passwordForm.register('password')}
                  disabled={setPasswordMutation.isPending}
                />
                {passwordForm.formState.errors.password && (
                  <p className="text-sm text-destructive">
                    {passwordForm.formState.errors.password.message}
                  </p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="repeatPassword">Repeat Password</Label>
                <Input
                  id="repeatPassword"
                  type="password"
                  {...passwordForm.register('repeatPassword')}
                  disabled={setPasswordMutation.isPending}
                />
                {passwordForm.formState.errors.repeatPassword && (
                  <p className="text-sm text-destructive">
                    {passwordForm.formState.errors.repeatPassword.message}
                  </p>
                )}
              </div>
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setPasswordModalOpen(false);
                  setSelectedUser(null);
                  passwordForm.reset();
                }}
                disabled={setPasswordMutation.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={setPasswordMutation.isPending}>
                {setPasswordMutation.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Save
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
