import { useDeferredValue, useEffect, useMemo, useState } from 'react';
import { ListTree, PackageSearch } from 'lucide-react';
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useVirtualRows } from '@/hooks/use-virtual-rows';
import type { ItemFileAPIData, ItemFileItemAPIData } from '@/lib/api';

const ITEM_ROW_HEIGHT = 156;

interface ItemFileViewProps {
  data: ItemFileAPIData;
}

export function ItemFileView({ data }: ItemFileViewProps) {
  const [query, setQuery] = useState('');
  const [levelItem, setLevelItem] = useState<ItemFileItemAPIData | null>(null);
  const deferredQuery = useDeferredValue(query);
  const normalizedQuery = deferredQuery.trim().toLowerCase();
  const filteredItems = useMemo(
    () => filterItems(data.items, normalizedQuery),
    [data.items, normalizedQuery],
  );
  const { containerRef, onScroll, resetScrollTop, totalHeight, virtualRows } =
    useVirtualRows({
      count: filteredItems.length,
      rowHeight: ITEM_ROW_HEIGHT,
      overscan: 8,
    });
  const countLabel =
    normalizedQuery.length > 0
      ? `${filteredItems.length}/${data.items.length}`
      : String(data.items.length);

  useEffect(() => {
    resetScrollTop();
  }, [normalizedQuery, resetScrollTop]);

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <CardTitle className="flex items-center gap-2">
              <PackageSearch className="h-5 w-5" />
              Item Records ({countLabel})
            </CardTitle>
            <Input
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search row, code, or name"
              aria-label="Search item records"
              className="w-full lg:max-w-sm"
            />
          </div>
        </CardHeader>
        <CardContent>
          {data.items.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              No item records configured
            </div>
          ) : filteredItems.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              No item records found
            </div>
          ) : (
            <div className="overflow-x-auto rounded-md border">
              <div className="min-w-[1040px]">
                <div className="grid grid-cols-[5rem_18rem_8rem_6rem_1fr_7rem] divide-x border-b bg-card text-sm font-medium text-muted-foreground">
                  <div className="px-3 py-2">Row</div>
                  <div className="px-3 py-2">Item</div>
                  <div className="px-3 py-2 text-right">Code</div>
                  <div className="px-3 py-2 text-right">Type</div>
                  <div className="px-3 py-2">Known Fields</div>
                  <div className="px-3 py-2 text-right">Levels</div>
                </div>
                <div
                  ref={containerRef}
                  onScroll={onScroll}
                  className="relative h-[min(70vh,680px)] overflow-y-auto"
                  role="rowgroup"
                  aria-label="Item records"
                >
                  <div className="relative" style={{ height: totalHeight }}>
                    {virtualRows.map(({ index, top }) => {
                      const item = filteredItems[index];

                      return (
                        <ItemViewRow
                          key={`${item.row ?? index}-${item.item_code ?? index}`}
                          item={item}
                          itemFileType={data.item_file_type}
                          top={top}
                          onViewLevels={() => setLevelItem(item)}
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

      <LevelViewDialog
        item={levelItem}
        open={levelItem !== null}
        onOpenChange={(open) => {
          if (!open) {
            setLevelItem(null);
          }
        }}
      />
    </>
  );
}

function ItemViewRow({
  item,
  itemFileType,
  top,
  onViewLevels,
}: {
  item: ItemFileItemAPIData;
  itemFileType: ItemFileAPIData['item_file_type'];
  top: number;
  onViewLevels: () => void;
}) {
  const hasLevels = (item.levels?.length ?? 0) > 0;

  return (
    <div
      className="absolute left-0 right-0 grid grid-cols-[5rem_18rem_8rem_6rem_1fr_7rem] items-start divide-x border-b"
      style={{
        height: ITEM_ROW_HEIGHT,
        transform: `translateY(${top}px)`,
      }}
      role="row"
    >
      <div className="px-3 py-3 font-mono text-sm">{formatValue(item.row)}</div>
      <div className="min-w-0 px-3 py-3">
        <div className="truncate font-medium">{item.name || '-'}</div>
        {item.item_code_base !== undefined && (
          <div className="text-xs text-muted-foreground">
            Base {item.item_code_base}
          </div>
        )}
      </div>
      <div className="px-3 py-3 text-right font-mono text-sm">
        {formatValue(item.item_code)}
      </div>
      <div className="px-3 py-3 text-right font-mono text-sm">
        {formatValue(item.type)}
      </div>
      <div className="px-3 py-3">
        <div className="grid gap-2 text-sm sm:grid-cols-2 xl:grid-cols-3">
          {knownFieldEntries(item, itemFileType).map(([label, value]) => (
            <div
              key={label}
              className="flex justify-between gap-3 rounded-md border px-2 py-1"
            >
              <span className="text-muted-foreground">{label}</span>
              <span className="font-mono">{formatValue(value)}</span>
            </div>
          ))}
        </div>
      </div>
      <div className="px-3 py-3 text-right">
        {hasLevels ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onViewLevels}
            aria-label={`View levels for item row ${formatValue(item.row)}`}
          >
            <ListTree className="mr-2 h-4 w-4" />
            {item.levels?.length}
          </Button>
        ) : (
          <span className="text-muted-foreground">-</span>
        )}
      </div>
    </div>
  );
}

function LevelViewDialog({
  item,
  open,
  onOpenChange,
}: {
  item: ItemFileItemAPIData | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle>
            Levels for {item?.name || `Row ${formatValue(item?.row)}`}
          </DialogTitle>
        </DialogHeader>
        <div className="max-h-[70vh] overflow-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="text-right">Level</TableHead>
                <TableHead className="text-right">Add Attr</TableHead>
                <TableHead className="text-right">Str</TableHead>
                <TableHead className="text-right">Dex</TableHead>
                <TableHead className="text-right">Int</TableHead>
                <TableHead className="text-right">Attr</TableHead>
                <TableHead className="text-right">Range</TableHead>
                <TableHead className="text-right">Blue</TableHead>
                <TableHead className="text-right">Red</TableHead>
                <TableHead className="text-right">Grey</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(item?.levels ?? []).map((level, index) => (
                <TableRow key={`${item?.row ?? 'row'}-level-${index}`}>
                  <TableCell className="text-right font-mono">
                    {formatValue(level.level)}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatValue(level.additional_attribute)}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatValue(level.strength)}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatValue(level.dexterity)}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatValue(level.intelligence)}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatValue(level.attribute)}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatValue(level.attribute_range)}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatValue(level.blue_option)}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatValue(level.red_option)}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {formatValue(level.grey_option)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function knownFieldEntries(
  item: ItemFileItemAPIData,
  itemFileType: ItemFileAPIData['item_file_type'],
): Array<[string, number | undefined]> {
  if (itemFileType === 'it0') {
    return [
      ['Slot', item.slot],
      ['NPC Price', item.npc_price],
    ];
  }

  if (itemFileType === 'it1') {
    return [
      ['NPC Price', item.npc_price],
      ['Req Level', item.required_level],
      ['Attribute', item.attribute],
      ['Blue', item.blue_option],
      ['Red', item.red_option],
      ['Grey', item.grey_option],
    ];
  }

  if (itemFileType === 'it2') {
    return [
      ['NPC Price', item.npc_price],
      ['Class', item.class],
      ['Req Level', item.required_level],
      ['Skill Level', item.skill_level],
    ];
  }

  if (itemFileType === 'it3') {
    return [['NPC Price', item.npc_price]];
  }

  return [
    ['Slot', item.slot],
    ['Base', item.item_code_base],
  ];
}

function filterItems(items: ItemFileItemAPIData[], normalizedQuery: string) {
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

function formatValue(value: number | string | undefined) {
  if (value === undefined || value === '') {
    return '-';
  }

  return value;
}
