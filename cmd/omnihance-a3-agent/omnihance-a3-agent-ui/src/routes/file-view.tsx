import { createRoute, redirect, useSearch } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { FileView } from '@/components/file-view';
import { PathError } from '@/components/path-error';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

function FileViewPageWithLayout() {
  const { path } = useSearch({ from: '/file/view' });

  if (!path) {
    return (
      <DashboardLayout>
        <PathError
          title="File Path Required"
          description="No file path was provided. Please select a file from the project directory to view."
        />
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <FileView filePath={path} />
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
