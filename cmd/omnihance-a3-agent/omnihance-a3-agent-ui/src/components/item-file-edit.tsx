import { useDeferredValue, useEffect, useMemo, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Link, useRouter } from '@tanstack/react-router';
import {
  ListTree,
  Loader2,
  PackagePlus,
  Plus,
  Save,
  Trash2,
  X,
} from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
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
import { queryKeys } from '@/constants';
import { useVirtualRows } from '@/hooks/use-virtual-rows';
import {
  APIError,
  updateItemFile,
  type ItemFileAPIData,
  type ItemFileBaseItemAPIData,
  type ItemFileItemAPIData,
  type ItemFileLevelAPIData,
  type ItemFileNameEncoding,
} from '@/lib/api';
import { toast } from 'sonner';

const MAX_UINT16 = 65535;
const MAX_UINT32 = 4294967295;
const DEFAULT_ITEM_EDIT_ROW_HEIGHT = 108;
const IT1_ITEM_EDIT_ROW_HEIGHT = 220;
const IT2_ITEM_EDIT_ROW_HEIGHT = 156;
const BASE_ITEM_ROW_HEIGHT = 56;
const ITEM_NAME_ENCODING_OPTIONS: {
  value: ItemFileNameEncoding;
  label: string;
}[] = [
  { value: 'utf-8', label: 'UTF-8' },
  { value: 'euc-kr', label: 'EUC-KR' },
  { value: 'gbk', label: 'GBK' },
  { value: 'big5', label: 'Big5' },
  { value: 'shift-jis', label: 'Shift-JIS' },
];

interface ItemFileEditProps {
  filePath: string;
  defaultData: ItemFileAPIData;
  nameEncoding: ItemFileNameEncoding;
  onNameEncodingChange: (encoding: ItemFileNameEncoding) => void;
}

type ItemField =
  | 'name'
  | 'slot'
  | 'npc_price'
  | 'required_level'
  | 'attribute'
  | 'blue_option'
  | 'red_option'
  | 'grey_option'
  | 'class'
  | 'skill_level';

type LevelField =
  | 'additional_attribute'
  | 'strength'
  | 'dexterity'
  | 'intelligence'
  | 'attribute'
  | 'attribute_range'
  | 'blue_option'
  | 'red_option'
  | 'grey_option';

