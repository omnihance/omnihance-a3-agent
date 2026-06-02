import {
  lazy,
  Suspense,
  type ComponentType,
  type ReactNode,
  type LazyExoticComponent,
} from 'react';
import { Loader2 } from 'lucide-react';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function lazyNamed<T extends ComponentType<any>>(
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  importFn: () => Promise<{ [key: string]: any }>,
  name: string,
): LazyExoticComponent<T> {
  return lazy(async () => {
    const module = await importFn();
    return { default: module[name] as T };
  });
}

export function LazySuspense({
  children,
  fallback,
}: {
  children: ReactNode;
  fallback?: ReactNode;
}) {
  return (
    <Suspense
      fallback={
        fallback ?? (
          <output
            className="flex h-96 items-center justify-center"
            aria-live="polite"
            aria-label="Loading content"
          >
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </output>
        )
      }
    >
      {children}
    </Suspense>
  );
}
