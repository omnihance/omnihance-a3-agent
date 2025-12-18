import { createRoute, redirect } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { ClientDataPage } from '@/components/client-data-page';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

function ClientDataPageWithLayout() {
  return (
    <DashboardLayout>
      <ClientDataPage />
    </DashboardLayout>
  );
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/client-data',
    head: () => ({
      meta: [
        {
          title: `Client Data - ${APP_NAME}`,
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
    component: ClientDataPageWithLayout,
  });
