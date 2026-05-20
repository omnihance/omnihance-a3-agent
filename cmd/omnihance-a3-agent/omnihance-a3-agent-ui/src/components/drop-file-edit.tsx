import { useEffect, useId, useMemo } from 'react';
import { Controller, useFieldArray, useForm, useWatch } from 'react-hook-form';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useRouter } from '@tanstack/react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Loader2, Package, Plus, Save, Trash2, X } from 'lucide-react';
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
import { toast } from 'sonner';
import {
  APIError,
  getItems,
  updateDropFile,
  type DropFileAPIData,
  type GameClientDataResponse,
} from '@/lib/api';
import { queryKeys } from '@/constants';

const MAX_UINT16_FORM = 65535;
const EMPTY_ITEM_CODE = 0xffff;
const ITEM_ID_MASK = 0x3fff;

const dropSchema = z.object({
  item_code: z.number().int().min(0).max(MAX_UINT16_FORM),
  drop_rate: z.number().int().min(0).max(MAX_UINT16_FORM),
  drop_group: z.number().int().min(0).max(MAX_UINT16_FORM),
});

const dropFileSchema = z.object({
  drops: z.array(dropSchema),
});

type DropFileFormData = z.infer<typeof dropFileSchema>;

interface DropFileEditProps {
  filePath: string;
  defaultData: DropFileAPIData;
}

