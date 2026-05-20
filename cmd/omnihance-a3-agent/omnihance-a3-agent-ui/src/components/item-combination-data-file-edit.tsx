import { useEffect, useId, useMemo } from 'react';
import { Controller, useFieldArray, useForm, useWatch } from 'react-hook-form';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useRouter } from '@tanstack/react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Hammer, Loader2, Plus, Save, Trash2, X } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { useVirtualRows } from '@/hooks/use-virtual-rows';
import { toast } from 'sonner';
import {
  APIError,
  getItems,
  updateItemCombinationDataFile,
  type GameClientDataResponse,
  type ItemCombinationDataFileAPIData,
} from '@/lib/api';
import { queryKeys } from '@/constants';

const INGREDIENT_COUNT = 10;
const MAX_UINT16_FORM = 65535;
const MAX_SUCCESS_RATE = 120;
const EMPTY_ITEM_CODE = 0;
const FORMULA_ROW_HEIGHT = 176;

const formulaSchema = z.object({
  ingredients: z
    .array(z.number().int().min(0).max(MAX_UINT16_FORM))
    .length(INGREDIENT_COUNT),
  success_rate: z.number().int().min(1).max(MAX_SUCCESS_RATE),
  outcome: z.number().int().min(0).max(MAX_UINT16_FORM),
});

const itemCombinationDataFileSchema = z.object({
  formulas: z.array(formulaSchema),
});

type ItemCombinationDataFileFormData = z.infer<
  typeof itemCombinationDataFileSchema
>;

interface ItemCombinationDataFileEditProps {
  filePath: string;
  defaultData: ItemCombinationDataFileAPIData;
}

