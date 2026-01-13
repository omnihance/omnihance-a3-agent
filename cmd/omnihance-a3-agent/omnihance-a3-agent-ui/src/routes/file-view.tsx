import { createRoute, redirect, useSearch } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { lazyNamed, LazySuspense } from '@/lib/lazy';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

const FileView = lazyNamed(() => import('@/components/file-view'), 'FileView');
const PathError = lazyNamed(
  () => import('@/components/path-error'),
  'PathError',
);

function FileViewPageWithLayout() {
  const { path } = useSearch({ from: '/file/view' });

  if (!path) {
    return (
      <DashboardLayout>
        <LazySuspense>
          <PathError
            title="File Path Required"
            description="No file path was provided. Please select a file from the project directory to view."
          />
        </LazySuspense>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <LazySuspense>
        <FileView filePath={path} />
      </LazySuspense>
    </DashboardLayout>
  );
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/file/view',
    validateSearch: (search: Record<string, unknown>) => {
      return {
        path: (search.path as string) || '',
      };
    },
    head: () => ({
      meta: [
        {
          title: `File View - ${APP_NAME}`,
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
    component: FileViewPageWithLayout,
  });
