import * as React from 'react';

const MOBILE_BREAKPOINT = 768;
const mobileMediaQuery = `(max-width: ${MOBILE_BREAKPOINT - 1}px)`;

export function useIsMobile() {
  return React.useSyncExternalStore(
    subscribeToMobileMediaQuery,
    getMobileMediaQuerySnapshot,
    getServerMobileMediaQuerySnapshot,
  );
}

function subscribeToMobileMediaQuery(onStoreChange: () => void) {
  const mediaQueryList = window.matchMedia(mobileMediaQuery);
  mediaQueryList.addEventListener('change', onStoreChange);

  return () => mediaQueryList.removeEventListener('change', onStoreChange);
}

function getMobileMediaQuerySnapshot() {
  return window.matchMedia(mobileMediaQuery).matches;
}

function getServerMobileMediaQuerySnapshot() {
  return false;
}
