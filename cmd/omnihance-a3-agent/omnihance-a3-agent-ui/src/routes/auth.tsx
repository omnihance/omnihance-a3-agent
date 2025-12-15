import { createRoute, redirect } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import { AuthPage } from '@/components/auth-page';
import { getSession, APIError } from '@/lib/api';
import { APP_NAME } from '@/constants';

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
    component: AuthPage,
  });
