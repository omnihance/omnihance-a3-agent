import { Database, Loader2, AlertCircle } from 'lucide-react';
import { Badge } from '@/components/ui/badge';

type ClientDataCountBadgeProps = {
  count?: number;
  isLoading: boolean;
  isError: boolean;
};

export function ClientDataCountBadge({
  count,
  isLoading,
  isError,
}: ClientDataCountBadgeProps) {
  if (isLoading) {
    return (
      <Badge variant="secondary" className="gap-1.5 whitespace-nowrap">
        <Loader2 className="h-3 w-3 animate-spin" />
        Checking records
      </Badge>
    );
  }

  if (isError) {
    return (
      <Badge variant="outline" className="gap-1.5 whitespace-nowrap">
        <AlertCircle className="h-3 w-3" />
        Count unavailable
      </Badge>
    );
  }

  return (
    <Badge
      variant={count ? 'secondary' : 'outline'}
      className="gap-1.5 whitespace-nowrap"
    >
      <Database className="h-3 w-3" />
      {count
        ? `${count.toLocaleString()} records existing`
        : 'No records existing'}
    </Badge>
  );
}
