import { useEffect, useRef, useMemo } from 'react';
import { Controller, useFieldArray, useForm, useWatch } from 'react-hook-form';
import { useMutation, useQueryClient, useQuery } from '@tanstack/react-query';
import { useRouter } from '@tanstack/react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Loader2, X, Save, Plus, Trash2 } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Link } from '@tanstack/react-router';
import { toast } from 'sonner';
import {
  APIError,
  updateSpawnFile,
  type SpawnFileAPIData,
  getMonsters,
} from '@/lib/api';
import { queryKeys } from '@/constants';

const spawnSchema = z.object({
  id: z.number().int().min(0).max(65000),
  x: z.number().int().min(0).max(255),
  y: z.number().int().min(0).max(255),
  unknown1: z.number().int().min(0).max(65000),
  orientation: z.number().int().min(0).max(255),
  spwan_step: z.number().int().min(0).max(255),
});

const spawnFileSchema = z.object({
  spawns: z.array(spawnSchema).min(0),
});

type SpawnFileFormData = z.infer<typeof spawnFileSchema>;

interface SpawnFileEditProps {
  filePath: string;
  defaultData: SpawnFileAPIData;
}

export function SpawnFileEdit({ filePath, defaultData }: SpawnFileEditProps) {
  const router = useRouter();
  const queryClient = useQueryClient();

  const form = useForm<SpawnFileFormData>({
    resolver: zodResolver(spawnFileSchema),
    defaultValues: defaultData,
  });

  const hasInitialized = useRef(false);

  useEffect(() => {
    if (!hasInitialized.current && defaultData) {
      form.reset(defaultData);
      hasInitialized.current = true;
    }
  }, [defaultData, form]);

  const { control } = form;

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

  const getMonsterName = (npcId: number): string => {
    const monsterName = monsterMap.get(npcId);
    return monsterName || `${npcId}`;
  };

  const spawnsArray = useFieldArray({
    control,
    name: 'spawns',
  });

  const mutation = useMutation({
    mutationFn: (values: SpawnFileFormData) =>
      updateSpawnFile({ path: filePath }, values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.spawnFile(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.fileTree(filePath),
      });
      toast.success('Spawn file saved');
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
            : 'Failed to save spawn file';
      toast.error(errorMessage);
    },
  });

  const mutationErrorMessage =
    mutation.error instanceof APIError
      ? mutation.error.getErrorMessage()
      : mutation.error instanceof Error
        ? mutation.error.message
        : 'Failed to save spawn file';

  const isSaving = mutation.status === 'pending';

  const addSpawn = () => {
    spawnsArray.append({
      id: 0,
      x: 0,
      y: 0,
      unknown1: 0,
      orientation: 0,
      spwan_step: 0,
    });
  };

  const removeSpawn = (index: number) => {
    spawnsArray.remove(index);
  };

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
          <div className="flex items-center justify-between">
            <CardTitle>Spawn Points</CardTitle>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={addSpawn}
            >
              <Plus className="mr-2 h-4 w-4" />
              Add Spawn
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {spawnsArray.fields.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <p className="mb-4">No spawn points configured</p>
              <Button type="button" variant="outline" onClick={addSpawn}>
                <Plus className="mr-2 h-4 w-4" />
                Add First Spawn Point
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>#</TableHead>
                  <TableHead>Monster Name</TableHead>
                  <TableHead className="text-right">NPC ID</TableHead>
                  <TableHead className="text-right">X</TableHead>
                  <TableHead className="text-right">Y</TableHead>
                  <TableHead className="text-right">Orientation</TableHead>
                  <TableHead className="text-right">Spawn Step</TableHead>
                  <TableHead className="text-right">Unknown1</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {spawnsArray.fields.map((field, index) => {
                  return (
                    <SpawnRow
                      key={field.id}
                      index={index}
                      control={control}
                      getMonsterName={getMonsterName}
                      removeSpawn={removeSpawn}
                    />
                  );
                })}
              </TableBody>
            </Table>
          )}
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
            Save Spawn File
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

interface SpawnRowProps {
  index: number;
  control: ReturnType<typeof useForm<SpawnFileFormData>>['control'];
  getMonsterName: (npcId: number) => string;
  removeSpawn: (index: number) => void;
}

