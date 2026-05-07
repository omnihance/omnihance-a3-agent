import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Edit, Loader2, Plus, PlugZap, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import {
  APIError,
  createSQLServerODBCDSN,
  deleteSQLServerODBCDSN,
  testSQLServerODBCDSNConnection,
  updateSQLServerODBCDSN,
  getSQLServerODBCDSNs,
  type SQLServerODBCDSN,
  type SQLServerODBCDSNRequest,
} from '@/lib/api';
import { queryKeys } from '@/constants';
import { usePermissions } from '@/hooks/use-permissions';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
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

const emptyForm: SQLServerODBCDSNRequest = {
  name: '',
  server: '',
  database: '',
  login_id: '',
  password: '',
  description: '',
  last_user: '',
};

export function SQLServerODBCPage() {
  const queryClient = useQueryClient();
  const { hasPermission } = usePermissions();
  const canManageServer = hasPermission('manage_server');

  const [form, setForm] = useState<SQLServerODBCDSNRequest>(emptyForm);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedDSN, setSelectedDSN] = useState<SQLServerODBCDSN | null>(null);

  const { data: dsns, isLoading } = useQuery({
    queryKey: queryKeys.sqlServerOdbcDsns,
    queryFn: getSQLServerODBCDSNs,
  });

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.sqlServerOdbcDsns });

  const createMutation = useMutation({
    mutationFn: createSQLServerODBCDSN,
    onSuccess: () => {
      invalidate();
      toast.success('DSN created');
      setAddDialogOpen(false);
      setForm(emptyForm);
    },
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
  });

  const updateMutation = useMutation({
    mutationFn: ({ name, payload }: { name: string; payload: SQLServerODBCDSNRequest }) =>
      updateSQLServerODBCDSN(name, payload),
    onSuccess: () => {
      invalidate();
      toast.success('DSN updated');
      setEditDialogOpen(false);
      setSelectedDSN(null);
    },
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteSQLServerODBCDSN,
    onSuccess: () => {
      invalidate();
      toast.success('DSN deleted');
      setDeleteDialogOpen(false);
      setSelectedDSN(null);
    },
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
  });

  const testMutation = useMutation({
    mutationFn: testSQLServerODBCDSNConnection,
    onSuccess: () => toast.success('Connection successful'),
    onError: (error: APIError) => toast.error(error.getErrorMessage()),
  });

  const validateAndGetPayload = (): SQLServerODBCDSNRequest | null => {
    if (
      !form.name.trim() ||
      !form.server.trim() ||
      !form.database.trim() ||
      !form.login_id.trim() ||
      !form.password.trim()
    ) {
      toast.error('Name, server, database, login id, and password are required');
      return null;
    }

    return {
      ...form,
      name: form.name.trim(),
      server: form.server.trim(),
      database: form.database.trim(),
      login_id: form.login_id.trim(),
      password: form.password,
      description: form.description?.trim() || '',
      last_user: form.last_user?.trim() || '',
    };
  };

  const openAdd = () => {
    setForm(emptyForm);
    setAddDialogOpen(true);
  };

  const openEdit = (dsn: SQLServerODBCDSN) => {
    setSelectedDSN(dsn);
    setForm({
      name: dsn.name,
      server: dsn.server,
      database: dsn.database,
      login_id: dsn.login_id,
      password: '',
      description: dsn.description ?? '',
      last_user: dsn.last_user ?? '',
    });
    setEditDialogOpen(true);
  };

  return (
    <div className="p-4 lg:p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">SQL Server ODBC</h1>
          <p className="text-muted-foreground">
            Manage 32-bit SQL Server User DSN configuration
          </p>
        </div>
        {canManageServer && (
          <Button onClick={openAdd}>
            <Plus className="mr-2 h-4 w-4" />
            Add DSN
          </Button>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Configured DSNs</CardTitle>
          <CardDescription>Legacy SQL Server User DSNs</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex justify-center py-10">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : !dsns || dsns.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              No SQL Server DSNs configured.
            </div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Server</TableHead>
                    <TableHead>Database</TableHead>
                    <TableHead>Login</TableHead>
                    {canManageServer && (
                      <TableHead className="text-right">Actions</TableHead>
                    )}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {dsns.map((dsn) => (
                    <TableRow key={dsn.name}>
                      <TableCell className="font-medium">{dsn.name}</TableCell>
                      <TableCell>{dsn.server}</TableCell>
                      <TableCell>{dsn.database}</TableCell>
                      <TableCell>{dsn.login_id}</TableCell>
                      {canManageServer && (
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-2">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => openEdit(dsn)}
                            >
                              <Edit className="h-4 w-4" />
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => {
                                setSelectedDSN(dsn);
                                setDeleteDialogOpen(true);
                              }}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </TableCell>
                      )}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={addDialogOpen} onOpenChange={setAddDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add SQL Server DSN</DialogTitle>
            <DialogDescription>Create a new User DSN</DialogDescription>
          </DialogHeader>
          <DSNForm form={form} setForm={setForm} />
          <DialogFooter>
            <Button variant="outline" onClick={() => setAddDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="outline"
              onClick={() => {
                const payload = validateAndGetPayload();
                if (payload) {
                  testMutation.mutate(payload);
                }
              }}
              disabled={testMutation.isPending || createMutation.isPending}
            >
              {testMutation.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <PlugZap className="mr-2 h-4 w-4" />
              )}
              Test
            </Button>
            <Button
              onClick={() => {
                const payload = validateAndGetPayload();
                if (payload) {
                  createMutation.mutate(payload);
                }
              }}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit SQL Server DSN</DialogTitle>
            <DialogDescription>Update DSN values</DialogDescription>
          </DialogHeader>
          <DSNForm form={form} setForm={setForm} disableName />
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="outline"
              onClick={() => {
                const payload = validateAndGetPayload();
                if (payload) {
                  testMutation.mutate(payload);
                }
              }}
              disabled={testMutation.isPending || updateMutation.isPending}
            >
              {testMutation.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <PlugZap className="mr-2 h-4 w-4" />
              )}
              Test
            </Button>
            <Button
              onClick={() => {
                const payload = validateAndGetPayload();
                if (payload && selectedDSN) {
                  updateMutation.mutate({ name: selectedDSN.name, payload });
                }
              }}
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Update
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete DSN</AlertDialogTitle>
            <AlertDialogDescription>
              Delete &quot;{selectedDSN?.name}&quot;? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (selectedDSN) {
                  deleteMutation.mutate(selectedDSN.name);
                }
              }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function DSNForm({
  form,
  setForm,
  disableName = false,
}: {
  form: SQLServerODBCDSNRequest;
  setForm: (next: SQLServerODBCDSNRequest) => void;
  disableName?: boolean;
}) {
  return (
    <div className="grid gap-4 py-2">
      <div className="space-y-2">
        <Label htmlFor="dsn-name">DSN Name</Label>
        <Input
          id="dsn-name"
          value={form.name}
          disabled={disableName}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="dsn-server">Server</Label>
        <Input
          id="dsn-server"
          value={form.server}
          onChange={(e) => setForm({ ...form, server: e.target.value })}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="dsn-db">Database</Label>
        <Input
          id="dsn-db"
          value={form.database}
          onChange={(e) => setForm({ ...form, database: e.target.value })}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="dsn-login">Login ID</Label>
        <Input
          id="dsn-login"
          value={form.login_id}
          onChange={(e) => setForm({ ...form, login_id: e.target.value })}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="dsn-password">Password</Label>
        <Input
          id="dsn-password"
          type="password"
          value={form.password}
          onChange={(e) => setForm({ ...form, password: e.target.value })}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="dsn-description">Description</Label>
        <Input
          id="dsn-description"
          value={form.description || ''}
          onChange={(e) => setForm({ ...form, description: e.target.value })}
        />
      </div>
    </div>
  );
}
