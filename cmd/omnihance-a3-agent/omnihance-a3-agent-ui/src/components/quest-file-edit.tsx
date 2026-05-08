import { useEffect, useMemo } from 'react';
import { Controller, type FieldPath, useForm } from 'react-hook-form';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useRouter } from '@tanstack/react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Loader2, Plus, Save, Trash2, X } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { toast } from 'sonner';
import {
  APIError,
  getItems,
  getMaps,
  getMonsters,
  updateQuestFile,
  type QuestFileAPIData,
} from '@/lib/api';
import { queryKeys } from '@/constants';

const QUEST_OBJECTIVE_TYPE = {
  KILL: 0,
  QUESTITEM: 1,
  BRINGNPC: 2,
  DROP: 3,
  FIND: 4,
  UNUSED: 255,
} as const;

const UNUSED_UINT8 = 255;
const UNUSED_LOCATION_Y = 254;
const UNUSED_UINT16 = 65535;
const MAX_UINT16_FORM = 65535;
const MAX_UINT32_FORM = 4_294_967_295;
const MAX_OBJECTIVE_NAME_LENGTH = 255;

const questObjectiveTypes = [
  { value: QUEST_OBJECTIVE_TYPE.KILL, label: 'KILL' },
  { value: QUEST_OBJECTIVE_TYPE.QUESTITEM, label: 'QUESTITEM' },
  { value: QUEST_OBJECTIVE_TYPE.BRINGNPC, label: 'BRINGNPC' },
  { value: QUEST_OBJECTIVE_TYPE.DROP, label: 'DROP' },
  { value: QUEST_OBJECTIVE_TYPE.FIND, label: 'FIND' },
  { value: QUEST_OBJECTIVE_TYPE.UNUSED, label: 'UNUSED' },
];

const namedObjectiveTypes = [
  QUEST_OBJECTIVE_TYPE.DROP,
  QUEST_OBJECTIVE_TYPE.FIND,
] as number[];

const objectiveSchema = z
  .object({
    type: z.number().int().min(0).max(255),
    type_name: z.string().optional(),
    map_id: z.number().int().min(0).max(MAX_UINT16_FORM),
    location: z.object({
      x: z.number().int().min(0).max(UNUSED_UINT8),
      y: z.number().int().min(0).max(UNUSED_UINT8),
    }),
    radius: z.number().int().min(0).max(UNUSED_UINT8),
    target_id: z.number().int().min(0).max(MAX_UINT16_FORM),
    kill_count: z.number().int().min(0).max(MAX_UINT16_FORM),
    quest_item_id: z.number().int().min(0).max(MAX_UINT16_FORM),
    drop_items: z
      .array(z.number().int().min(0).max(MAX_UINT16_FORM).nullable())
      .length(3),
    required_item_count: z.number().int().min(0).max(MAX_UINT16_FORM),
    drop_probs: z.array(z.number().int().min(0).max(UNUSED_UINT8)).length(3),
    name: z.string().max(MAX_OBJECTIVE_NAME_LENGTH),
    is_unused: z.boolean().optional(),
  })
  .superRefine((objective, ctx) => {
    const validType = questObjectiveTypes.some(
      (type) => type.value === objective.type,
    );
    if (!validType) {
      ctx.addIssue({
        code: 'custom',
        path: ['type'],
        message: 'Select a valid objective type',
      });
    }

    if (objective.is_unused && objective.type !== QUEST_OBJECTIVE_TYPE.UNUSED) {
      ctx.addIssue({
        code: 'custom',
        path: ['type'],
        message: 'Unused objectives must use UNUSED type',
      });
    }

    if (!namedObjectiveTypes.includes(objective.type) && objective.name) {
      ctx.addIssue({
        code: 'custom',
        path: ['name'],
        message: 'Only DROP and FIND objectives can have names',
      });
    }
  });

