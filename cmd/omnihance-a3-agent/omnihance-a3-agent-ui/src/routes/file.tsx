import { createRoute, redirect, useSearch } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { lazyNamed, LazySuspense } from '@/lib/lazy';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

const FileTree = lazyNamed(() => import('@/components/file-tree'), 'FileTree');

function FilePageWithLayout() {
  const { path } = useSearch({ from: '/file' });

  return (
    <DashboardLayout>
      <div className="flex h-[calc(100vh-4rem)] flex-col overflow-hidden p-4 lg:p-6">
        <LazySuspense>
          <FileTree key={path ?? '__root__'} initialPath={path} />
        </LazySuspense>
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
