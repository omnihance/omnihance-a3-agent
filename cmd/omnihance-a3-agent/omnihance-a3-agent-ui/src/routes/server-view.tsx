import { createRoute, redirect, useParams } from '@tanstack/react-router';
import type { AnyRootRoute } from '@tanstack/react-router';
import type React from 'react';
import { lazyNamed, LazySuspense } from '@/lib/lazy';
import { DashboardLayout } from '@/components/dashboard-layout';
import { getSession } from '@/lib/api';
import { APP_NAME } from '@/constants';

const rolePermissions: Record<string, string[]> = {
  view_game_data: ['super_admin', 'admin', 'viewer'],
};

function normalizeRole(role: string): string {
  return role.trim().toLowerCase();
}

function isAllowed(action: string, roles: string[]): boolean {
  const allowedRoles = rolePermissions[action];
  if (!allowedRoles) {
    return false;
  }

  const allowedRolesMap = new Set(
    allowedRoles.map((role) => normalizeRole(role)),
  );

  return roles.some((role) => allowedRolesMap.has(normalizeRole(role)));
}

const ServerViewPage = lazyNamed(
  () => import('@/components/server-view-page'),
  'ServerViewPage',
);
const ServerViewMainMapsPage = lazyNamed(
  () => import('@/components/server-view-page'),
  'ServerViewMainMapsPage',
);
const ServerViewZoneMapsPage = lazyNamed(
  () => import('@/components/server-view-page'),
  'ServerViewZoneMapsPage',
);
const ServerViewZoneSpawnsPage = lazyNamed(
  () => import('@/components/server-view-page'),
  'ServerViewZoneSpawnsPage',
);
const ServerViewZoneSpawnDetailsPage = lazyNamed(
  () => import('@/components/server-view-page'),
  'ServerViewZoneSpawnDetailsPage',
);
const ServerViewZoneDropsPage = lazyNamed(
  () => import('@/components/server-view-page'),
  'ServerViewZoneDropsPage',
);
const ServerViewZoneDropDetailsPage = lazyNamed(
  () => import('@/components/server-view-page'),
  'ServerViewZoneDropDetailsPage',
);
const ServerViewZoneShopsPage = lazyNamed(
  () => import('@/components/server-view-page'),
  'ServerViewZoneShopsPage',
);
const ServerViewZoneShopDetailsPage = lazyNamed(
  () => import('@/components/server-view-page'),
  'ServerViewZoneShopDetailsPage',
);

function PageLayout({ children }: { children: React.ReactNode }) {
  return (
    <DashboardLayout>
      <LazySuspense>{children}</LazySuspense>
    </DashboardLayout>
  );
}

function OverviewRouteComponent() {
  return (
    <PageLayout>
      <ServerViewPage />
    </PageLayout>
  );
}

function MainMapsRouteComponent() {
  return (
    <PageLayout>
      <ServerViewMainMapsPage />
    </PageLayout>
  );
}

function ZoneMapsRouteComponent() {
  return (
    <PageLayout>
      <ServerViewZoneMapsPage />
    </PageLayout>
  );
}

function ZoneSpawnsRouteComponent() {
  return (
    <PageLayout>
      <ServerViewZoneSpawnsPage />
    </PageLayout>
  );
}

function ZoneSpawnDetailsRouteComponent() {
  const { mapId } = useParams({
    from: '/server-view/zone/spawns/$mapId',
  });

  return (
    <PageLayout>
      <ServerViewZoneSpawnDetailsPage mapId={Number(mapId)} />
    </PageLayout>
  );
}

function ZoneDropsRouteComponent() {
  return (
    <PageLayout>
      <ServerViewZoneDropsPage />
    </PageLayout>
  );
}

function ZoneDropDetailsRouteComponent() {
  const { npcId } = useParams({
    from: '/server-view/zone/drops/$npcId',
  });

  return (
    <PageLayout>
      <ServerViewZoneDropDetailsPage npcId={Number(npcId)} />
    </PageLayout>
  );
}

function ZoneShopsRouteComponent() {
  return (
    <PageLayout>
      <ServerViewZoneShopsPage />
    </PageLayout>
  );
}

function ZoneShopDetailsRouteComponent() {
  const { npcId } = useParams({
    from: '/server-view/zone/shops/$npcId',
  });

  return (
    <PageLayout>
      <ServerViewZoneShopDetailsPage npcId={Number(npcId)} />
    </PageLayout>
  );
}

async function beforeLoadServerView({
  location,
}: {
  location: { href: string };
}) {
  try {
    const session = await getSession();
    if (!isAllowed('view_game_data', session.roles)) {
      throw redirect({
        to: '/dashboard',
      });
    }
  } catch (error) {
    if (error && typeof error === 'object' && 'to' in error) {
      throw error;
    }

    throw redirect({
      to: '/',
      search: {
        redirect: location.href,
      },
    });
  }
}

export default (parentRoute: AnyRootRoute) => [
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/server-view',
    head: () => ({
      meta: [{ title: `Server View - ${APP_NAME}` }],
    }),
    beforeLoad: beforeLoadServerView,
    component: OverviewRouteComponent,
  }),
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/server-view/main/maps',
    head: () => ({
      meta: [{ title: `Main Server Maps - ${APP_NAME}` }],
    }),
    beforeLoad: beforeLoadServerView,
    component: MainMapsRouteComponent,
  }),
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/server-view/zone/maps',
    head: () => ({
      meta: [{ title: `Zone Server Maps - ${APP_NAME}` }],
    }),
    beforeLoad: beforeLoadServerView,
    component: ZoneMapsRouteComponent,
  }),
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/server-view/zone/spawns',
    head: () => ({
      meta: [{ title: `Monster Spawns - ${APP_NAME}` }],
    }),
    beforeLoad: beforeLoadServerView,
    component: ZoneSpawnsRouteComponent,
  }),
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/server-view/zone/spawns/$mapId',
    head: () => ({
      meta: [{ title: `Monster Spawn Details - ${APP_NAME}` }],
    }),
    beforeLoad: beforeLoadServerView,
    component: ZoneSpawnDetailsRouteComponent,
  }),
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/server-view/zone/drops',
    head: () => ({
      meta: [{ title: `Monster Drops - ${APP_NAME}` }],
    }),
    beforeLoad: beforeLoadServerView,
    component: ZoneDropsRouteComponent,
  }),
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/server-view/zone/drops/$npcId',
    head: () => ({
      meta: [{ title: `Monster Drop Details - ${APP_NAME}` }],
    }),
    beforeLoad: beforeLoadServerView,
    component: ZoneDropDetailsRouteComponent,
  }),
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/server-view/zone/shops',
    head: () => ({
      meta: [{ title: `Shops - ${APP_NAME}` }],
    }),
    beforeLoad: beforeLoadServerView,
    component: ZoneShopsRouteComponent,
  }),
  createRoute({
    getParentRoute: () => parentRoute,
    path: '/server-view/zone/shops/$npcId',
    head: () => ({
      meta: [{ title: `Shop Details - ${APP_NAME}` }],
    }),
    beforeLoad: beforeLoadServerView,
    component: ZoneShopDetailsRouteComponent,
  }),
];
