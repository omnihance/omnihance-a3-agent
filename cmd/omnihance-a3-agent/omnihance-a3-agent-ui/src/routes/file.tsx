import { createRoute, redirect, useSearch } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { FileTree } from '@/components/file-tree';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

function FilePageWithLayout() {
  const { path } = useSearch({ from: '/file' });

  return (
    <DashboardLayout>
      <div className="flex h-[calc(100vh-4rem)] flex-col overflow-hidden p-4 lg:p-6">
        <FileTree initialPath={path} />
      </div>
    </DashboardLayout>
  );
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/file',
    validateSearch: (search: Record<string, unknown>) => {
      return {
        path: (search.path as string) || undefined,
      };
    },
    head: () => ({
      meta: [
        {
          title: `File Browser - ${APP_NAME}`,
        },
      ],
    }),
    beforeLoad: async ({ location }) => {
      try {
        await getSession();
      } catch {
        throw redirect({
          to: '/',
          search: {
            redirect: location.href,
          },
        });
      }
    },
    component: FilePageWithLayout,
  });
