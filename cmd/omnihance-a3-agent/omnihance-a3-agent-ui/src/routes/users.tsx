import { createRoute, redirect } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { lazyNamed, LazySuspense } from '@/lib/lazy';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

const rolePermissions: Record<string, string[]> = {
  manage_users: ['super_admin'],
};

function normalizeRole(role: string): string {
  return role.trim().toLowerCase();
}

function isAllowed(action: string, roles: string[]): boolean {
  const allowedRoles = rolePermissions[action];
  if (!allowedRoles) {
    return false;
  }

  const allowedRolesMap = new Set(
    allowedRoles.map((role) => normalizeRole(role)),
  );

  return roles.some((role) => allowedRolesMap.has(normalizeRole(role)));
}

const UsersPage = lazyNamed(
  () => import('@/components/users-page'),
  'UsersPage',
);

function UsersPageWithLayout() {
  return (
    <DashboardLayout>
      <LazySuspense>
        <UsersPage />
      </LazySuspense>
    </DashboardLayout>
  );
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/users',
    head: () => ({
      meta: [
        {
          title: `Users - ${APP_NAME}`,
        },
      ],
    }),
    beforeLoad: async ({ location }) => {
      try {
        const session = await getSession();
        if (!isAllowed('manage_users', session.roles)) {
          throw redirect({
            to: '/dashboard',
          });
        }
      } catch (error) {
        if (error && typeof error === 'object' && 'to' in error) {
          throw error;
        }

        throw redirect({
          to: '/',
          search: {
            redirect: location.href,
          },
        });
      }
    },
    component: UsersPageWithLayout,
  });
