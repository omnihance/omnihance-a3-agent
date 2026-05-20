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
  const [containerEl, setContainerEl] = useState<HTMLDivElement | null>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const [viewportHeight, setViewportHeight] = useState(0);

  const setContainerRef = useCallback((element: HTMLDivElement | null) => {
    containerRef.current = element;
    setContainerEl(element);
  }, []);

  const measureViewport = useCallback(() => {
    if (!containerEl) {
      return;
    }

    setViewportHeight(containerEl.clientHeight);
  }, [containerEl]);

  useEffect(() => {
    const frameID = window.requestAnimationFrame(measureViewport);

    if (!containerEl || typeof ResizeObserver === 'undefined') {
      return () => window.cancelAnimationFrame(frameID);
    }

    const observer = new ResizeObserver(measureViewport);
    observer.observe(containerEl);
    return () => {
      window.cancelAnimationFrame(frameID);
      observer.disconnect();
    };
  }, [containerEl, measureViewport]);

  const onScroll = useCallback(() => {
    setScrollTop(containerRef.current?.scrollTop ?? 0);
  }, []);

  const resetScrollTop = useCallback(() => {
    if (!containerRef.current) {
      return;
    }

    containerRef.current.scrollTop = 0;
    setScrollTop(0);
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
    containerRef: setContainerRef,
    onScroll,
    resetScrollTop,
    totalHeight,
    virtualRows,
  };
}