export function ItemCombinationDataFileEdit({
  filePath,
  defaultData,
}: ItemCombinationDataFileEditProps) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const itemCodeListId = useId();

  const form = useForm<ItemCombinationDataFileFormData>({
    resolver: zodResolver(itemCombinationDataFileSchema),
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

  const formulasArray = useFieldArray({
    control,
    name: 'formulas',
  });
  const { containerRef, onScroll, totalHeight, virtualRows } = useVirtualRows({
    count: formulasArray.fields.length,
    rowHeight: FORMULA_ROW_HEIGHT,
    overscan: 4,
  });

  const mutation = useMutation({
    mutationFn: (values: ItemCombinationDataFileFormData) =>
      updateItemCombinationDataFile({ path: filePath }, values),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.itemCombinationDataFile(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.fileTree(filePath),
      });
      toast.success('Item combination data saved');
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
            : 'Failed to save item combination data';
      toast.error(errorMessage);
    },
  });

  const mutationErrorMessage =
    mutation.error instanceof APIError
      ? mutation.error.getErrorMessage()
      : mutation.error instanceof Error
        ? mutation.error.message
        : 'Failed to save item combination data';

  const isSaving = mutation.status === 'pending';

  const addFormula = () => {
    formulasArray.append(createEmptyFormula());
  };

  const removeFormula = (index: number) => {
    formulasArray.remove(index);
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
        <option value={EMPTY_ITEM_CODE}>Empty</option>
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
              <Hammer className="h-5 w-5" />
              Formulas
            </CardTitle>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={addFormula}
            >
              <Plus className="mr-2 h-4 w-4" />
              Add Formula
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {formulasArray.fields.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              <p className="mb-4">No formulas configured</p>
              <Button type="button" variant="outline" onClick={addFormula}>
                <Plus className="mr-2 h-4 w-4" />
                Add First Formula
              </Button>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <div className="min-w-[1160px]">
                <div className="grid grid-cols-[4rem_12rem_7rem_1fr_5rem] divide-x border-b bg-card text-sm font-medium text-muted-foreground">
                  <div className="px-3 py-2">#</div>
                  <div className="px-3 py-2">Outcome</div>
                  <div className="px-3 py-2 text-right">Success</div>
                  <div className="px-3 py-2">Ingredients</div>
                  <div className="px-3 py-2 text-right">Actions</div>
                </div>
                <div
                  ref={containerRef}
                  onScroll={onScroll}
                  className="relative h-[min(70vh,680px)] overflow-y-auto"
                  role="rowgroup"
                  aria-label="Item combination formula editor"
                >
                  <div className="relative" style={{ height: totalHeight }}>
                    {virtualRows.map(({ index, top }) => {
                      const field = formulasArray.fields[index];

                      return (
                        <FormulaRow
                          key={field.id}
                          index={index}
                          top={top}
                          control={control}
                          itemCodeListId={itemCodeListId}
                          itemLookup={itemLookup}
                          removeFormula={removeFormula}
                        />
                      );
                    })}
                  </div>
                </div>
              </div>
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
            Save Item Combination Data
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

interface FormulaRowProps {
  index: number;
  top: number;
  control: ReturnType<
    typeof useForm<ItemCombinationDataFileFormData>
  >['control'];
  itemCodeListId: string;
  itemLookup: Map<number, string>;
  removeFormula: (index: number) => void;
}

function FormulaRow({
  index,
  top,
  control,
  itemCodeListId,
  itemLookup,
  removeFormula,
}: FormulaRowProps) {
  const outcome = useWatch({
    control,
    name: `formulas.${index}.outcome`,
  });

  return (
    <div
      className="absolute left-0 right-0 grid grid-cols-[4rem_12rem_7rem_1fr_5rem] items-start divide-x border-b"
      style={{
        height: FORMULA_ROW_HEIGHT,
        transform: `translateY(${top}px)`,
      }}
      role="row"
      aria-rowindex={index + 1}
    >
      <div className="px-3 py-3 font-medium">{index + 1}</div>
      <div className="min-w-0 px-3 py-3">
        <Controller
          name={`formulas.${index}.outcome`}
          control={control}
          render={({ field }) => (
            <ItemCodeInput
              ariaLabel={`Formula ${index + 1} outcome item code`}
              itemCodeListId={itemCodeListId}
              value={field.value}
              onChange={field.onChange}
              onBlur={field.onBlur}
            />
          )}
        />
        <div className="mt-1 max-w-48 truncate text-xs text-muted-foreground">
          {formatItemName(outcome, itemLookup)}
        </div>
      </div>
      <div className="px-3 py-3 text-right">
        <Controller
          name={`formulas.${index}.success_rate`}
          control={control}
          render={({ field }) => (
            <Input
              type="number"
              inputMode="numeric"
              className="ml-auto w-24 text-right"
              min={1}
              max={MAX_SUCCESS_RATE}
              value={typeof field.value === 'number' ? String(field.value) : ''}
              onChange={(e) => {
                const value = e.target.value;
                field.onChange(value === '' ? 1 : Number(value));
              }}
              onBlur={field.onBlur}
              aria-label={`Formula ${index + 1} success rate`}
            />
          )}
        />
      </div>
      <div className="min-w-0 px-3 py-3">
        <div className="grid grid-cols-5 gap-2">
          {Array.from({ length: INGREDIENT_COUNT }, (_, ingredientIndex) => (
            <IngredientInput
              key={ingredientIndex}
              formulaIndex={index}
              ingredientIndex={ingredientIndex}
              control={control}
              itemCodeListId={itemCodeListId}
              itemLookup={itemLookup}
            />
          ))}
        </div>
      </div>
      <div className="px-3 py-3 text-right">
        <Button
          type="button"
          variant="outline"
          size="icon"
          aria-label={`Remove formula ${index + 1}`}
          onClick={() => removeFormula(index)}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

interface IngredientInputProps {
  formulaIndex: number;
  ingredientIndex: number;
  control: ReturnType<
    typeof useForm<ItemCombinationDataFileFormData>
  >['control'];
  itemCodeListId: string;
  itemLookup: Map<number, string>;
}

function IngredientInput({
  formulaIndex,
  ingredientIndex,
  control,
  itemCodeListId,
  itemLookup,
}: IngredientInputProps) {
  const itemCode = useWatch({
    control,
    name: `formulas.${formulaIndex}.ingredients.${ingredientIndex}`,
  });

  return (
    <div className="min-w-32">
      <Controller
        name={`formulas.${formulaIndex}.ingredients.${ingredientIndex}`}
        control={control}
        render={({ field }) => (
          <ItemCodeInput
            ariaLabel={`Formula ${formulaIndex + 1} ingredient ${
              ingredientIndex + 1
            } item code`}
            itemCodeListId={itemCodeListId}
            value={field.value}
            onChange={field.onChange}
            onBlur={field.onBlur}
          />
        )}
      />
      <div className="mt-1 max-w-32 truncate text-xs text-muted-foreground">
        {ingredientIndex + 1}. {formatItemName(itemCode, itemLookup)}
      </div>
    </div>
  );
}

interface ItemCodeInputProps {
  ariaLabel: string;
  itemCodeListId: string;
  value: number;
  onChange: (value: number) => void;
  onBlur: () => void;
}

function ItemCodeInput({
  ariaLabel,
  itemCodeListId,
  value,
  onChange,
  onBlur,
}: ItemCodeInputProps) {
  return (
    <Input
      type="number"
      inputMode="numeric"
      list={itemCodeListId}
      className="w-32 text-right"
      min={0}
      max={MAX_UINT16_FORM}
      value={typeof value === 'number' ? String(value) : ''}
      onChange={(e) => {
        const nextValue = e.target.value;
        onChange(nextValue === '' ? EMPTY_ITEM_CODE : Number(nextValue));
      }}
      onBlur={onBlur}
      aria-label={ariaLabel}
    />
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
    if (!map.has(item.id)) {
      map.set(item.id, item.name);
    }
  }

  return map;
}

function createEmptyFormula(): ItemCombinationDataFileFormData['formulas'][number] {
  return {
    ingredients: Array.from(
      { length: INGREDIENT_COUNT },
      () => EMPTY_ITEM_CODE,
    ),
    success_rate: 1,
    outcome: EMPTY_ITEM_CODE,
  };
}

function formatItemName(
  itemCode: number | undefined,
  itemLookup: Map<number, string>,
): string {
  if (itemCode === undefined) {
    return '-';
  }

  if (itemCode === EMPTY_ITEM_CODE) {
    return 'Empty';
  }

  return itemLookup.get(itemCode) || 'Custom item code';
}

function formatItemOption(item: GameClientDataResponse): string {
  const itemType = item.item_type ? item.item_type.toUpperCase() : 'ITEM';
  return `${item.name} (${itemType})`;
}
