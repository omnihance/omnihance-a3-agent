import { createRoute, redirect } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { DashboardPage } from '@/components/dashboard-page';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

function DashboardPageWithLayout() {
  return (
    <DashboardLayout>
      <DashboardPage />
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
