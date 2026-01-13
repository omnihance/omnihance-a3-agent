import { createRoute, redirect } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { lazyNamed, LazySuspense } from '@/lib/lazy';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

const ManageServerPage = lazyNamed(
  () => import('@/components/manage-server-page'),
  'ManageServerPage',
);

function ManageServerPageWithLayout() {
  return (
    <DashboardLayout>
      <LazySuspense>
        <ManageServerPage />
      </LazySuspense>
    </DashboardLayout>
  );
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/manage-server',
    head: () => ({
      meta: [
        {
          title: `Server Management - ${APP_NAME}`,
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
    component: ManageServerPageWithLayout,
  });
