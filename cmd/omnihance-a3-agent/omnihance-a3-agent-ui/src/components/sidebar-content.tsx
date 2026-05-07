import { Link, useRouterState } from '@tanstack/react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Server,
  LayoutDashboard,
  Settings,
  ChevronLeft,
  FolderOpen,
  Database,
  Users,
  PlaySquare,
  DatabaseZap,
  X,
  Folder,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
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
import { cn } from '@/lib/utils';
import { usePermissions } from '@/hooks/use-permissions';
import {
  getDirectoryShortcuts,
  deleteDirectoryShortcut,
  APIError,
  type DirectoryShortcutsResponse,
} from '@/lib/api';
import { queryKeys } from '@/constants';
import { toast } from 'sonner';
import { useState } from 'react';

const sidebarLinks = [
  {
    href: '/dashboard',
    icon: LayoutDashboard,
    label: 'Dashboard',
    permission: 'view_metrics' as const,
  },
  {
    href: '/file',
    icon: FolderOpen,
    label: 'File Browser',
    permission: 'view_files' as const,
  },
  {
    href: '/manage-server',
    icon: PlaySquare,
    label: 'Server Management',
    permission: 'view_files' as const,
  },
  {
    href: '/sql-server-odbc',
    icon: DatabaseZap,
    label: 'SQL ODBC',
    permission: 'manage_server' as const,
  },
  {
    href: '/client-data',
    icon: Database,
    label: 'Client Data',
    permission: 'upload_game_data' as const,
  },
  {
    href: '/users',
    icon: Users,
    label: 'Users',
    permission: 'manage_users' as const,
  },
];

const bottomLinks = [{ href: '/settings', icon: Settings, label: 'Settings' }];

interface SidebarContentProps {
  collapsed: boolean;
  setCollapsed: (collapsed: boolean) => void;
  setSidebarOpen: (open: boolean) => void;
  isActive: (href: string) => boolean;
}

export function SidebarContent({
  collapsed,
  setCollapsed,
  setSidebarOpen,
  isActive,
}: SidebarContentProps) {
  const { hasPermission } = usePermissions();
  const queryClient = useQueryClient();
  const [deleteShortcutId, setDeleteShortcutId] = useState<number | null>(null);
  const routerState = useRouterState();
  const currentPath = routerState.location.search?.path as string | undefined;

  const visibleSidebarLinks = sidebarLinks.filter((link) =>
    hasPermission(link.permission),
  );

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

  const shortcuts = directoryShortcutsData?.shortcuts || [];
  const limit = directoryShortcutsData?.limit || 0;
  const overLimitBy = directoryShortcutsData?.over_limit_by || 0;
  const visibleShortcuts = limit > 0 ? shortcuts.slice(0, limit) : shortcuts;

  return (
    <div className="flex h-full flex-col">
      {/* Logo */}
      <div
        className={cn(
          'flex h-16 items-center border-b px-4',
          collapsed ? 'justify-center' : 'gap-2',
        )}
      >
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary">
          <Server className="h-5 w-5 text-primary-foreground" />
        </div>
        {!collapsed && <span className="text-xl font-bold">Omnihance</span>}
      </div>

      {/* Navigation */}
      <ScrollArea className="flex-1 px-3 py-4">
        <nav className="flex flex-col gap-1">
          {visibleSidebarLinks.map((link) => (
            <Link
              key={link.href}
              to={link.href}
              onClick={() => setSidebarOpen(false)}
              className={cn(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                isActive(link.href)
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                collapsed && 'justify-center px-2',
              )}
            >
              <link.icon className="h-5 w-5 shrink-0" />
              {!collapsed && <span>{link.label}</span>}
            </Link>
          ))}
          {visibleShortcuts.length > 0 && (
            <>
              <div className="my-2 border-t" />
              {!collapsed && (
                <div className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Shortcuts
                </div>
              )}
              {visibleShortcuts.map((shortcut) => (
                <div
                  key={shortcut.id}
                  className={cn(
                    'group flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                    isActive('/file') &&
                      currentPath &&
                      currentPath.replace(/\\/g, '/').toLowerCase().trim() ===
                        shortcut.path.replace(/\\/g, '/').toLowerCase().trim()
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                    collapsed && 'justify-center px-2',
                  )}
                >
                  <Link
                    to="/file"
                    search={{ path: shortcut.path }}
                    onClick={() => setSidebarOpen(false)}
                    className={cn(
                      'flex flex-1 items-center gap-3 min-w-0',
                      collapsed && 'justify-center',
                    )}
                    title={collapsed ? shortcut.name : undefined}
                  >
                    <Folder className="h-4 w-4 shrink-0" />
                    {!collapsed && (
                      <span className="truncate">{shortcut.name}</span>
                    )}
                  </Link>
                  {!collapsed && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 w-6 shrink-0 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        setDeleteShortcutId(shortcut.id);
                      }}
                      aria-label={`Remove ${shortcut.name}`}
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  )}
                </div>
              ))}
              {overLimitBy > 0 && !collapsed && (
                <div className="px-3 py-1 text-xs text-muted-foreground">
                  Over limit by {overLimitBy} — manage in Settings
                </div>
              )}
            </>
          )}
        </nav>
      </ScrollArea>

      {/* Bottom Links and Collapse Button Container */}
      <div className="mt-auto shrink-0">
        {/* Bottom Links */}
        <div className="border-t px-3 py-4">
          <nav className="flex flex-col gap-1">
            {bottomLinks.map((link) => (
              <Link
                key={link.href}
                to={link.href}
                onClick={() => setSidebarOpen(false)}
                className={cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                  isActive(link.href)
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                  collapsed && 'justify-center px-2',
                )}
              >
                <link.icon className="h-5 w-5 shrink-0" />
                {!collapsed && <span>{link.label}</span>}
              </Link>
            ))}
          </nav>
        </div>

        {/* Collapse Button (Desktop) */}
        <div className="hidden border-t p-2 md:block">
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-center"
            onClick={() => setCollapsed(!collapsed)}
          >
            <ChevronLeft
              className={cn(
                'h-4 w-4 transition-transform',
                collapsed && 'rotate-180',
              )}
            />
          </Button>
        </div>
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
