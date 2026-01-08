import { createRoute, redirect } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { lazyNamed, LazySuspense } from '@/lib/lazy';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

const SettingsPage = lazyNamed(
  () => import('@/components/settings-page'),
  'SettingsPage',
);

function SettingsPageWithLayout() {
  return (
    <DashboardLayout>
      <LazySuspense>
        <SettingsPage />
      </LazySuspense>
    </DashboardLayout>
  );
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/settings',
    head: () => ({
      meta: [
        {
          title: `Settings - ${APP_NAME}`,
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
    component: SettingsPageWithLayout,
  });
