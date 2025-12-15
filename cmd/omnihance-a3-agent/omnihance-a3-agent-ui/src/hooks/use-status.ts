import { useQuery } from '@tanstack/react-query';
import { getStatus, type StatusResponse } from '@/lib/api';

export function useStatus() {
  const query = useQuery<StatusResponse>({
    queryKey: ['status'],
    queryFn: getStatus,
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
  });

  return {
    status: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    isSuccess: query.isSuccess,
    refetch: query.refetch,
  };
}