function SpawnRow({
  index,
  control,
  getMonsterName,
  removeSpawn,
}: SpawnRowProps) {
  const npcId = useWatch({
    control,
    name: `spawns.${index}.id`,
  });

  return (
    <TableRow>
      <TableCell className="font-medium">{index + 1}</TableCell>
      <TableCell>{getMonsterName(npcId)}</TableCell>
      <TableCell className="text-right">
        <Controller
          name={`spawns.${index}.id`}
          control={control}
          render={({ field: controllerField }) => (
            <Input
              type="number"
              inputMode="numeric"
              className="text-right w-24"
              min={0}
              max={65000}
              value={
                typeof controllerField.value === 'number'
                  ? controllerField.value.toString()
                  : ''
              }
              onChange={(e) => {
                const value = e.target.value;
                controllerField.onChange(value === '' ? 0 : Number(value));
              }}
              onBlur={controllerField.onBlur}
            />
          )}
        />
      </TableCell>
      <TableCell className="text-right">
        <Controller
          name={`spawns.${index}.x`}
          control={control}
          render={({ field: controllerField }) => (
            <Input
              type="number"
              inputMode="numeric"
              className="text-right w-20"
              min={0}
              max={255}
              value={
                typeof controllerField.value === 'number'
                  ? controllerField.value.toString()
                  : ''
              }
              onChange={(e) => {
                const value = e.target.value;
                controllerField.onChange(value === '' ? 0 : Number(value));
              }}
              onBlur={controllerField.onBlur}
            />
          )}
        />
      </TableCell>
      <TableCell className="text-right">
        <Controller
          name={`spawns.${index}.y`}
          control={control}
          render={({ field: controllerField }) => (
            <Input
              type="number"
              inputMode="numeric"
              className="text-right w-20"
              min={0}
              max={255}
              value={
                typeof controllerField.value === 'number'
                  ? controllerField.value.toString()
                  : ''
              }
              onChange={(e) => {
                const value = e.target.value;
                controllerField.onChange(value === '' ? 0 : Number(value));
              }}
              onBlur={controllerField.onBlur}
            />
          )}
        />
      </TableCell>
      <TableCell className="text-right">
        <Controller
          name={`spawns.${index}.orientation`}
          control={control}
          render={({ field: controllerField }) => (
            <Input
              type="number"
              inputMode="numeric"
              className="text-right w-20"
              min={0}
              max={255}
              value={
                typeof controllerField.value === 'number'
                  ? controllerField.value.toString()
                  : ''
              }
              onChange={(e) => {
                const value = e.target.value;
                controllerField.onChange(value === '' ? 0 : Number(value));
              }}
              onBlur={controllerField.onBlur}
            />
          )}
        />
      </TableCell>
      <TableCell className="text-right">
        <Controller
          name={`spawns.${index}.spwan_step`}
          control={control}
          render={({ field: controllerField }) => (
            <Input
              type="number"
              inputMode="numeric"
              className="text-right w-20"
              min={0}
              max={255}
              value={
                typeof controllerField.value === 'number'
                  ? controllerField.value.toString()
                  : ''
              }
              onChange={(e) => {
                const value = e.target.value;
                controllerField.onChange(value === '' ? 0 : Number(value));
              }}
              onBlur={controllerField.onBlur}
            />
          )}
        />
      </TableCell>
      <TableCell className="text-right">
        <Controller
          name={`spawns.${index}.unknown1`}
          control={control}
          render={({ field: controllerField }) => (
            <Input
              type="number"
              inputMode="numeric"
              className="text-right w-24"
              min={0}
              max={65000}
              value={
                typeof controllerField.value === 'number'
                  ? controllerField.value.toString()
                  : ''
              }
              onChange={(e) => {
                const value = e.target.value;
                controllerField.onChange(value === '' ? 0 : Number(value));
              }}
              onBlur={controllerField.onBlur}
            />
          )}
        />
      </TableCell>
      <TableCell className="text-right">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => removeSpawn(index)}
        >
          <Trash2 className="h-4 w-4 text-destructive" />
        </Button>
      </TableCell>
    </TableRow>
  );
}
