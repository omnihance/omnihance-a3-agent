import { useMemo } from 'react';
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
import {
  Award,
  BookOpen,
  Coins,
  Flag,
  Gift,
  Link2,
  Scroll,
  Target,
  Users,
} from 'lucide-react';
import { getMaps, getMonsters, type QuestFileAPIData } from '@/lib/api';
import { UNUSED_CONTINUATION } from '@/lib/api';
import { queryKeys } from '@/constants';

interface QuestFileViewProps {
  data: QuestFileAPIData;
}

const UNUSED_FLAGS = 0xffffffff;
const UNUSED_LEVEL = 0xff;
const UNUSED_UINT8 = 0xff;
const UNUSED_UINT16 = 0xffff;

const OBJECTIVE_TYPE = {
  KILL: 0,
  QUESTITEM: 1,
  BRINGNPC: 2,
  DROP: 3,
  FIND: 4,
} as const;

function formatContinuation(value: number | null | undefined): string {
  if (value == null || value === UNUSED_CONTINUATION) {
    return '-';
  }

  return String(value);
}

function formatLevel(level: number): string {
  return level === UNUSED_LEVEL ? '-' : String(level);
}

function formatUInt8Value(value: number | null | undefined): string {
  return value == null || value === UNUSED_UINT8 ? '-' : String(value);
}

function formatUInt16Value(value: number | null | undefined): string {
  return value == null || value === UNUSED_UINT16 ? '-' : String(value);
}

function formatClientDataValue(
  value: number | null | undefined,
  lookupMap: Map<number, string>,
): string {
  if (value == null || value === UNUSED_UINT16) {
    return '-';
  }

  const name = lookupMap.get(value);
  if (name) {
    return `${name} (${value})`;
  }

  return String(value);
}

function cleanQuestName(name: string): string {
  return name.replace(/\0+$/g, '').trim();
}

function hasDropValue(
  item: number | null | undefined,
  probability: number | null | undefined,
): boolean {
  return (
    item != null &&
    item !== 0 &&
    item !== UNUSED_UINT16 &&
    probability != null &&
    probability !== 0 &&
    probability !== UNUSED_UINT8
  );
}

