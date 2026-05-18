import { createRoute, redirect } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { lazyNamed, LazySuspense } from '@/lib/lazy';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

const BackupPage = lazyNamed(
  () => import('@/components/backup-page'),
  'BackupPage',
);

function BackupPageWithLayout() {
  return (
    <DashboardLayout>
      <LazySuspense>
        <BackupPage />
      </LazySuspense>
    </DashboardLayout>
  );
}

function getErrorStatus(error: unknown) {
  if (error instanceof Response) {
    return error.status;
  }

  if (error && typeof error === 'object' && 'status' in error) {
    const status = error.status;
    return typeof status === 'number' ? status : null;
  }

  return null;
}

function isAuthError(error: unknown) {
  const status = getErrorStatus(error);
  return status === 401 || status === 403;
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/backups',
    head: () => ({
      meta: [
        {
          title: `Backups - ${APP_NAME}`,
        },
      ],
    }),
    beforeLoad: async ({ location }) => {
      try {
        await getSession();
      } catch (error) {
        if (!isAuthError(error)) {
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
    component: BackupPageWithLayout,
  });
