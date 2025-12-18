import { useEffect, useRef } from 'react';
import { Controller, useFieldArray, useForm } from 'react-hook-form';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from '@tanstack/react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Loader2, X, Save } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
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
import { APIError, updateNPCFile, type NPCFileAPIData } from '@/lib/api';
import { queryKeys } from '@/constants';

const attackSchema = z.object({
  range: z.number().int().min(0).max(65000),
  area: z.number().int().min(0).max(65000),
  damage: z.number().int().min(0).max(65000),
  additional_damage: z.number().int().min(0).max(65000),
});

const npcFileSchema = z.object({
  name: z.string().min(1).max(19),
  id: z.number().int().min(1).max(65000),
  level: z.number().int().min(0).max(200),
  hp: z.number().int().min(0).max(4_000_000_000),
  respawn_rate: z.number().int().min(0).max(65000),
  defense: z.number().int().min(0).max(254),
  additional_defense: z.number().int().min(0).max(254),
  appearance: z.number().int().min(0).max(254),
  attack_speed_low: z.number().int().min(0).max(65000),
  attack_speed_high: z.number().int().min(0).max(65000),
  movement_speed: z.number().int().min(0).max(4_000_000_000),
  attack_type_info: z.number().int().min(0).max(254),
  target_selection_info: z.number().int().min(0).max(254),
  player_exp: z.number().int().min(0).max(65000),
  mercenary_exp: z.number().int().min(0).max(65000),
  blue_attack_defense: z.number().int().min(0).max(65000),
  red_attack_defense: z.number().int().min(0).max(65000),
  grey_attack_defense: z.number().int().min(0).max(65000),
  attacks: z.array(attackSchema),
});

type NPCFileFormData = z.infer<typeof npcFileSchema>;

interface NPCFileEditProps {
  filePath: string;
  defaultData: NPCFileAPIData;
}

