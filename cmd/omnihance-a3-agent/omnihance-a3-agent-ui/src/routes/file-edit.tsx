import { createRoute, redirect, useSearch } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { FileEdit } from '@/components/file-edit';
import { PathError } from '@/components/path-error';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

function FileEditPageWithLayout() {
  const { path } = useSearch({ from: '/file/edit' });

  if (!path) {
    return (
      <DashboardLayout>
        <PathError
          title="File Path Required"
          description="No file path was provided. Please select a file from the project directory to edit."
        />
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <FileEdit filePath={path} />
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
    component: FileEditPageWithLayout,
  });
