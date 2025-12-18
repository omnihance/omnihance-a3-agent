import type React from 'react';
import { useState } from 'react';
import { useRouter } from '@tanstack/react-router';
import { useQuery, useMutation } from '@tanstack/react-query';
import { LogOut, Menu } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Sheet,
  SheetContent,
  SheetTrigger,
  SheetTitle,
} from '@/components/ui/sheet';
import { Separator } from '@/components/ui/separator';
import { ThemeToggle } from '@/components/theme-toggle';
import { SidebarContent } from '@/components/sidebar-content';
import { beautifyRole, cn } from '@/lib/utils';
import { signOut, getSession, APIError } from '@/lib/api';
import { queryKeys } from '@/constants';

interface DashboardLayoutProps {
  children: React.ReactNode;
}

export function DashboardLayout({ children }: DashboardLayoutProps) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const router = useRouter();
  const pathname = router.state.location.pathname;

  const { data: session } = useQuery({
    queryKey: queryKeys.session,
    queryFn: getSession,
    retry: false,
  });

  const signOutMutation = useMutation({
    mutationFn: signOut,
    onSuccess: () => {
      router.navigate({ to: '/' });
    },
    onError: (err: unknown) => {
      if (err instanceof APIError) {
        console.error(err.getErrorMessage());
      } else {
        console.error(err instanceof Error ? err.message : 'Sign out failed');
      }
    },
  });

  const handleLogout = () => {
    signOutMutation.mutate();
  };

  const isActive = (href: string) => {
    if (href === '/dashboard') {
      return pathname === '/dashboard';
    }

    if (href === '/settings') {
      return pathname === '/settings';
    }

    return pathname.startsWith(href);
  };

  return (
    <div className="flex min-h-screen bg-background">
      {/* Desktop Sidebar */}
      <aside
        className={cn(
          'hidden h-screen border-r bg-sidebar text-sidebar-foreground transition-all duration-300 md:block',
          collapsed ? 'w-16' : 'w-64',
        )}
      >
        <SidebarContent
          collapsed={collapsed}
          setCollapsed={setCollapsed}
          setSidebarOpen={setSidebarOpen}
          isActive={isActive}
        />
      </aside>

      {/* Mobile Sidebar */}
      <Sheet open={sidebarOpen} onOpenChange={setSidebarOpen}>
        <SheetContent side="left" className="w-64 p-0">
          <SheetTitle className="sr-only">Navigation Menu</SheetTitle>
          <SidebarContent
            collapsed={collapsed}
            setCollapsed={setCollapsed}
            setSidebarOpen={setSidebarOpen}
            isActive={isActive}
          />
        </SheetContent>
      </Sheet>

      {/* Main Content */}
      <div className="flex flex-1 flex-col">
        {/* Top Navbar */}
        <header className="flex h-16 items-center justify-between border-b bg-background px-4 lg:px-6">
          <div className="flex items-center gap-4">
            <Sheet open={sidebarOpen} onOpenChange={setSidebarOpen}>
              <SheetTrigger asChild>
                <Button variant="ghost" size="icon" className="md:hidden">
                  <Menu className="h-5 w-5" />
                  <span className="sr-only">Toggle menu</span>
                </Button>
              </SheetTrigger>
            </Sheet>
          </div>

          <div className="flex items-center gap-4">
            <ThemeToggle />
            <Separator orientation="vertical" className="h-6" />
            <div className="flex items-center gap-3">
              <div className="hidden text-right sm:block">
                <div className="text-sm font-medium">
                  {session?.email || 'user'}
                </div>
                <div className="text-xs text-muted-foreground">
                  {session?.roles
                    ?.map((role) => beautifyRole(role))
                    .join(', ') || 'Viewer'}
                </div>
              </div>
              <Button
                variant="ghost"
                size="icon"
                onClick={handleLogout}
                disabled={signOutMutation.isPending}
              >
                <LogOut className="h-5 w-5" />
                <span className="sr-only">Logout</span>
              </Button>
            </div>
          </div>
        </header>

        {/* Page Content */}
        <main className="flex-1 overflow-auto">{children}</main>
      </div>
    </div>
  );
}
