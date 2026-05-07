import { createRoute, redirect } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { lazyNamed, LazySuspense } from '@/lib/lazy';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

const SQLServerODBCPage = lazyNamed(
  () => import('@/components/sql-server-odbc-page'),
  'SQLServerODBCPage',
);

function SQLServerODBCPageWithLayout() {
  return (
    <DashboardLayout>
      <LazySuspense>
        <SQLServerODBCPage />
      </LazySuspense>
    </DashboardLayout>
  );
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/sql-server-odbc',
    head: () => ({
      meta: [
        {
          title: `SQL Server ODBC - ${APP_NAME}`,
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
    component: SQLServerODBCPageWithLayout,
  });