export function QuestFileView({ data }: QuestFileViewProps) {
  const { data: maps } = useQuery({
    queryKey: queryKeys.maps,
    queryFn: () => getMaps(),
  });

  const { data: monsters } = useQuery({
    queryKey: queryKeys.monsters,
    queryFn: () => getMonsters(),
  });

  const mapLookup = useMemo(() => {
    if (!maps) {
      return new Map<number, string>();
    }

    const map = new Map<number, string>();
    for (const mapItem of maps) {
      map.set(mapItem.id, mapItem.name);
    }

    return map;
  }, [maps]);

  const monsterLookup = useMemo(() => {
    if (!monsters) {
      return new Map<number, string>();
    }

    const map = new Map<number, string>();
    for (const monster of monsters) {
      map.set(monster.id, monster.name);
    }

    return map;
  }, [monsters]);

  const activeRewards = data.reward_items
    .slice(0, 3)
    .map((item, idx) =>
      item != null && item !== UNUSED_UINT16 && data.reward_counts[idx] != null
        ? {
            item,
            count: data.reward_counts[idx] as number,
            index: idx,
          }
        : null,
    )
    .filter(
      (reward): reward is { item: number; count: number; index: number } =>
        reward !== null,
    );

  const usedObjectives = data.objectives.filter((objective) => {
    return !objective.is_unused && objective.type !== UNUSED_UINT8;
  });

  const hasUnusedLevel =
    data.min_level === UNUSED_LEVEL || data.max_level === UNUSED_LEVEL;
  const levelRangeLabel =
    data.min_level === UNUSED_LEVEL && data.max_level === UNUSED_LEVEL
      ? '-'
      : data.min_level === data.max_level && !hasUnusedLevel
        ? `Lv. ${data.min_level}`
        : `Lv. ${formatLevel(data.min_level)}-${formatLevel(data.max_level)}`;
  const levelRangeHint =
    data.min_level === UNUSED_LEVEL && data.max_level === UNUSED_LEVEL
      ? 'Not specified'
      : data.min_level === data.max_level && !hasUnusedLevel
        ? 'Fixed Level'
        : 'Level Range';
  const flagsLabel = `0x${data.flags.toString(16).toUpperCase()}`;
  const showFlags = data.flags !== 0 && data.flags !== UNUSED_FLAGS;

  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Quest ID</CardTitle>
            <BookOpen className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.quest_id}</div>
            <p className="text-xs text-muted-foreground mt-1">
              Quest Identifier
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Giver NPC</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatClientDataValue(data.giver_npc, monsterLookup)}
            </div>
            <p className="text-xs text-muted-foreground mt-1">NPC ID</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Target NPC</CardTitle>
            <Target className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatClientDataValue(data.target_npc, monsterLookup)}
            </div>
            <p className="text-xs text-muted-foreground mt-1">NPC ID</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Level Range</CardTitle>
            <Award className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{levelRangeLabel}</div>
            <p className="text-xs text-muted-foreground mt-1">
              {levelRangeHint}
            </p>
          </CardContent>
        </Card>
      </div>

      {showFlags && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Flags</CardTitle>
            <Flag className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-xl font-semibold">{flagsLabel}</div>
            <p className="text-xs text-muted-foreground mt-1">Quest Flags</p>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Experience</CardTitle>
            <Award className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.exp_reward}</div>
            <p className="text-xs text-muted-foreground mt-1">EXP Reward</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Woonz</CardTitle>
            <Coins className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.woonz_reward}</div>
            <p className="text-xs text-muted-foreground mt-1">Woonz Reward</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Lore</CardTitle>
            <Scroll className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.lore_reward}</div>
            <p className="text-xs text-muted-foreground mt-1">Lore Reward</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Item Rewards</CardTitle>
            <Gift className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{activeRewards.length}</div>
            <p className="text-xs text-muted-foreground mt-1">Active Rewards</p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Gift className="h-5 w-5" />
            Item Rewards
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Slot</TableHead>
                <TableHead className="text-right">Item ID</TableHead>
                <TableHead className="text-right">Count</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {[0, 1, 2].map((idx) => {
                const item = data.reward_items[idx];
                const count = data.reward_counts[idx];
                const isUnused =
                  item == null || item === UNUSED_UINT16 || count == null;
                return (
                  <TableRow key={idx}>
                    <TableCell className="font-medium">
                      Slot {idx + 1}
                    </TableCell>
                    <TableCell className="text-right">
                      {isUnused ? '-' : item}
                    </TableCell>
                    <TableCell className="text-right">
                      {isUnused ? '-' : count}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Target className="h-5 w-5" />
            Objectives ({usedObjectives.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {usedObjectives.map((objective, index) => {
              const cleanName = cleanQuestName(objective.name);
              const supportsTarget =
                objective.type === OBJECTIVE_TYPE.KILL ||
                objective.type === OBJECTIVE_TYPE.BRINGNPC;
              const supportsKill = objective.type === OBJECTIVE_TYPE.KILL;
              const supportsQuestItem =
                objective.type === OBJECTIVE_TYPE.QUESTITEM ||
                objective.type === OBJECTIVE_TYPE.BRINGNPC;
              const visibleDrops = objective.drop_items
                .map((item, idx) => ({
                  item,
                  probability: objective.drop_probs[idx],
                  idx,
                }))
                .filter((drop) => hasDropValue(drop.item, drop.probability));

              return (
                <Card key={index} className="border-l-4">
                  <CardHeader>
                    <CardTitle className="text-base">
                      Objective {index + 1}
                      {cleanName && (
                        <span className="text-sm font-normal text-muted-foreground ml-2">
                          - {cleanName}
                        </span>
                      )}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                      <div>
                        <div className="text-sm text-muted-foreground">
                          Type
                        </div>
                        <div className="text-lg font-semibold">
                          {objective.type_name
                            ? `${objective.type_name} (${objective.type})`
                            : objective.type}
                        </div>
                      </div>
                      <div>
                        <div className="text-sm text-muted-foreground">
                          Map ID
                        </div>
                        <div className="text-lg font-semibold">
                          {formatClientDataValue(objective.map_id, mapLookup)}
                        </div>
                      </div>
                      <div>
                        <div className="text-sm text-muted-foreground">
                          Location
                        </div>
                        <div className="text-lg font-semibold">
                          X: {formatUInt8Value(objective.location.x)}, Y:{' '}
                          {formatUInt8Value(objective.location.y)}
                        </div>
                      </div>
                      <div>
                        <div className="text-sm text-muted-foreground">
                          Radius
                        </div>
                        <div className="text-lg font-semibold">
                          {formatUInt8Value(objective.radius)}
                        </div>
                      </div>
                      {supportsTarget && (
                        <div>
                          <div className="text-sm text-muted-foreground">
                            {objective.type === OBJECTIVE_TYPE.BRINGNPC
                              ? 'Target NPC'
                              : 'Target Monster'}
                          </div>
                          <div className="text-lg font-semibold">
                            {formatClientDataValue(
                              objective.target_id,
                              monsterLookup,
                            )}
                          </div>
                        </div>
                      )}
                      {supportsKill && (
                        <div>
                          <div className="text-sm text-muted-foreground">
                            Kill Count
                          </div>
                          <div className="text-lg font-semibold">
                            {formatUInt16Value(objective.kill_count)}
                          </div>
                        </div>
                      )}
                      {supportsQuestItem && (
                        <>
                          <div>
                            <div className="text-sm text-muted-foreground">
                              Quest Item ID
                            </div>
                            <div className="text-lg font-semibold">
                              {formatUInt16Value(objective.quest_item_id)}
                            </div>
                          </div>
                          <div>
                            <div className="text-sm text-muted-foreground">
                              Required Items
                            </div>
                            <div className="text-lg font-semibold">
                              {formatUInt16Value(objective.required_item_count)}
                            </div>
                          </div>
                        </>
                      )}
                    </div>

                    {visibleDrops.length > 0 && (
                      <div className="mt-4">
                        <div className="text-sm font-medium mb-2">
                          Drop Items
                        </div>
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>Slot</TableHead>
                              <TableHead className="text-right">
                                Item ID
                              </TableHead>
                              <TableHead className="text-right">
                                Probability
                              </TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {visibleDrops.map((drop) => (
                              <TableRow key={drop.idx}>
                                <TableCell className="font-medium">
                                  Slot {drop.idx + 1}
                                </TableCell>
                                <TableCell className="text-right">
                                  {formatUInt16Value(drop.item)}
                                </TableCell>
                                <TableCell className="text-right">
                                  {formatUInt8Value(drop.probability)}
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                    )}
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Link2 className="h-5 w-5" />
            Continuation Quests
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            {data.continuations.map((continuation, index) => (
              <div
                key={index}
                className="px-3 py-1 bg-muted rounded-md text-sm font-medium"
              >
                {formatContinuation(continuation)}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