const questFileSchema = z
  .object({
    quest_id: z.number().int().min(0).max(MAX_UINT16_FORM),
    giver_npc: z.number().int().min(0).max(MAX_UINT16_FORM),
    target_npc: z.number().int().min(0).max(MAX_UINT16_FORM),
    min_level: z.number().int().min(0).max(UNUSED_UINT8),
    max_level: z.number().int().min(0).max(UNUSED_UINT8),
    flags: z.number().int().min(0).max(MAX_UINT32_FORM),
    reward_items: z
      .array(z.number().int().min(0).max(MAX_UINT16_FORM).nullable())
      .length(4),
    reward_counts: z
      .array(z.number().int().min(0).max(UNUSED_UINT8).nullable())
      .length(4),
    exp_reward: z.number().int().min(0).max(MAX_UINT32_FORM),
    woonz_reward: z.number().int().min(0).max(MAX_UINT32_FORM),
    lore_reward: z.number().int().min(0).max(MAX_UINT32_FORM),
    objectives: z.array(objectiveSchema).length(7),
    continuations: z
      .array(z.number().int().min(0).max(MAX_UINT32_FORM).nullable())
      .length(3),
  })
  .superRefine((quest, ctx) => {
    if (
      quest.min_level !== UNUSED_UINT8 &&
      quest.max_level !== UNUSED_UINT8 &&
      quest.min_level > quest.max_level
    ) {
      ctx.addIssue({
        code: 'custom',
        path: ['min_level'],
        message: 'Min level must be less than or equal to max level',
      });
    }
  });

type QuestFileFormData = z.infer<typeof questFileSchema>;

interface QuestFileEditProps {
  filePath: string;
  defaultData: QuestFileAPIData;
}

