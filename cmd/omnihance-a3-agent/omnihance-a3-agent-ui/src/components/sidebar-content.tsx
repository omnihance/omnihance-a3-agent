import { Link } from '@tanstack/react-router';
import {
  Server,
  LayoutDashboard,
  Settings,
  ChevronLeft,
  FolderOpen,
  Database,
  Users,
  PlaySquare,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import { usePermissions } from '@/hooks/use-permissions';

const sidebarLinks = [
  {
    href: '/dashboard',
    icon: LayoutDashboard,
    label: 'Dashboard',
    permission: 'view_metrics' as const,
  },
  {
    href: '/file',
    icon: FolderOpen,
    label: 'File Browser',
    permission: 'view_files' as const,
  },
  {
    href: '/manage-server',
    icon: PlaySquare,
    label: 'Server Management',
    permission: 'view_files' as const,
  },
  {
    href: '/client-data',
    icon: Database,
    label: 'Client Data',
    permission: 'upload_game_data' as const,
  },
  {
    href: '/users',
    icon: Users,
    label: 'Users',
    permission: 'manage_users' as const,
  },
];

const bottomLinks = [{ href: '/settings', icon: Settings, label: 'Settings' }];

interface SidebarContentProps {
  collapsed: boolean;
  setCollapsed: (collapsed: boolean) => void;
  setSidebarOpen: (open: boolean) => void;
  isActive: (href: string) => boolean;
}

export function SidebarContent({
  collapsed,
  setCollapsed,
  setSidebarOpen,
  isActive,
}: SidebarContentProps) {
  const { hasPermission } = usePermissions();

  const visibleSidebarLinks = sidebarLinks.filter((link) =>
    hasPermission(link.permission),
  );

  return (
    <div className="flex h-full flex-col">
      {/* Logo */}
      <div
        className={cn(
          'flex h-16 items-center border-b px-4',
          collapsed ? 'justify-center' : 'gap-2',
        )}
      >
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary">
          <Server className="h-5 w-5 text-primary-foreground" />
        </div>
        {!collapsed && <span className="text-xl font-bold">Omnihance</span>}
      </div>

      {/* Navigation */}
      <ScrollArea className="flex-1 px-3 py-4">
        <nav className="flex flex-col gap-1">
          {visibleSidebarLinks.map((link) => (
            <Link
              key={link.href}
              to={link.href}
              onClick={() => setSidebarOpen(false)}
              className={cn(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                isActive(link.href)
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                collapsed && 'justify-center px-2',
              )}
            >
              <link.icon className="h-5 w-5 shrink-0" />
              {!collapsed && <span>{link.label}</span>}
            </Link>
          ))}
        </nav>
      </ScrollArea>

      {/* Bottom Links and Collapse Button Container */}
      <div className="mt-auto shrink-0">
        {/* Bottom Links */}
        <div className="border-t px-3 py-4">
          <nav className="flex flex-col gap-1">
            {bottomLinks.map((link) => (
              <Link
                key={link.href}
                to={link.href}
                onClick={() => setSidebarOpen(false)}
                className={cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                  isActive(link.href)
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                  collapsed && 'justify-center px-2',
                )}
              >
                <link.icon className="h-5 w-5 shrink-0" />
                {!collapsed && <span>{link.label}</span>}
              </Link>
            ))}
          </nav>
        </div>

        {/* Collapse Button (Desktop) */}
        <div className="hidden border-t p-2 md:block">
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-center"
            onClick={() => setCollapsed(!collapsed)}
          >
            <ChevronLeft
              className={cn(
                'h-4 w-4 transition-transform',
                collapsed && 'rotate-180',
              )}
            />
          </Button>
        </div>
      </div>
    </div>
  );
}
