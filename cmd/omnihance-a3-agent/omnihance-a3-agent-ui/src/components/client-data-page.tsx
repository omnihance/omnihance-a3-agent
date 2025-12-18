import { MonsterFileUpload } from '@/components/client-data/monster-file-upload';
import { MapFileUpload } from '@/components/client-data/map-file-upload';

export function ClientDataPage() {
  return (
    <div className="p-4 lg:p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Client Data</h1>
        <p className="text-muted-foreground">
          Upload game client files to populate the database
        </p>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <MonsterFileUpload />
        <MapFileUpload />
      </div>
    </div>
  );
}