export function QuestFileEdit({ filePath, defaultData }: QuestFileEditProps) {
  const router = useRouter();
  const queryClient = useQueryClient();

  const form = useForm<QuestFileFormData>({
    resolver: zodResolver(questFileSchema),
    defaultValues: toQuestFormData(defaultData),
  });

  useEffect(() => {
    if (defaultData) {
      form.reset(toQuestFormData(defaultData));
    }
  }, [defaultData, form]);

  const { control, setValue, watch } = form;
  const objectives = watch('objectives');
  const rewardItems = watch('reward_items');
  const rewardCounts = watch('reward_counts');
  const continuations = watch('continuations');
  const giverNPC = watch('giver_npc');
  const targetNPC = watch('target_npc');

  const { data: maps } = useQuery({
    queryKey: queryKeys.maps,
    queryFn: () => getMaps(),
  });

  const { data: monsters } = useQuery({
    queryKey: queryKeys.monsters,
    queryFn: () => getMonsters(),
  });

  const { data: items } = useQuery({
    queryKey: queryKeys.items,
    queryFn: () => getItems(),
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

  const itemLookup = useMemo(() => {
    if (!items) {
      return new Map<number, string>();
    }

    const map = new Map<number, string>();
    for (const item of items) {
      map.set(item.id, item.name);
    }

    return map;
  }, [items]);

  const mutation = useMutation({
    mutationFn: (values: QuestFileFormData) =>
      updateQuestFile({ path: filePath }, values as QuestFileAPIData),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.questFile(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.fileTree(filePath),
      });
      toast.success('Quest file saved');
      router.navigate({
        to: '/file/view',
        search: { path: filePath },
      });
    },
    onError: (error) => {
      const errorMessage =
        error instanceof APIError
          ? error.getErrorMessage()
          : error instanceof Error
            ? error.message
            : 'Failed to save quest file';
      toast.error(errorMessage);
    },
  });

  const mutationErrorMessage =
    mutation.error instanceof APIError
      ? mutation.error.getErrorMessage()
      : mutation.error instanceof Error
        ? mutation.error.message
        : 'Failed to save quest file';

  const isSaving = mutation.status === 'pending';

  const setObjectiveType = (index: number, nextType: number) => {
    const base = `objectives.${index}` as const;
    setValue(`${base}.type`, nextType, { shouldDirty: true });
    setValue(`${base}.is_unused`, nextType === QUEST_OBJECTIVE_TYPE.UNUSED, {
      shouldDirty: true,
    });

    if (nextType === QUEST_OBJECTIVE_TYPE.UNUSED) {
      setUnusedObjectiveValues(index);
      return;
    }

    if (objectives[index]?.type === QUEST_OBJECTIVE_TYPE.UNUSED) {
      setActiveObjectiveDefaults(index);
    }

    if (!namedObjectiveTypes.includes(nextType)) {
      setValue(`${base}.name`, '', { shouldDirty: true });
    }

    if (nextType !== QUEST_OBJECTIVE_TYPE.KILL) {
      setValue(`${base}.kill_count`, UNUSED_UINT16, { shouldDirty: true });
    }

    if (
      nextType !== QUEST_OBJECTIVE_TYPE.QUESTITEM &&
      nextType !== QUEST_OBJECTIVE_TYPE.BRINGNPC
    ) {
      setValue(`${base}.quest_item_id`, UNUSED_UINT16, { shouldDirty: true });
      setValue(`${base}.required_item_count`, UNUSED_UINT16, {
        shouldDirty: true,
      });
    }

    if (
      nextType !== QUEST_OBJECTIVE_TYPE.KILL &&
      nextType !== QUEST_OBJECTIVE_TYPE.DROP
    ) {
      setDropSentinels(index);
    }
  };

  const setUnusedObjectiveValues = (index: number) => {
    const base = `objectives.${index}` as const;
    setValue(`${base}.map_id`, UNUSED_UINT16, { shouldDirty: true });
    setValue(`${base}.location.x`, UNUSED_UINT8, { shouldDirty: true });
    setValue(`${base}.location.y`, UNUSED_LOCATION_Y, { shouldDirty: true });
    setValue(`${base}.radius`, UNUSED_UINT8, { shouldDirty: true });
    setValue(`${base}.target_id`, UNUSED_UINT16, { shouldDirty: true });
    setValue(`${base}.kill_count`, UNUSED_UINT16, { shouldDirty: true });
    setValue(`${base}.quest_item_id`, UNUSED_UINT16, { shouldDirty: true });
    setValue(`${base}.required_item_count`, UNUSED_UINT16, {
      shouldDirty: true,
    });
    setValue(`${base}.name`, '', { shouldDirty: true });
    setDropSentinels(index);
  };

  const setActiveObjectiveDefaults = (index: number) => {
    const base = `objectives.${index}` as const;
    setValue(`${base}.map_id`, 0, { shouldDirty: true });
    setValue(`${base}.location.x`, 0, { shouldDirty: true });
    setValue(`${base}.location.y`, 0, { shouldDirty: true });
    setValue(`${base}.radius`, 0, { shouldDirty: true });
    setValue(`${base}.target_id`, UNUSED_UINT16, { shouldDirty: true });
    setValue(`${base}.kill_count`, UNUSED_UINT16, { shouldDirty: true });
    setValue(`${base}.quest_item_id`, UNUSED_UINT16, { shouldDirty: true });
    setValue(`${base}.required_item_count`, UNUSED_UINT16, {
      shouldDirty: true,
    });
    setDropSentinels(index);
  };

  const setDropSentinels = (index: number) => {
    for (let dropIndex = 0; dropIndex < 3; dropIndex++) {
      setValue(`objectives.${index}.drop_items.${dropIndex}`, UNUSED_UINT16, {
        shouldDirty: true,
      });
      setValue(`objectives.${index}.drop_probs.${dropIndex}`, UNUSED_UINT8, {
        shouldDirty: true,
      });
    }
  };

  const addRewardSlot = (index: number) => {
    setValue(`reward_items.${index}`, 0, { shouldDirty: true });
    setValue(`reward_counts.${index}`, 1, { shouldDirty: true });
  };

  const removeRewardSlot = (index: number) => {
    setValue(`reward_items.${index}`, null, { shouldDirty: true });
    setValue(`reward_counts.${index}`, null, { shouldDirty: true });
  };

  const addDropSlot = (objectiveIndex: number, dropIndex: number) => {
    setValue(`objectives.${objectiveIndex}.drop_items.${dropIndex}`, 0, {
      shouldDirty: true,
    });
    setValue(`objectives.${objectiveIndex}.drop_probs.${dropIndex}`, 1, {
      shouldDirty: true,
    });
  };

  const removeDropSlot = (objectiveIndex: number, dropIndex: number) => {
    setValue(
      `objectives.${objectiveIndex}.drop_items.${dropIndex}`,
      UNUSED_UINT16,
      {
        shouldDirty: true,
      },
    );
    setValue(
      `objectives.${objectiveIndex}.drop_probs.${dropIndex}`,
      UNUSED_UINT8,
      {
        shouldDirty: true,
      },
    );
  };

  const addQuestItemRequirement = (objectiveIndex: number) => {
    setValue(`objectives.${objectiveIndex}.quest_item_id`, 0, {
      shouldDirty: true,
    });
    setValue(`objectives.${objectiveIndex}.required_item_count`, 1, {
      shouldDirty: true,
    });
  };

  const removeQuestItemRequirement = (objectiveIndex: number) => {
    setValue(`objectives.${objectiveIndex}.quest_item_id`, UNUSED_UINT16, {
      shouldDirty: true,
    });
    setValue(
      `objectives.${objectiveIndex}.required_item_count`,
      UNUSED_UINT16,
      {
        shouldDirty: true,
      },
    );
  };

  const addContinuation = (index: number) => {
    setValue(`continuations.${index}`, 0, { shouldDirty: true });
  };

  const removeContinuation = (index: number) => {
    setValue(`continuations.${index}`, null, { shouldDirty: true });
  };

  const isRewardSlotActive = (index: number) => {
    const item = rewardItems[index];
    const count = rewardCounts[index];
    return item !== null && item !== UNUSED_UINT16 && count !== null;
  };

  const isDropSlotActive = (objectiveIndex: number, dropIndex: number) => {
    const item = objectives[objectiveIndex]?.drop_items[dropIndex];
    const probability = objectives[objectiveIndex]?.drop_probs[dropIndex];
    return (
      item != null &&
      item !== UNUSED_UINT16 &&
      probability != null &&
      probability !== UNUSED_UINT8
    );
  };

  const isQuestItemRequirementActive = (objectiveIndex: number) => {
    const questItemID = objectives[objectiveIndex]?.quest_item_id;
    const requiredItemCount = objectives[objectiveIndex]?.required_item_count;
    return (
      questItemID != null &&
      requiredItemCount != null &&
      questItemID !== UNUSED_UINT16 &&
      requiredItemCount !== UNUSED_UINT16
    );
  };

  const isContinuationActive = (index: number) => {
    return continuations[index] !== null;
  };

  const numberField = (
    id: FieldPath<QuestFileFormData>,
    label: string,
    options?: {
      min?: number;
      max?: number;
      disabled?: boolean;
      displayValue?: string;
    },
  ) => (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Controller
        name={id}
        control={control}
        render={({ field }) => (
          <Input
            id={id}
            type="number"
            inputMode="numeric"
            min={options?.min}
            max={options?.max}
            disabled={options?.disabled}
            value={
              typeof field.value === 'number' ? field.value.toString() : ''
            }
            onChange={(e) => {
              const value = e.target.value;
              field.onChange(value === '' ? 0 : Number(value));
            }}
            onBlur={field.onBlur}
          />
        )}
      />
      {options?.displayValue && (
        <p className="text-sm text-muted-foreground">{options.displayValue}</p>
      )}
    </div>
  );

  const nullableNumberField = (
    id: FieldPath<QuestFileFormData>,
    label: string,
    options?: {
      min?: number;
      max?: number;
      disabled?: boolean;
      emptyValue?: number | null;
      displayValue?: string;
    },
  ) => (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Controller
        name={id}
        control={control}
        render={({ field }) => (
          <Input
            id={id}
            type="number"
            inputMode="numeric"
            min={options?.min}
            max={options?.max}
            disabled={options?.disabled}
            value={
              field.value !== null && field.value !== undefined
                ? String(field.value)
                : ''
            }
            onChange={(e) => {
              const value = e.target.value;
              field.onChange(
                value === '' ? (options?.emptyValue ?? null) : Number(value),
              );
            }}
            onBlur={field.onBlur}
            placeholder="-"
          />
        )}
      />
      {options?.displayValue && (
        <p className="text-sm text-muted-foreground">{options.displayValue}</p>
      )}
    </div>
  );

  return (
    <form
      onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
      className="space-y-6"
    >
      {mutation.isError && (
        <Alert variant="destructive">
          <AlertDescription>{mutationErrorMessage}</AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Quest Header</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {numberField('quest_id', 'Quest ID', {
              min: 0,
              max: MAX_UINT16_FORM,
            })}
            {numberField('giver_npc', 'Giver NPC ID', {
              min: 0,
              max: MAX_UINT16_FORM,
              displayValue: formatClientDataName(giverNPC, monsterLookup),
            })}
            {numberField('target_npc', 'Target NPC ID', {
              min: 0,
              max: MAX_UINT16_FORM,
              displayValue: formatClientDataName(targetNPC, monsterLookup),
            })}
            {numberField('min_level', 'Min Level', {
              min: 0,
              max: UNUSED_UINT8,
            })}
            {numberField('max_level', 'Max Level', {
              min: 0,
              max: UNUSED_UINT8,
            })}
            {numberField('flags', 'Flags', {
              min: 0,
              max: MAX_UINT32_FORM,
            })}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Rewards</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {numberField('exp_reward', 'Experience Reward', {
              min: 0,
              max: MAX_UINT32_FORM,
            })}
            {numberField('woonz_reward', 'Woonz Reward', {
              min: 0,
              max: MAX_UINT32_FORM,
            })}
            {numberField('lore_reward', 'Lore Reward', {
              min: 0,
              max: MAX_UINT32_FORM,
            })}
          </div>

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Reward Slot</TableHead>
                <TableHead>Item ID</TableHead>
                <TableHead>Count</TableHead>
                <TableHead className="w-24 text-right">Action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {[0, 1, 2].map((index) => {
                const isActive = isRewardSlotActive(index);

                return (
                  <TableRow key={index}>
                    <TableCell className="font-medium">
                      Slot {index + 1}
                    </TableCell>
                    <TableCell>
                      {nullableNumberField(`reward_items.${index}`, 'Item ID', {
                        min: 0,
                        max: MAX_UINT16_FORM,
                        disabled: !isActive,
                        emptyValue: 0,
                        displayValue: isActive
                          ? formatClientDataName(rewardItems[index], itemLookup)
                          : '-',
                      })}
                    </TableCell>
                    <TableCell>
                      {nullableNumberField(`reward_counts.${index}`, 'Count', {
                        min: 0,
                        max: UNUSED_UINT8,
                        disabled: !isActive,
                        emptyValue: 0,
                      })}
                    </TableCell>
                    <TableCell className="text-right">
                      {isActive ? (
                        <Button
                          type="button"
                          variant="outline"
                          size="icon"
                          aria-label={`Remove reward slot ${index + 1}`}
                          onClick={() => removeRewardSlot(index)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      ) : (
                        <Button
                          type="button"
                          variant="outline"
                          size="icon"
                          aria-label={`Add reward slot ${index + 1}`}
                          onClick={() => addRewardSlot(index)}
                        >
                          <Plus className="h-4 w-4" />
                        </Button>
                      )}
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
          <CardTitle>Objectives</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {objectives.map((objective, index) => {
              const objectiveType = objective.type;
              const isUnused = objectiveType === QUEST_OBJECTIVE_TYPE.UNUSED;
              const supportsTarget =
                objectiveType === QUEST_OBJECTIVE_TYPE.KILL ||
                objectiveType === QUEST_OBJECTIVE_TYPE.BRINGNPC;
              const supportsKill = objectiveType === QUEST_OBJECTIVE_TYPE.KILL;
              const supportsQuestItem =
                objectiveType === QUEST_OBJECTIVE_TYPE.QUESTITEM ||
                objectiveType === QUEST_OBJECTIVE_TYPE.BRINGNPC;
              const hasQuestItemRequirement =
                isQuestItemRequirementActive(index);
              const supportsDrops =
                objectiveType === QUEST_OBJECTIVE_TYPE.KILL ||
                objectiveType === QUEST_OBJECTIVE_TYPE.DROP;
              const supportsName = namedObjectiveTypes.includes(objectiveType);

              return (
                <Card key={index} className="border-l-4">
                  <CardHeader>
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                      <CardTitle className="text-base">
                        Objective {index + 1}
                      </CardTitle>
                      <div className="flex gap-2">
                        <Controller
                          name={`objectives.${index}.type`}
                          control={control}
                          render={({ field }) => (
                            <Select
                              value={String(field.value)}
                              onValueChange={(value) => {
                                setObjectiveType(index, Number(value));
                              }}
                            >
                              <SelectTrigger
                                className="w-full sm:w-48"
                                aria-label={`Objective ${index + 1} type`}
                              >
                                <SelectValue placeholder="Select type" />
                              </SelectTrigger>
                              <SelectContent>
                                {questObjectiveTypes.map((type) => (
                                  <SelectItem
                                    key={type.value}
                                    value={String(type.value)}
                                  >
                                    {type.label}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          )}
                        />
                        {isUnused ? (
                          <Button
                            type="button"
                            variant="outline"
                            size="icon"
                            aria-label={`Add objective ${index + 1}`}
                            onClick={() =>
                              setObjectiveType(index, QUEST_OBJECTIVE_TYPE.KILL)
                            }
                          >
                            <Plus className="h-4 w-4" />
                          </Button>
                        ) : (
                          <Button
                            type="button"
                            variant="outline"
                            size="icon"
                            aria-label={`Remove objective ${index + 1}`}
                            onClick={() =>
                              setObjectiveType(
                                index,
                                QUEST_OBJECTIVE_TYPE.UNUSED,
                              )
                            }
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                      {numberField(`objectives.${index}.map_id`, 'Map ID', {
                        min: 0,
                        max: MAX_UINT16_FORM,
                        disabled: isUnused,
                        displayValue: formatClientDataName(
                          objective.map_id,
                          mapLookup,
                        ),
                      })}
                      {numberField(
                        `objectives.${index}.location.x`,
                        'Location X',
                        {
                          min: 0,
                          max: UNUSED_UINT8,
                          disabled: isUnused,
                        },
                      )}
                      {numberField(
                        `objectives.${index}.location.y`,
                        'Location Y',
                        {
                          min: 0,
                          max: UNUSED_UINT8,
                          disabled: isUnused,
                        },
                      )}
                      {numberField(`objectives.${index}.radius`, 'Radius', {
                        min: 0,
                        max: UNUSED_UINT8,
                        disabled: isUnused,
                      })}
                      {numberField(
                        `objectives.${index}.target_id`,
                        objectiveType === QUEST_OBJECTIVE_TYPE.BRINGNPC
                          ? 'Target NPC ID'
                          : 'Target Monster ID',
                        {
                          min: 0,
                          max: MAX_UINT16_FORM,
                          disabled: !supportsTarget,
                          displayValue: supportsTarget
                            ? formatClientDataName(
                                objective.target_id,
                                monsterLookup,
                              )
                            : '-',
                        },
                      )}
                      {numberField(
                        `objectives.${index}.kill_count`,
                        'Kill Count',
                        {
                          min: 0,
                          max: MAX_UINT16_FORM,
                          disabled: !supportsKill,
                        },
                      )}
                      {numberField(
                        `objectives.${index}.quest_item_id`,
                        'Quest Item ID',
                        {
                          min: 0,
                          max: MAX_UINT16_FORM,
                          disabled:
                            !supportsQuestItem || !hasQuestItemRequirement,
                          displayValue:
                            supportsQuestItem && hasQuestItemRequirement
                              ? formatClientDataName(
                                  objective.quest_item_id,
                                  itemLookup,
                                )
                              : '-',
                        },
                      )}
                      {numberField(
                        `objectives.${index}.required_item_count`,
                        'Required Item Count',
                        {
                          min: 0,
                          max: MAX_UINT16_FORM,
                          disabled:
                            !supportsQuestItem || !hasQuestItemRequirement,
                        },
                      )}
                    </div>

                    {supportsQuestItem && (
                      <div className="flex justify-end">
                        {hasQuestItemRequirement ? (
                          <Button
                            type="button"
                            variant="outline"
                            aria-label={`Remove quest item requirement for objective ${index + 1}`}
                            onClick={() => removeQuestItemRequirement(index)}
                          >
                            <span className="flex items-center gap-1.5">
                              <Trash2 className="h-4 w-4" />
                              Remove Quest Item
                            </span>
                          </Button>
                        ) : (
                          <Button
                            type="button"
                            variant="outline"
                            aria-label={`Add quest item requirement for objective ${index + 1}`}
                            onClick={() => addQuestItemRequirement(index)}
                          >
                            <span className="flex items-center gap-1.5">
                              <Plus className="h-4 w-4" />
                              Add Quest Item
                            </span>
                          </Button>
                        )}
                      </div>
                    )}

                    <div className="space-y-2">
                      <Label htmlFor={`objectives.${index}.name`}>
                        Objective Name
                      </Label>
                      <Controller
                        name={`objectives.${index}.name`}
                        control={control}
                        render={({ field }) => (
                          <Input
                            {...field}
                            id={`objectives.${index}.name`}
                            disabled={!supportsName}
                            maxLength={MAX_OBJECTIVE_NAME_LENGTH}
                            placeholder={
                              supportsName ? 'Name payload' : 'Unavailable'
                            }
                          />
                        )}
                      />
                    </div>

                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Drop Slot</TableHead>
                          <TableHead>Item ID</TableHead>
                          <TableHead>Probability</TableHead>
                          <TableHead className="w-24 text-right">
                            Action
                          </TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {[0, 1, 2].map((dropIndex) => {
                          const hasDropSlot = isDropSlotActive(
                            index,
                            dropIndex,
                          );

                          return (
                            <TableRow key={dropIndex}>
                              <TableCell className="font-medium">
                                Slot {dropIndex + 1}
                              </TableCell>
                              <TableCell>
                                {nullableNumberField(
                                  `objectives.${index}.drop_items.${dropIndex}`,
                                  'Drop Item ID',
                                  {
                                    min: 0,
                                    max: MAX_UINT16_FORM,
                                    disabled: !supportsDrops || !hasDropSlot,
                                    emptyValue: 0,
                                    displayValue:
                                      supportsDrops && hasDropSlot
                                        ? formatClientDataName(
                                            objective.drop_items[dropIndex],
                                            itemLookup,
                                          )
                                        : '-',
                                  },
                                )}
                              </TableCell>
                              <TableCell>
                                {numberField(
                                  `objectives.${index}.drop_probs.${dropIndex}`,
                                  'Drop Probability',
                                  {
                                    min: 0,
                                    max: UNUSED_UINT8,
                                    disabled: !supportsDrops || !hasDropSlot,
                                  },
                                )}
                              </TableCell>
                              <TableCell className="text-right">
                                {hasDropSlot ? (
                                  <Button
                                    type="button"
                                    variant="outline"
                                    size="icon"
                                    disabled={!supportsDrops}
                                    aria-label={`Remove drop slot ${dropIndex + 1} from objective ${index + 1}`}
                                    onClick={() =>
                                      removeDropSlot(index, dropIndex)
                                    }
                                  >
                                    <Trash2 className="h-4 w-4" />
                                  </Button>
                                ) : (
                                  <Button
                                    type="button"
                                    variant="outline"
                                    size="icon"
                                    disabled={!supportsDrops}
                                    aria-label={`Add drop slot ${dropIndex + 1} to objective ${index + 1}`}
                                    onClick={() =>
                                      addDropSlot(index, dropIndex)
                                    }
                                  >
                                    <Plus className="h-4 w-4" />
                                  </Button>
                                )}
                              </TableCell>
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Continuation Quests</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-3">
            {[0, 1, 2].map((index) => {
              const isActive = isContinuationActive(index);

              return (
                <div key={index} className="flex items-end gap-2">
                  <div className="min-w-0 flex-1">
                    {nullableNumberField(
                      `continuations.${index}`,
                      `Continuation Quest ${index + 1} ID`,
                      {
                        min: 0,
                        max: MAX_UINT32_FORM,
                        disabled: !isActive,
                        emptyValue: 0,
                      },
                    )}
                  </div>
                  {isActive ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      aria-label={`Remove continuation quest ${index + 1}`}
                      onClick={() => removeContinuation(index)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  ) : (
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      aria-label={`Add continuation quest ${index + 1}`}
                      onClick={() => addContinuation(index)}
                    >
                      <Plus className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>

      <div className="flex flex-wrap items-center gap-3">
        <Button type="submit" disabled={isSaving}>
          <span className="flex items-center gap-1.5">
            {isSaving ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            Save Quest File
          </span>
        </Button>
        <Button variant="outline" asChild>
          <Link to="/file/view" search={{ path: filePath }}>
            <span className="flex items-center gap-1.5">
              <X className="h-4 w-4" />
              Cancel
            </span>
          </Link>
        </Button>
      </div>
    </form>
  );
}

function formatClientDataName(
  value: number | null | undefined,
  lookupMap: Map<number, string>,
): string {
  if (value == null || value === UNUSED_UINT16) {
    return '-';
  }

  return lookupMap.get(value) || String(value);
}

function toQuestFormData(data: QuestFileAPIData): QuestFileFormData {
  return {
    quest_id: data.quest_id,
    giver_npc: data.giver_npc,
    target_npc: data.target_npc,
    min_level: data.min_level,
    max_level: data.max_level,
    flags: data.flags,
    reward_items: data.reward_items.map((item) => item ?? null),
    reward_counts: data.reward_counts.map((count) => count ?? null),
    exp_reward: data.exp_reward,
    woonz_reward: data.woonz_reward,
    lore_reward: data.lore_reward,
    objectives: data.objectives.map((objective) => ({
      ...objective,
      is_unused:
        objective.is_unused || objective.type === QUEST_OBJECTIVE_TYPE.UNUSED,
      drop_items: objective.drop_items.map((item) => item ?? null),
      drop_probs: objective.drop_probs.map((probability) =>
        probability == null ? UNUSED_UINT8 : probability,
      ),
      name: objective.name ?? '',
    })),
    continuations: data.continuations.map((continuation) =>
      continuation == null ? null : continuation,
    ),
  };
}
