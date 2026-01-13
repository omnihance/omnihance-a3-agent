import { createRoute, redirect } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { lazyNamed, LazySuspense } from '@/lib/lazy';
import { getSession, APIError } from '@/lib/api';
import { APP_NAME } from '@/constants';

const AuthPage = lazyNamed(() => import('@/components/auth-page'), 'AuthPage');

function AuthPageWithSuspense() {
  return (
    <LazySuspense>
      <AuthPage />
    </LazySuspense>
  );
}

export default (parentRoute: AnyRootRoute) =>
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/',
    head: () => ({
      meta: [
        {
          title: `Sign In - ${APP_NAME}`,
        },
      ],
    }),
    beforeLoad: async () => {
      try {
        await getSession();
        throw redirect({
          to: '/dashboard',
        });
      } catch (error) {
        if (error instanceof APIError) {
          return;
        }

        throw error;
      }
    },
    component: AuthPageWithSuspense,
  });
