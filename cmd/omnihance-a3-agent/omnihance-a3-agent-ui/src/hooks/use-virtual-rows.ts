import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

interface UseVirtualRowsOptions {
  count: number;
  rowHeight: number;
  overscan?: number;
}

export interface VirtualRow {
  index: number;
  top: number;
}

export function useVirtualRows({
  count,
  rowHeight,
  overscan = 6,
}: UseVirtualRowsOptions) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const [viewportHeight, setViewportHeight] = useState(0);

  const measureViewport = useCallback(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }

    setViewportHeight(container.clientHeight);
  }, []);

  useEffect(() => {
    measureViewport();

    const container = containerRef.current;
    if (!container || typeof ResizeObserver === 'undefined') {
      return;
    }

    const observer = new ResizeObserver(measureViewport);
    observer.observe(container);
    return () => observer.disconnect();
  }, [measureViewport]);

  const onScroll = useCallback(() => {
    setScrollTop(containerRef.current?.scrollTop ?? 0);
  }, []);

  const totalHeight = count * rowHeight;
  const virtualRows = useMemo(() => {
    if (count === 0 || viewportHeight === 0) {
      return [];
    }

    const maxScrollTop = Math.max(0, totalHeight - viewportHeight);
    const effectiveScrollTop = Math.min(scrollTop, maxScrollTop);
    const startIndex = Math.max(
      0,
      Math.floor(effectiveScrollTop / rowHeight) - overscan,
    );
    const endIndex = Math.min(
      count - 1,
      Math.ceil((effectiveScrollTop + viewportHeight) / rowHeight) + overscan,
    );
    const rows: VirtualRow[] = [];
    for (let index = startIndex; index <= endIndex; index += 1) {
      rows.push({
        index,
        top: index * rowHeight,
      });
    }

    return rows;
  }, [count, overscan, rowHeight, scrollTop, totalHeight, viewportHeight]);

  return {
    containerRef,
    onScroll,
    totalHeight,
    virtualRows,
  };
}