export function DropFileEdit({ filePath, defaultData }: DropFileEditProps) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const itemCodeListId = useId();

  const form = useForm<DropFileFormData>({
    resolver: zodResolver(dropFileSchema),
    defaultValues: defaultData,
  });

  useEffect(() => {
    form.reset(defaultData);
  }, [defaultData, form]);

  const { control } = form;

  const { data: items } = useQuery({
    queryKey: queryKeys.items,
    queryFn: () => getItems(),
  });

  const itemLookup = useMemo(() => createItemLookup(items), [items]);
  const itemOptions = useMemo(() => items ?? [], [items]);

  const dropsArray = useFieldArray({
    control,
    name: 'drops',
  });

  const mutation = useMutation({
    mutationFn: (values: DropFileFormData) =>
      updateDropFile({ path: filePath }, values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.dropFile(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.fileTree(filePath),
      });
      toast.success('Drop file saved');
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
            : 'Failed to save drop file';
      toast.error(errorMessage);
    },
  });

  const mutationErrorMessage =
    mutation.error instanceof APIError
      ? mutation.error.getErrorMessage()
      : mutation.error instanceof Error
        ? mutation.error.message
        : 'Failed to save drop file';

  const isSaving = mutation.status === 'pending';

  const addDrop = () => {
    dropsArray.append({
      item_code: 0,
      drop_rate: 1,
      drop_group: 0,
    });
  };

  const removeDrop = (index: number) => {
    dropsArray.remove(index);
  };

  const addEmptyDrop = () => {
    dropsArray.append({
      item_code: EMPTY_ITEM_CODE,
      drop_rate: 0,
      drop_group: 0,
    });
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

      <datalist id={itemCodeListId}>
        <option value={EMPTY_ITEM_CODE}>Empty slot</option>
        {itemOptions.map((item) => (
          <option key={`${item.item_type}-${item.id}`} value={item.id}>
            {formatItemOption(item)}
          </option>
        ))}
      </datalist>

      <Card>
        <CardHeader>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <CardTitle className="flex items-center gap-2">
              <Package className="h-5 w-5" />
              Drop Entries
            </CardTitle>
            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={addDrop}
              >
                <Plus className="mr-2 h-4 w-4" />
                Add Drop
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={addEmptyDrop}
              >
                <Plus className="mr-2 h-4 w-4" />
                Add Empty Slot
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {dropsArray.fields.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              <p className="mb-4">No drop entries configured</p>
              <Button type="button" variant="outline" onClick={addDrop}>
                <Plus className="mr-2 h-4 w-4" />
                Add First Drop
              </Button>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>#</TableHead>
                    <TableHead>Item Name</TableHead>
                    <TableHead className="text-right">Item Code</TableHead>
                    <TableHead className="text-right">Drop Rate</TableHead>
                    <TableHead className="text-right">Drop Group</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {dropsArray.fields.map((field, index) => (
                    <DropRow
                      key={field.id}
                      index={index}
                      control={control}
                      itemCodeListId={itemCodeListId}
                      itemLookup={itemLookup}
                      removeDrop={removeDrop}
                    />
                  ))}
                </TableBody>
              </Table>
            </div>
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
            Save Drop File
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

interface DropRowProps {
  index: number;
  control: ReturnType<typeof useForm<DropFileFormData>>['control'];
  itemCodeListId: string;
  itemLookup: Map<number, string>;
  removeDrop: (index: number) => void;
}

function DropRow({
  index,
  control,
  itemCodeListId,
  itemLookup,
  removeDrop,
}: DropRowProps) {
  const itemCode = useWatch({
    control,
    name: `drops.${index}.item_code`,
  });

  return (
    <TableRow>
      <TableCell className="font-medium">{index + 1}</TableCell>
      <TableCell className="min-w-48">
        {formatItemName(itemCode, itemLookup)}
      </TableCell>
      <TableCell className="text-right">
        <Controller
          name={`drops.${index}.item_code`}
          control={control}
          render={({ field }) => (
            <Input
              type="number"
              inputMode="numeric"
              list={itemCodeListId}
              className="ml-auto w-32 text-right"
              min={0}
              max={MAX_UINT16_FORM}
              value={typeof field.value === 'number' ? String(field.value) : ''}
              onChange={(e) => {
                const value = e.target.value;
                field.onChange(value === '' ? 0 : Number(value));
              }}
              onBlur={field.onBlur}
              aria-label={`Drop ${index + 1} item code`}
            />
          )}
        />
      </TableCell>
      <TableCell className="text-right">
        <Controller
          name={`drops.${index}.drop_rate`}
          control={control}
          render={({ field }) => (
            <Input
              type="number"
              inputMode="numeric"
              className="ml-auto w-28 text-right"
              min={0}
              max={MAX_UINT16_FORM}
              value={typeof field.value === 'number' ? String(field.value) : ''}
              onChange={(e) => {
                const value = e.target.value;
                field.onChange(value === '' ? 0 : Number(value));
              }}
              onBlur={field.onBlur}
              aria-label={`Drop ${index + 1} rate`}
            />
          )}
        />
      </TableCell>
      <TableCell className="text-right">
        <Controller
          name={`drops.${index}.drop_group`}
          control={control}
          render={({ field }) => (
            <Input
              type="number"
              inputMode="numeric"
              className="ml-auto w-28 text-right"
              min={0}
              max={MAX_UINT16_FORM}
              value={typeof field.value === 'number' ? String(field.value) : ''}
              onChange={(e) => {
                const value = e.target.value;
                field.onChange(value === '' ? 0 : Number(value));
              }}
              onBlur={field.onBlur}
              aria-label={`Drop ${index + 1} group`}
            />
          )}
        />
      </TableCell>
      <TableCell className="text-right">
        <Button
          type="button"
          variant="outline"
          size="icon"
          aria-label={`Remove drop ${index + 1}`}
          onClick={() => removeDrop(index)}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </TableCell>
    </TableRow>
  );
}

function createItemLookup(
  items: GameClientDataResponse[] | undefined,
): Map<number, string> {
  if (!items) {
    return new Map<number, string>();
  }

  const map = new Map<number, string>();
  for (const item of items) {
    const baseCode = item.id & ITEM_ID_MASK;
    if (!map.has(baseCode)) {
      map.set(baseCode, item.name);
    }
  }

  return map;
}

function formatItemName(
  itemCode: number | undefined,
  itemLookup: Map<number, string>,
): string {
  if (itemCode === undefined) {
    return '-';
  }

  if (itemCode === EMPTY_ITEM_CODE) {
    return 'Empty slot';
  }

  const baseCode = itemCode & ITEM_ID_MASK;
  const itemName = itemLookup.get(baseCode);
  if (itemName && baseCode !== itemCode) {
    return `${itemName} (base ${baseCode})`;
  }

  return itemName || 'Custom item code';
}

function formatItemOption(item: GameClientDataResponse): string {
  const itemType = item.item_type ? item.item_type.toUpperCase() : 'ITEM';
  return `${item.name} (${itemType})`;
}
