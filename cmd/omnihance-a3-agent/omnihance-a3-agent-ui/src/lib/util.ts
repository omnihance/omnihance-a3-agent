import { useEffect, useState } from 'react';
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

const relativeTimeFormatter = new Intl.RelativeTimeFormat('en-US', {
  numeric: 'auto',
});

const minuteMs = 60 * 1000;
const hourMs = 60 * minuteMs;
const dayMs = 24 * hourMs;
const weekMs = 7 * dayMs;
const monthMs = 30 * dayMs;
const yearMs = 365 * dayMs;

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatBytes(bytes: number, decimals = 2) {
  if (!Number.isFinite(bytes) || bytes < 0) {
    return '0 Bytes';
  }

  const k = 1024;
  const dm = Number.isFinite(decimals)
    ? Math.min(Math.max(0, Math.trunc(decimals)), 20)
    : 0;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  let i = 0;

  if (bytes >= 1) {
    i = Math.floor(Math.log(bytes) / Math.log(k));
    i = Math.min(Math.max(i, 0), sizes.length - 1);
  }

  return (
    Number.parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
  );
}

export function formatDate(date: string | Date | null | undefined) {
  return formatExactDateTime(date);
}

export function formatExactDateTime(
  value: string | Date | null | undefined,
  fallback = '-',
) {
  const date = parseDateValue(value);
  if (!date) {
    return typeof value === 'string' && value ? value : fallback;
  }

  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(date);
}

export function formatRelativeDateTime(
  value: string | Date | null | undefined,
  fallback = '-',
) {
  const date = parseDateValue(value);
  if (!date) {
    return typeof value === 'string' && value ? value : fallback;
  }

  const diffMs = date.getTime() - Date.now();
  const absMs = Math.abs(diffMs);

  if (absMs < minuteMs) {
    return 'just now';
  }

  if (absMs < hourMs) {
    return relativeTimeFormatter.format(
      Math.round(diffMs / minuteMs),
      'minute',
    );
  }

  if (absMs < dayMs) {
    return relativeTimeFormatter.format(Math.round(diffMs / hourMs), 'hour');
  }

  if (absMs < weekMs) {
    return relativeTimeFormatter.format(Math.round(diffMs / dayMs), 'day');
  }

  if (absMs < monthMs) {
    return relativeTimeFormatter.format(Math.round(diffMs / weekMs), 'week');
  }

  if (absMs < yearMs) {
    return relativeTimeFormatter.format(Math.round(diffMs / monthMs), 'month');
  }

  return relativeTimeFormatter.format(Math.round(diffMs / yearMs), 'year');
}

export function formatStatusLabel(status: string) {
  return status
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export function maskSecret(value: string) {
  return '*'.repeat(Math.max(8, Math.min(value.length, 32)));
}

export function emptyToNull(value: string): string | null {
  const trimmed = value.trim();
  return trimmed ? trimmed : null;
}

export function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debouncedValue, setDebouncedValue] = useState(value);

  useEffect(() => {
    const timeoutID = window.setTimeout(() => {
      setDebouncedValue(value);
    }, delayMs);

    return () => window.clearTimeout(timeoutID);
  }, [delayMs, value]);

  return debouncedValue;
}

export function beautifyRole(role: string) {
  return role
    .replace(/_/g, ' ')
    .replace(
      /\w\S*/g,
      (txt) => txt.charAt(0).toUpperCase() + txt.slice(1).toLowerCase(),
    );
}

function parseDateValue(value: string | Date | null | undefined): Date | null {
  if (!value) {
    return null;
  }

  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }

  return date;
}