export function ItemFileEdit({
  filePath,
  defaultData,
  nameEncoding,
  onNameEncodingChange,
}: ItemFileEditProps) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [items, setItems] = useState(defaultData.items);
  const [query, setQuery] = useState('');
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [baseQuery, setBaseQuery] = useState('');
  const [levelEditorIndex, setLevelEditorIndex] = useState<number | null>(null);
  const deferredQuery = useDeferredValue(query);
  const deferredBaseQuery = useDeferredValue(baseQuery);
  const normalizedQuery = deferredQuery.trim().toLowerCase();
  const normalizedBaseQuery = deferredBaseQuery.trim().toLowerCase();

  const filteredItems = useMemo(
    () => filterIndexedItems(items, normalizedQuery),
    [items, normalizedQuery],
  );
  const visibleItemCount = filteredItems?.length ?? items.length;
  const itemRowHeight = getItemEditRowHeight(defaultData.item_file_type);
  const selectedRows = useMemo(
    () => new Set(items.map((item) => item.row)),
    [items],
  );
  const duplicateRows = useMemo(() => findDuplicateRows(items), [items]);
  const filteredBaseItems = useMemo(
    () => filterBaseItems(defaultData.base_items ?? [], normalizedBaseQuery),
    [defaultData.base_items, normalizedBaseQuery],
  );
  const {
    containerRef: itemContainerRef,
    onScroll: onItemScroll,
    resetScrollTop: resetItemScrollTop,
    totalHeight: itemTotalHeight,
    virtualRows: virtualItemRows,
  } = useVirtualRows({
    count: visibleItemCount,
    rowHeight: itemRowHeight,
    overscan: 6,
  });
  const {
    containerRef: baseContainerRef,
    onScroll: onBaseScroll,
    resetScrollTop: resetBaseScrollTop,
    totalHeight: baseTotalHeight,
    virtualRows: virtualBaseRows,
  } = useVirtualRows({
    count: filteredBaseItems.length,
    rowHeight: BASE_ITEM_ROW_HEIGHT,
    overscan: 8,
  });

  useEffect(() => {
    resetItemScrollTop();
  }, [normalizedQuery, resetItemScrollTop]);

  useEffect(() => {
    resetBaseScrollTop();
  }, [normalizedBaseQuery, resetBaseScrollTop]);

  const mutation = useMutation({
    mutationFn: () =>
      updateItemFile(
        { path: filePath, name_encoding: nameEncoding },
        {
          item_file_type: defaultData.item_file_type,
          name_encoding: nameEncoding,
          items,
        },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.itemFile(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.fileTree(filePath),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.revisionSummary(filePath),
      });
      toast.success('Item file saved');
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
            : 'Failed to save item file';
      toast.error(errorMessage);
    },
  });

  const mutationErrorMessage =
    mutation.error instanceof APIError
      ? mutation.error.getErrorMessage()
      : mutation.error instanceof Error
        ? mutation.error.message
        : 'Failed to save item file';

  const updateItem = (
    index: number,
    field: ItemField,
    value: string | number,
  ) => {
    setItems((currentItems) =>
      currentItems.map((item, itemIndex) =>
        itemIndex === index ? { ...item, [field]: value } : item,
      ),
    );
  };

  const updateLevel = (
    itemIndex: number,
    levelIndex: number,
    field: LevelField,
    value: number,
  ) => {
    setItems((currentItems) =>
      currentItems.map((item, currentItemIndex) => {
        if (currentItemIndex !== itemIndex) {
          return item;
        }

        const levels = (item.levels ?? []).map((level, currentLevelIndex) =>
          currentLevelIndex === levelIndex
            ? { ...level, [field]: value }
            : level,
        );
        return { ...item, levels };
      }),
    );
  };

  const addIT0ExItem = (baseItem: ItemFileBaseItemAPIData) => {
    if (baseItem.row === undefined || selectedRows.has(baseItem.row)) {
      return;
    }

    setItems((currentItems) => {
      const newItemIndex = currentItems.length;
      const newItems = [
        ...currentItems,
        {
          row: baseItem.row,
          item_code: baseItem.item_code,
          name: baseItem.name,
          levels: createIT0ExLevels(baseItem),
        },
      ];

      setLevelEditorIndex(newItemIndex);
      return newItems;
    });
    setAddDialogOpen(false);
    setBaseQuery('');
  };

  const removeIT0ExItem = (index: number) => {
    setItems((currentItems) =>
      currentItems.filter((_, itemIndex) => itemIndex !== index),
    );
    setLevelEditorIndex((currentIndex) => {
      if (currentIndex === null) {
        return null;
      }

      if (currentIndex === index) {
        return null;
      }

      if (currentIndex > index) {
        return currentIndex - 1;
      }

      return currentIndex;
    });
  };

  const isSaving = mutation.status === 'pending';
  const canAddOrDelete = defaultData.item_file_type === 'it0ex';
  const levelEditorItem =
    levelEditorIndex === null ? undefined : items[levelEditorIndex];
  const countLabel =
    filteredItems !== null
      ? `${filteredItems.length}/${items.length}`
      : String(items.length);

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        mutation.mutate();
      }}
      className="space-y-6"
    >
      {mutation.isError && (
        <Alert variant="destructive">
          <AlertDescription>{mutationErrorMessage}</AlertDescription>
        </Alert>
      )}

      {duplicateRows.length > 0 && (
        <Alert variant="destructive">
          <AlertDescription>
            Duplicate IT0Ex rows: {duplicateRows.join(', ')}
          </AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <CardTitle className="flex items-center gap-2">
              <PackagePlus className="h-5 w-5" />
              Item Records ({countLabel})
            </CardTitle>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
              <Select
                value={nameEncoding}
                onValueChange={(value) =>
                  onNameEncodingChange(value as ItemFileNameEncoding)
                }
              >
                <SelectTrigger
                  className="w-full sm:w-36"
                  aria-label="Item name encoding"
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ITEM_NAME_ENCODING_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search row, code, or name"
                aria-label="Search editable item records"
                className="w-full sm:w-72"
              />
              {canAddOrDelete && (
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setAddDialogOpen(true)}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Add Item
                </Button>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {items.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              No item records configured
            </div>
          ) : visibleItemCount === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              No item records found
            </div>
          ) : (
            <div className="overflow-x-auto rounded-md border">
              <div className="min-w-[1160px]">
                <div className="grid grid-cols-[5rem_18rem_8rem_6rem_1fr_8rem] divide-x border-b bg-card text-sm font-medium text-muted-foreground">
                  <div className="px-3 py-2">Row</div>
                  <div className="px-3 py-2">Item</div>
                  <div className="px-3 py-2 text-right">Code</div>
                  <div className="px-3 py-2 text-right">Type</div>
                  <div className="px-3 py-2">Editable Fields</div>
                  <div className="px-3 py-2 text-right">Actions</div>
                </div>
                <div
                  ref={itemContainerRef}
                  onScroll={onItemScroll}
                  className="relative h-[min(70vh,700px)] overflow-y-auto"
                  role="rowgroup"
                  aria-label="Editable item records"
                >
                  <div className="relative" style={{ height: itemTotalHeight }}>
                    {virtualItemRows.map(({ index, top }) => {
                      const filteredItem = filteredItems?.[index];
                      const item = filteredItem?.item ?? items[index];
                      const itemIndex = filteredItem?.index ?? index;

                      return (
                        <ItemEditRow
                          key={`${item.row ?? itemIndex}-${item.item_code ?? itemIndex}`}
                          item={item}
                          index={itemIndex}
                          itemFileType={defaultData.item_file_type}
                          canAddOrDelete={canAddOrDelete}
                          top={top}
                          rowHeight={itemRowHeight}
                          updateItem={updateItem}
                          removeIT0ExItem={removeIT0ExItem}
                          editLevels={() => setLevelEditorIndex(itemIndex)}
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
        <Button type="submit" disabled={isSaving || duplicateRows.length > 0}>
          <span className="flex items-center gap-1.5">
            {isSaving ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            Save Item File
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

      <Dialog open={addDialogOpen} onOpenChange={setAddDialogOpen}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>Add IT0 Extension Item</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <Input
              value={baseQuery}
              onChange={(event) => setBaseQuery(event.target.value)}
              placeholder="Search base items"
              aria-label="Search base IT0 items"
            />
            <div className="overflow-x-auto rounded-md border">
              <div className="min-w-[640px]">
                <div className="grid grid-cols-[5rem_1fr_8rem_7rem] divide-x border-b bg-card text-sm font-medium text-muted-foreground">
                  <div className="px-3 py-2">Row</div>
                  <div className="px-3 py-2">Item</div>
                  <div className="px-3 py-2 text-right">Code</div>
                  <div className="px-3 py-2 text-right">Action</div>
                </div>
                {filteredBaseItems.length === 0 ? (
                  <div className="py-8 text-center text-muted-foreground">
                    No base items found
                  </div>
                ) : (
                  <div
                    ref={baseContainerRef}
                    onScroll={onBaseScroll}
                    className="relative h-96 overflow-y-auto"
                    role="rowgroup"
                    aria-label="Base IT0 items"
                  >
                    <div
                      className="relative"
                      style={{ height: baseTotalHeight }}
                    >
                      {virtualBaseRows.map(({ index, top }) => {
                        const baseItem = filteredBaseItems[index];
                        const alreadyAdded =
                          baseItem.row !== undefined &&
                          selectedRows.has(baseItem.row);

                        return (
                          <BaseItemRow
                            key={`${baseItem.row}-${baseItem.item_code}`}
                            baseItem={baseItem}
                            alreadyAdded={alreadyAdded}
                            top={top}
                            addIT0ExItem={addIT0ExItem}
                          />
                        );
                      })}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <LevelEditDialog
        item={levelEditorItem}
        itemIndex={levelEditorIndex}
        open={levelEditorIndex !== null}
        onOpenChange={(open) => {
          if (!open) {
            setLevelEditorIndex(null);
          }
        }}
        updateLevel={updateLevel}
      />
    </form>
  );
}

function ItemEditRow({
  item,
  index,
  itemFileType,
  canAddOrDelete,
  top,
  rowHeight,
  updateItem,
  removeIT0ExItem,
  editLevels,
}: {
  item: ItemFileItemAPIData;
  index: number;
  itemFileType: ItemFileAPIData['item_file_type'];
  canAddOrDelete: boolean;
  top: number;
  rowHeight: number;
  updateItem: (index: number, field: ItemField, value: string | number) => void;
  removeIT0ExItem: (index: number) => void;
  editLevels: () => void;
}) {
  const hasLevels = (item.levels?.length ?? 0) > 0;

  return (
    <div
      className="absolute left-0 right-0 grid grid-cols-[5rem_18rem_8rem_6rem_1fr_8rem] items-start divide-x border-b"
      style={{
        height: rowHeight,
        transform: `translateY(${top}px)`,
      }}
      role="row"
    >
      <div className="px-3 py-3 font-mono text-sm">{formatValue(item.row)}</div>
      <div className="min-w-0 px-3 py-3">
        {itemFileType === 'it0ex' ? (
          <div>
            <div className="truncate font-medium">{item.name || '-'}</div>
            {item.item_code_base !== undefined && (
              <div className="text-xs text-muted-foreground">
                Base {item.item_code_base}
              </div>
            )}
          </div>
        ) : (
          <Input
            value={item.name}
            onChange={(event) => updateItem(index, 'name', event.target.value)}
            aria-label={`Item row ${formatValue(item.row)} name`}
          />
        )}
      </div>
      <div className="px-3 py-3 text-right font-mono text-sm">
        {formatValue(item.item_code)}
      </div>
      <div className="px-3 py-3 text-right font-mono text-sm">
        {formatValue(item.type)}
      </div>
      <div className="px-3 py-3">
        <div className="grid grid-cols-2 gap-3 xl:grid-cols-3">
          {renderItemEditors(item, index, itemFileType, updateItem)}
        </div>
      </div>
      <div className="flex flex-col items-end gap-2 px-3 py-3">
        {hasLevels && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={editLevels}
            aria-label={`Edit levels for item row ${formatValue(item.row)}`}
          >
            <ListTree className="mr-2 h-4 w-4" />
            Levels
          </Button>
        )}
        {canAddOrDelete && (
          <>
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={() => removeIT0ExItem(index)}
              aria-label={`Remove IT0Ex row ${formatValue(item.row)}`}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </>
        )}
      </div>
    </div>
  );
}

function BaseItemRow({
  baseItem,
  alreadyAdded,
  top,
  addIT0ExItem,
}: {
  baseItem: ItemFileBaseItemAPIData;
  alreadyAdded: boolean;
  top: number;
  addIT0ExItem: (baseItem: ItemFileBaseItemAPIData) => void;
}) {
  return (
    <div
      className="absolute left-0 right-0 grid grid-cols-[5rem_1fr_8rem_7rem] items-center divide-x border-b"
      style={{
        height: BASE_ITEM_ROW_HEIGHT,
        transform: `translateY(${top}px)`,
      }}
      role="row"
    >
      <div className="px-3 font-mono">{formatValue(baseItem.row)}</div>
      <div className="min-w-0 truncate px-3">{baseItem.name || '-'}</div>
      <div className="px-3 text-right font-mono">
        {formatValue(baseItem.item_code)}
      </div>
      <div className="px-3 text-right">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={alreadyAdded || baseItem.row === undefined}
          onClick={() => addIT0ExItem(baseItem)}
        >
          <Plus className="mr-2 h-4 w-4" />
          Add
        </Button>
      </div>
    </div>
  );
}

function LevelEditDialog({
  item,
  itemIndex,
  open,
  onOpenChange,
  updateLevel,
}: {
  item: ItemFileItemAPIData | undefined;
  itemIndex: number | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  updateLevel: (
    itemIndex: number,
    levelIndex: number,
    field: LevelField,
    value: number,
  ) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-6xl">
        <DialogHeader>
          <DialogTitle>
            Edit levels for {item?.name || `Row ${formatValue(item?.row)}`}
          </DialogTitle>
        </DialogHeader>
        <div className="max-h-[70vh] overflow-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-20 text-right">Level</TableHead>
                <TableHead className="min-w-28 text-right">Add Attr</TableHead>
                <TableHead className="min-w-24 text-right">Str</TableHead>
                <TableHead className="min-w-24 text-right">Dex</TableHead>
                <TableHead className="min-w-24 text-right">Int</TableHead>
                <TableHead className="min-w-24 text-right">Attr</TableHead>
                <TableHead className="min-w-24 text-right">Range</TableHead>
                <TableHead className="min-w-24 text-right">Blue</TableHead>
                <TableHead className="min-w-24 text-right">Red</TableHead>
                <TableHead className="min-w-24 text-right">Grey</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(item?.levels ?? []).map((level, levelIndex) => (
                <LevelEditRow
                  key={`${item?.row ?? 'row'}-level-${level.level ?? levelIndex}`}
                  itemIndex={itemIndex ?? 0}
                  levelIndex={levelIndex}
                  level={level}
                  updateLevel={updateLevel}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function LevelEditRow({
  itemIndex,
  levelIndex,
  level,
  updateLevel,
}: {
  itemIndex: number;
  levelIndex: number;
  level: ItemFileLevelAPIData;
  updateLevel: (
    itemIndex: number,
    levelIndex: number,
    field: LevelField,
    value: number,
  ) => void;
}) {
  return (
    <TableRow>
      <TableCell className="text-right font-mono">
        {formatValue(level.level)}
      </TableCell>
      <TableCell>
        <NumberInput
          value={level.additional_attribute}
          max={MAX_UINT16}
          ariaLabel={`Level ${formatValue(level.level)} additional attribute`}
          onChange={(value) =>
            updateLevel(itemIndex, levelIndex, 'additional_attribute', value)
          }
        />
      </TableCell>
      <TableCell>
        <NumberInput
          value={level.strength}
          max={MAX_UINT16}
          ariaLabel={`Level ${formatValue(level.level)} strength`}
          onChange={(value) =>
            updateLevel(itemIndex, levelIndex, 'strength', value)
          }
        />
      </TableCell>
      <TableCell>
        <NumberInput
          value={level.dexterity}
          max={MAX_UINT16}
          ariaLabel={`Level ${formatValue(level.level)} dexterity`}
          onChange={(value) =>
            updateLevel(itemIndex, levelIndex, 'dexterity', value)
          }
        />
      </TableCell>
      <TableCell>
        <NumberInput
          value={level.intelligence}
          max={MAX_UINT16}
          ariaLabel={`Level ${formatValue(level.level)} intelligence`}
          onChange={(value) =>
            updateLevel(itemIndex, levelIndex, 'intelligence', value)
          }
        />
      </TableCell>
      <TableCell>
        <NumberInput
          value={level.attribute}
          max={MAX_UINT16}
          ariaLabel={`Level ${formatValue(level.level)} attribute`}
          onChange={(value) =>
            updateLevel(itemIndex, levelIndex, 'attribute', value)
          }
        />
      </TableCell>
      <TableCell>
        <NumberInput
          value={level.attribute_range}
          max={MAX_UINT16}
          ariaLabel={`Level ${formatValue(level.level)} attribute range`}
          onChange={(value) =>
            updateLevel(itemIndex, levelIndex, 'attribute_range', value)
          }
        />
      </TableCell>
      <TableCell>
        <NumberInput
          value={level.blue_option}
          max={MAX_UINT16}
          ariaLabel={`Level ${formatValue(level.level)} blue option`}
          onChange={(value) =>
            updateLevel(itemIndex, levelIndex, 'blue_option', value)
          }
        />
      </TableCell>
      <TableCell>
        <NumberInput
          value={level.red_option}
          max={MAX_UINT16}
          ariaLabel={`Level ${formatValue(level.level)} red option`}
          onChange={(value) =>
            updateLevel(itemIndex, levelIndex, 'red_option', value)
          }
        />
      </TableCell>
      <TableCell>
        <NumberInput
          value={level.grey_option}
          max={MAX_UINT16}
          ariaLabel={`Level ${formatValue(level.level)} grey option`}
          onChange={(value) =>
            updateLevel(itemIndex, levelIndex, 'grey_option', value)
          }
        />
      </TableCell>
    </TableRow>
  );
}

function renderItemEditors(
  item: ItemFileItemAPIData,
  index: number,
  itemFileType: ItemFileAPIData['item_file_type'],
  updateItem: (index: number, field: ItemField, value: string | number) => void,
) {
  if (itemFileType === 'it0') {
    return [
      <NumberField
        key="slot"
        label="Slot"
        value={item.slot}
        onChange={(value) => updateItem(index, 'slot', value)}
      />,
      <NumberField
        key="npc_price"
        label="NPC Price"
        value={item.npc_price}
        max={MAX_UINT32}
        onChange={(value) => updateItem(index, 'npc_price', value)}
      />,
    ];
  }

  if (itemFileType === 'it1') {
    return [
      <NumberField
        key="npc_price"
        label="NPC Price"
        value={item.npc_price}
        max={MAX_UINT32}
        onChange={(value) => updateItem(index, 'npc_price', value)}
      />,
      <NumberField
        key="required_level"
        label="Req Level"
        value={item.required_level}
        onChange={(value) => updateItem(index, 'required_level', value)}
      />,
      <NumberField
        key="attribute"
        label="Attribute"
        value={item.attribute}
        onChange={(value) => updateItem(index, 'attribute', value)}
      />,
      <NumberField
        key="blue_option"
        label="Blue"
        value={item.blue_option}
        onChange={(value) => updateItem(index, 'blue_option', value)}
      />,
      <NumberField
        key="red_option"
        label="Red"
        value={item.red_option}
        onChange={(value) => updateItem(index, 'red_option', value)}
      />,
      <NumberField
        key="grey_option"
        label="Grey"
        value={item.grey_option}
        onChange={(value) => updateItem(index, 'grey_option', value)}
      />,
    ];
  }

  if (itemFileType === 'it2') {
    return [
      <NumberField
        key="npc_price"
        label="NPC Price"
        value={item.npc_price}
        max={MAX_UINT32}
        onChange={(value) => updateItem(index, 'npc_price', value)}
      />,
      <NumberField
        key="class"
        label="Class"
        value={item.class}
        onChange={(value) => updateItem(index, 'class', value)}
      />,
      <NumberField
        key="required_level"
        label="Req Level"
        value={item.required_level}
        onChange={(value) => updateItem(index, 'required_level', value)}
      />,
      <NumberField
        key="skill_level"
        label="Skill Level"
        value={item.skill_level}
        onChange={(value) => updateItem(index, 'skill_level', value)}
      />,
    ];
  }

  if (itemFileType === 'it3') {
    return [
      <NumberField
        key="npc_price"
        label="NPC Price"
        value={item.npc_price}
        max={MAX_UINT32}
        onChange={(value) => updateItem(index, 'npc_price', value)}
      />,
    ];
  }

  return [];
}

function getItemEditRowHeight(itemFileType: ItemFileAPIData['item_file_type']) {
  if (itemFileType === 'it1') {
    return IT1_ITEM_EDIT_ROW_HEIGHT;
  }

  if (itemFileType === 'it2') {
    return IT2_ITEM_EDIT_ROW_HEIGHT;
  }

  return DEFAULT_ITEM_EDIT_ROW_HEIGHT;
}

function NumberField({
  label,
  value,
  max = MAX_UINT16,
  onChange,
}: {
  label: string;
  value: number | undefined;
  max?: number;
  onChange: (value: number) => void;
}) {
  return (
    <label className="grid gap-1 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <NumberInput
        value={value}
        max={max}
        ariaLabel={label}
        onChange={onChange}
      />
    </label>
  );
}

function NumberInput({
  value,
  max,
  ariaLabel,
  onChange,
}: {
  value: number | undefined;
  max: number;
  ariaLabel: string;
  onChange: (value: number) => void;
}) {
  return (
    <Input
      type="number"
      min={0}
      max={max}
      value={value ?? 0}
      onChange={(event) => onChange(clampNumber(event.target.value, max))}
      aria-label={ariaLabel}
      className="min-w-24 text-right font-mono"
    />
  );
}

function createEmptyLevels(firstLevel: number, count: number) {
  return Array.from({ length: count }, (_, index) => ({
    level: firstLevel + index,
    additional_attribute: 0,
    strength: 0,
    dexterity: 0,
    intelligence: 0,
    attribute: 0,
    attribute_range: 0,
    blue_option: 0,
    red_option: 0,
    grey_option: 0,
  }));
}

function createIT0ExLevels(baseItem: ItemFileBaseItemAPIData) {
  return (baseItem.levels ?? createEmptyLevels(11, 5)).map((level) => ({
    ...level,
  }));
}

function filterIndexedItems(
  items: ItemFileItemAPIData[],
  normalizedQuery: string,
) {
  if (normalizedQuery === '') {
    return null;
  }

  return items
    .map((item, index) => ({ item, index }))
    .filter(({ item }) => {
      const values = [item.row, item.item_code, item.name];
      return values.some((value) =>
        String(value ?? '')
          .toLowerCase()
          .includes(normalizedQuery),
      );
    });
}

function filterBaseItems(
  items: ItemFileBaseItemAPIData[],
  normalizedQuery: string,
) {
  if (normalizedQuery === '') {
    return items;
  }

  return items.filter((item) => {
    const values = [item.row, item.item_code, item.name];
    return values.some((value) =>
      String(value ?? '')
        .toLowerCase()
        .includes(normalizedQuery),
    );
  });
}

function findDuplicateRows(items: ItemFileItemAPIData[]) {
  const seenRows = new Set<number>();
  const duplicateRows = new Set<number>();
  for (const item of items) {
    if (item.row === undefined) {
      continue;
    }

    if (seenRows.has(item.row)) {
      duplicateRows.add(item.row);
    }

    seenRows.add(item.row);
  }

  return Array.from(duplicateRows);
}

function clampNumber(value: string, max: number) {
  const parsedValue = Number(value);
  if (!Number.isFinite(parsedValue)) {
    return 0;
  }

  return Math.min(Math.max(Math.trunc(parsedValue), 0), max);
}

function formatValue(value: number | string | undefined) {
  if (value === undefined || value === '') {
    return '-';
  }

  return value;
}
