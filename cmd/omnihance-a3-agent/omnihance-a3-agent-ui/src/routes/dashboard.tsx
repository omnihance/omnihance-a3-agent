import { createRoute, redirect } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { lazyNamed, LazySuspense } from '@/lib/lazy';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

const DashboardPage = lazyNamed(
  () => import('@/components/dashboard-page'),
  'DashboardPage',
);

function DashboardPageWithLayout() {
  return (
    <DashboardLayout>
      <LazySuspense>
        <DashboardPage />
      </LazySuspense>
    </DashboardLayout>
  );
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/dashboard',
    head: () => ({
      meta: [
        {
          title: `Dashboard - ${APP_NAME}`,
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
    component: DashboardPageWithLayout,
  });
