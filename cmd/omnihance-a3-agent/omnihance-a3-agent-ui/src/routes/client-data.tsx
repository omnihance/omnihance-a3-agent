import { createRoute, redirect } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { ClientDataPage } from '@/components/client-data-page';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

const rolePermissions: Record<string, string[]> = {
  upload_game_data: ['super_admin', 'admin'],
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

function ClientDataPageWithLayout() {
  return (
    <DashboardLayout>
      <ClientDataPage />
    </DashboardLayout>
  );
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/client-data',
    head: () => ({
      meta: [
        {
          title: `Client Data - ${APP_NAME}`,
        },
      ],
    }),
    beforeLoad: async ({ location }) => {
      try {
        const session = await getSession();
        if (!isAllowed('upload_game_data', session.roles)) {
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
    component: ClientDataPageWithLayout,
  });
