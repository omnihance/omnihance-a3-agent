import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getSession } from '@/lib/api';
import { queryKeys } from '@/constants';

type PermissionAction =
  | 'view_files'
  | 'edit_files'
  | 'revert_files'
  | 'upload_game_data'
  | 'manage_users'
  | 'view_metrics'
  | 'view_game_data'
  | 'manage_server';

const rolePermissions: Record<PermissionAction, string[]> = {
  view_files: ['super_admin', 'admin', 'viewer'],
  edit_files: ['super_admin', 'admin'],
  revert_files: ['super_admin', 'admin'],
  upload_game_data: ['super_admin', 'admin'],
  manage_users: ['super_admin'],
  view_metrics: ['super_admin', 'admin', 'viewer'],
  view_game_data: ['super_admin', 'admin', 'viewer'],
  manage_server: ['super_admin', 'admin'],
};

function normalizeRole(role: string): string {
  return role.trim().toLowerCase();
}

function isAllowed(action: PermissionAction, roles: string[]): boolean {
  const allowedRoles = rolePermissions[action];
  if (!allowedRoles) {
    return false;
  }

  const allowedRolesMap = new Set(
    allowedRoles.map((role) => normalizeRole(role)),
  );

  return roles.some((role) => allowedRolesMap.has(normalizeRole(role)));
}

export function usePermissions() {
  const { data: session } = useQuery({
    queryKey: queryKeys.session,
    queryFn: getSession,
    retry: false,
  });

  const roles = useMemo(() => session?.roles || [], [session?.roles]);

  const hasPermission = useMemo(
    () => (action: PermissionAction) => isAllowed(action, roles),
    [roles],
  );

  return {
    roles,
    hasPermission,
    isSuperAdmin: roles.includes('super_admin'),
    isAdmin: roles.includes('admin') || roles.includes('super_admin'),
    isViewer: roles.includes('viewer'),
  };
}
