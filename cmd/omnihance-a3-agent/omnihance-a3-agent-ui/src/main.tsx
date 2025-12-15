import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './styles.css';
import {
  createRootRoute,
  createRouter,
  HeadContent,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router';
import { Toaster } from '@/components/ui/sonner';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import * as TanStackQueryProvider from './integrations/tanstack-query/root-provider';
import authRoute from './routes/auth';
import dashboardRoute from './routes/dashboard';
import fileRoute from './routes/file';
import fileViewRoute from './routes/file-view';
import fileEditRoute from './routes/file-edit';
import reportWebVitals from './reportWebVitals.ts';

const rootRoute = createRootRoute({
  component: () => (
    <>
      <HeadContent />
      <Outlet />
      <TanStackRouterDevtools />
      <Toaster />
    </>
  ),
});

const routeTree = rootRoute.addChildren([
  authRoute(rootRoute),
  dashboardRoute(rootRoute),
  fileRoute(rootRoute),
  fileViewRoute(rootRoute),
  fileEditRoute(rootRoute),
]);

const TanStackQueryProviderContext = TanStackQueryProvider.getContext();
const router = createRouter({
  routeTree,
  context: {
    ...TanStackQueryProviderContext,
  },
  defaultPreload: 'intent',
  scrollRestoration: true,
  defaultStructuralSharing: true,
  defaultPreloadStaleTime: 0,
});

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

const rootElement = document.getElementById('root');
if (rootElement && !rootElement.innerHTML) {
  const root = createRoot(rootElement);
  root.render(
    <StrictMode>
      <TanStackQueryProvider.Provider {...TanStackQueryProviderContext}>
        <RouterProvider router={router} />
      </TanStackQueryProvider.Provider>
    </StrictMode>,
  );
}

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
