import { MonsterFileUpload } from '@/components/client-data/monster-file-upload';
import { MapFileUpload } from '@/components/client-data/map-file-upload';
import { ItemFileUpload } from '@/components/client-data/item-file-upload';
import { usePermissions } from '@/hooks/use-permissions';
import { useQuery } from '@tanstack/react-query';
import { queryKeys } from '@/constants';
import {
  getGameClientDataCounts,
  uploadIt0File,
  uploadIt1File,
  uploadIt2File,
  uploadIt3File,
} from '@/lib/api';
import { useStatus } from '@/hooks/use-status';

export function ClientDataPage() {
  const { hasPermission } = usePermissions();
  const { status } = useStatus();
  const canUploadGameData = hasPermission('upload_game_data');
  const maxFileUploadSizeBytes = status?.max_file_upload_size_bytes;
  const {
    data: counts,
    isLoading: countsLoading,
    isError: countsError,
  } = useQuery({
    queryKey: queryKeys.gameClientDataCounts,
    queryFn: getGameClientDataCounts,
    enabled: canUploadGameData,
  });

  return (
    <div className="p-4 lg:p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Client Data</h1>
        <p className="text-muted-foreground">
          Upload game client files to populate the database
        </p>
      </div>

      {canUploadGameData ? (
        <div className="grid gap-6 md:grid-cols-2">
          <MonsterFileUpload
            existingCount={counts?.monsters}
            countLoading={countsLoading}
            countError={countsError}
            maxFileUploadSizeBytes={maxFileUploadSizeBytes}
          />
          <MapFileUpload
            existingCount={counts?.maps}
            countLoading={countsLoading}
            countError={countsError}
            maxFileUploadSizeBytes={maxFileUploadSizeBytes}
          />
          <ItemFileUpload
            fileLabel="IT0"
            existingCount={counts?.items.it0}
            countLoading={countsLoading}
            countError={countsError}
            uploadFile={uploadIt0File}
            maxFileUploadSizeBytes={maxFileUploadSizeBytes}
          />
          <ItemFileUpload
            fileLabel="IT1"
            existingCount={counts?.items.it1}
            countLoading={countsLoading}
            countError={countsError}
            uploadFile={uploadIt1File}
            maxFileUploadSizeBytes={maxFileUploadSizeBytes}
          />
          <ItemFileUpload
            fileLabel="IT2"
            existingCount={counts?.items.it2}
            countLoading={countsLoading}
            countError={countsError}
            uploadFile={uploadIt2File}
            maxFileUploadSizeBytes={maxFileUploadSizeBytes}
          />
          <ItemFileUpload
            fileLabel="IT3"
            existingCount={counts?.items.it3}
            countLoading={countsLoading}
            countError={countsError}
            uploadFile={uploadIt3File}
            maxFileUploadSizeBytes={maxFileUploadSizeBytes}
          />
        </div>
      ) : (
        <div className="text-center py-12 text-muted-foreground">
          You don't have permission to upload game data files.
        </div>
      )}
    </div>
  );
}