export function NPCFileEdit({ filePath, defaultData }: NPCFileEditProps) {
  const router = useRouter();
  const queryClient = useQueryClient();

  const form = useForm<NPCFileFormData>({
    resolver: zodResolver(npcFileSchema),
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

  const attacksArray = useFieldArray({
    control,
    name: 'attacks',
  });

  const mutation = useMutation({
    mutationFn: (values: NPCFileFormData) =>
      updateNPCFile({ path: filePath }, values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.npcFile(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.fileTree(filePath),
      });
      toast.success('NPC file saved');
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
            : 'Failed to save NPC file';
      toast.error(errorMessage);
    },
  });

  const mutationErrorMessage =
    mutation.error instanceof APIError
      ? mutation.error.getErrorMessage()
      : mutation.error instanceof Error
        ? mutation.error.message
        : 'Failed to save NPC file';

  const isSaving = mutation.status === 'pending';

  const numberField = (
    id: string,
    label: string,
    helper?: string,
    options?: { min?: number; max?: number },
  ) => (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Controller
        name={id as keyof NPCFileFormData}
        control={control}
        render={({ field }) => (
          <Input
            id={id}
            type="number"
            inputMode="numeric"
            min={options?.min}
            max={options?.max}
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
      {form.formState.errors[id as keyof NPCFileFormData] && (
        <p className="text-sm text-destructive">
          {
            form.formState.errors[id as keyof NPCFileFormData]
              ?.message as string
          }
        </p>
      )}
      {helper && <p className="text-xs text-muted-foreground">{helper}</p>}
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
          <CardTitle>Basic Stats</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                maxLength={19}
                minLength={1}
                {...form.register('name')}
              />
              {form.formState.errors.name && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.name?.message}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="id">ID</Label>
              <Controller
                name="id"
                control={control}
                render={({ field }) => (
                  <Input
                    id="id"
                    type="number"
                    inputMode="numeric"
                    min={1}
                    max={65000}
                    value={
                      typeof field.value === 'number'
                        ? field.value.toString()
                        : ''
                    }
                    onChange={(e) => {
                      const value = e.target.value;
                      field.onChange(value === '' ? 0 : Number(value));
                    }}
                    onBlur={field.onBlur}
                  />
                )}
              />
              {form.formState.errors.id && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.id?.message}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="level">Level</Label>
              <Controller
                name="level"
                control={control}
                render={({ field }) => (
                  <Input
                    id="level"
                    type="number"
                    inputMode="numeric"
                    min={0}
                    max={200}
                    value={
                      typeof field.value === 'number'
                        ? field.value.toString()
                        : ''
                    }
                    onChange={(e) => {
                      const value = e.target.value;
                      field.onChange(value === '' ? 0 : Number(value));
                    }}
                    onBlur={field.onBlur}
                  />
                )}
              />
              {form.formState.errors.level && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.level?.message}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="hp">HP</Label>
              <Controller
                name="hp"
                control={control}
                render={({ field }) => (
                  <Input
                    id="hp"
                    type="number"
                    inputMode="numeric"
                    min={0}
                    max={4000000000}
                    value={
                      typeof field.value === 'number'
                        ? field.value.toString()
                        : ''
                    }
                    onChange={(e) => {
                      const value = e.target.value;
                      field.onChange(value === '' ? 0 : Number(value));
                    }}
                    onBlur={field.onBlur}
                  />
                )}
              />
              {form.formState.errors.hp && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.hp?.message}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="respawn_rate">Respawn Rate</Label>
              <Controller
                name="respawn_rate"
                control={control}
                render={({ field }) => (
                  <Input
                    id="respawn_rate"
                    type="number"
                    inputMode="numeric"
                    min={0}
                    max={65000}
                    value={
                      typeof field.value === 'number'
                        ? field.value.toString()
                        : ''
                    }
                    onChange={(e) => {
                      const value = e.target.value;
                      field.onChange(value === '' ? 0 : Number(value));
                    }}
                    onBlur={field.onBlur}
                  />
                )}
              />
              {form.formState.errors.respawn_rate && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.respawn_rate?.message}
                </p>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Defense Stats</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {numberField('defense', 'Defense', undefined, { min: 0, max: 254 })}
            {numberField(
              'additional_defense',
              'Additional Defense',
              undefined,
              {
                min: 0,
                max: 254,
              },
            )}
            {numberField('appearance', 'Appearance', undefined, {
              min: 0,
              max: 254,
            })}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Speed & Targeting</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {numberField('attack_speed_low', 'Attack Speed Low', undefined, {
              min: 0,
              max: 65000,
            })}
            {numberField('attack_speed_high', 'Attack Speed High', undefined, {
              min: 0,
              max: 65000,
            })}
            {numberField('movement_speed', 'Movement Speed', undefined, {
              min: 0,
              max: 4000000000,
            })}
            {numberField('attack_type_info', 'Attack Type Info', undefined, {
              min: 0,
              max: 254,
            })}
            {numberField(
              'target_selection_info',
              'Target Selection Info',
              undefined,
              {
                min: 0,
                max: 254,
              },
            )}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Experience</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2">
            {numberField('player_exp', 'Player EXP', undefined, {
              min: 0,
              max: 65000,
            })}
            {numberField('mercenary_exp', 'Mercenary EXP', undefined, {
              min: 0,
              max: 65000,
            })}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Elemental Defense</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {numberField(
              'blue_attack_defense',
              'Blue Attack Defense',
              undefined,
              {
                min: 0,
                max: 65000,
              },
            )}
            {numberField(
              'red_attack_defense',
              'Red Attack Defense',
              undefined,
              {
                min: 0,
                max: 65000,
              },
            )}
            {numberField(
              'grey_attack_defense',
              'Grey Attack Defense',
              undefined,
              {
                min: 0,
                max: 65000,
              },
            )}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Attacks</CardTitle>
        </CardHeader>
        <CardContent>
          {attacksArray.fields.length === 0 ? (
            <div className="text-center text-muted-foreground">
              No attacks available
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Attack</TableHead>
                  <TableHead className="text-right">Range</TableHead>
                  <TableHead className="text-right">Area</TableHead>
                  <TableHead className="text-right">Damage</TableHead>
                  <TableHead className="text-right">
                    Additional Damage
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {attacksArray.fields.map((field, index) => (
                  <TableRow key={field.id}>
                    <TableCell className="font-medium">{index + 1}</TableCell>
                    <TableCell className="text-right">
                      <Controller
                        name={`attacks.${index}.range`}
                        control={control}
                        render={({ field: controllerField }) => (
                          <Input
                            type="number"
                            inputMode="numeric"
                            className="text-right"
                            min={0}
                            max={65000}
                            value={
                              typeof controllerField.value === 'number'
                                ? controllerField.value.toString()
                                : ''
                            }
                            onChange={(e) => {
                              const value = e.target.value;
                              controllerField.onChange(
                                value === '' ? 0 : Number(value),
                              );
                            }}
                            onBlur={controllerField.onBlur}
                          />
                        )}
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <Controller
                        name={`attacks.${index}.area`}
                        control={control}
                        render={({ field: controllerField }) => (
                          <Input
                            type="number"
                            inputMode="numeric"
                            className="text-right"
                            min={0}
                            max={65000}
                            value={
                              typeof controllerField.value === 'number'
                                ? controllerField.value.toString()
                                : ''
                            }
                            onChange={(e) => {
                              const value = e.target.value;
                              controllerField.onChange(
                                value === '' ? 0 : Number(value),
                              );
                            }}
                            onBlur={controllerField.onBlur}
                          />
                        )}
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <Controller
                        name={`attacks.${index}.damage`}
                        control={control}
                        render={({ field: controllerField }) => (
                          <Input
                            type="number"
                            inputMode="numeric"
                            className="text-right"
                            min={0}
                            max={65000}
                            value={
                              typeof controllerField.value === 'number'
                                ? controllerField.value.toString()
                                : ''
                            }
                            onChange={(e) => {
                              const value = e.target.value;
                              controllerField.onChange(
                                value === '' ? 0 : Number(value),
                              );
                            }}
                            onBlur={controllerField.onBlur}
                          />
                        )}
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <Controller
                        name={`attacks.${index}.additional_damage`}
                        control={control}
                        render={({ field: controllerField }) => (
                          <Input
                            type="number"
                            inputMode="numeric"
                            className="text-right"
                            min={0}
                            max={65000}
                            value={
                              typeof controllerField.value === 'number'
                                ? controllerField.value.toString()
                                : ''
                            }
                            onChange={(e) => {
                              const value = e.target.value;
                              controllerField.onChange(
                                value === '' ? 0 : Number(value),
                              );
                            }}
                            onBlur={controllerField.onBlur}
                          />
                        )}
                      />
                    </TableCell>
                  </TableRow>
                ))}
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
            Save NPC
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
