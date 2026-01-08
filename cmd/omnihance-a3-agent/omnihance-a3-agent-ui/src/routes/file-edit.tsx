import { createRoute, redirect, useSearch } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { lazyNamed, LazySuspense } from '@/lib/lazy';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

const rolePermissions: Record<string, string[]> = {
  edit_files: ['super_admin', 'admin'],
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

const FileEdit = lazyNamed(() => import('@/components/file-edit'), 'FileEdit');
const PathError = lazyNamed(
  () => import('@/components/path-error'),
  'PathError',
);

function FileEditPageWithLayout() {
  const { path } = useSearch({ from: '/file/edit' });

  if (!path) {
    return (
      <DashboardLayout>
        <LazySuspense>
          <PathError
            title="File Path Required"
            description="No file path was provided. Please select a file from the project directory to edit."
          />
        </LazySuspense>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <LazySuspense>
        <FileEdit filePath={path} />
      </LazySuspense>
    </DashboardLayout>
  );
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/file/edit',
    validateSearch: (search: Record<string, unknown>) => {
      return {
        path: (search.path as string) || '',
      };
    },
    head: () => ({
      meta: [
        {
          title: `File Edit - ${APP_NAME}`,
        },
      ],
    }),
    beforeLoad: async ({ location, search }) => {
      try {
        const session = await getSession();
        if (!isAllowed('edit_files', session.roles)) {
          throw redirect({
            to: '/file/view',
            search: { path: (search.path as string) || '' },
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
    component: FileEditPageWithLayout,
  });
