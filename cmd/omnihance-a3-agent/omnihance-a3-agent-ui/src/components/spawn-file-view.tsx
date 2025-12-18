import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { MapPin, Navigation, Hash, Compass } from 'lucide-react';
import { getMonsters } from '@/lib/api';
import type { SpawnFileAPIData } from '@/lib/api';
import { useMemo } from 'react';
import { queryKeys } from '@/constants';

interface SpawnFileViewProps {
  data: SpawnFileAPIData;
}

export function SpawnFileView({ data }: SpawnFileViewProps) {
  const { data: monsters } = useQuery({
    queryKey: queryKeys.monsters,
    queryFn: () => getMonsters(),
  });

  const monsterMap = useMemo(() => {
    if (!monsters) {
      return new Map<number, string>();
    }

    const map = new Map<number, string>();
    for (const monster of monsters) {
      map.set(monster.id, monster.name);
    }

    return map;
  }, [monsters]);

  const getMonsterDisplay = (id: number): string => {
    const monsterName = monsterMap.get(id);
    if (monsterName) {
      return `${monsterName} (${id})`;
    }

    return `${id}`;
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MapPin className="h-5 w-5" />
            Spawn Points ({data.spawns.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {data.spawns.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>#</TableHead>
                  <TableHead className="text-right">NPC ID</TableHead>
                  <TableHead className="text-right">X</TableHead>
                  <TableHead className="text-right">Y</TableHead>
                  <TableHead className="text-right">Orientation</TableHead>
                  <TableHead className="text-right">Spawn Step</TableHead>
                  <TableHead className="text-right">Unknown1</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.spawns.map((spawn, index) => (
                  <TableRow key={index}>
                    <TableCell className="font-medium">{index + 1}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Hash className="h-3 w-3 text-muted-foreground" />
                        {getMonsterDisplay(spawn.id)}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Navigation className="h-3 w-3 text-muted-foreground" />
                        {spawn.x}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Navigation className="h-3 w-3 text-muted-foreground" />
                        {spawn.y}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Compass className="h-3 w-3 text-muted-foreground" />
                        {spawn.orientation}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      {spawn.spwan_step}
                    </TableCell>
                    <TableCell className="text-right text-muted-foreground">
                      {spawn.unknown1}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              No spawn points configured
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
