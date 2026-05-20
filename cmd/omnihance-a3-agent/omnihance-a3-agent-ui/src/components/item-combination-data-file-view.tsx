import { useDeferredValue, useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Hash, Hammer, Package, Search } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { useVirtualRows } from '@/hooks/use-virtual-rows';
import {
  getItems,
  type GameClientDataResponse,
  type ItemCombinationDataFileAPIData,
} from '@/lib/api';
import { queryKeys } from '@/constants';

const EMPTY_ITEM_CODE = 0;
const FORMULA_ROW_HEIGHT = 112;

interface ItemCombinationDataFileViewProps {
  data: ItemCombinationDataFileAPIData;
}

export function ItemCombinationDataFileView({
  data,
}: ItemCombinationDataFileViewProps) {
  const { data: items } = useQuery({
    queryKey: queryKeys.items,
    queryFn: () => getItems(),
  });

  const [searchQuery, setSearchQuery] = useState('');
  const deferredSearchQuery = useDeferredValue(searchQuery);
  const itemLookup = useMemo(() => createItemLookup(items), [items]);
  const normalizedSearchQuery = deferredSearchQuery.trim().toLowerCase();
  const filteredFormulas = useMemo(
    () =>
      data.formulas
        .map((formula, formulaIndex) => ({ formula, formulaIndex }))
        .filter(({ formula }) =>
          formulaMatchesOutcomeSearch(
            formula,
            normalizedSearchQuery,
            itemLookup,
          ),
        ),
    [data.formulas, itemLookup, normalizedSearchQuery],
  );
  const { containerRef, onScroll, totalHeight, virtualRows } = useVirtualRows({
    count: filteredFormulas.length,
    rowHeight: FORMULA_ROW_HEIGHT,
  });
  const hasSearchQuery = normalizedSearchQuery.length > 0;
  const formulaCountLabel = hasSearchQuery
    ? `${filteredFormulas.length}/${data.formulas.length}`
    : String(data.formulas.length);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }

    container.scrollTop = 0;
  }, [containerRef, normalizedSearchQuery]);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <CardTitle className="flex items-center gap-2">
              <Hammer className="h-5 w-5" />
              Formulas ({formulaCountLabel})
            </CardTitle>
            {data.formulas.length > 0 && (
              <div className="relative w-full sm:max-w-sm">
                <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  type="search"
                  className="pl-9"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search outcome code or name"
                  aria-label="Search formulas by outcome item code or name"
                />
              </div>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {data.formulas.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              No formulas configured
            </div>
          ) : filteredFormulas.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              No matching formulas
            </div>
          ) : (
            <div className="overflow-x-auto">
              <div className="min-w-[980px]">
                <div className="grid grid-cols-[4rem_16rem_7rem_1fr] divide-x border-b bg-card text-sm font-medium text-muted-foreground">
                  <div className="px-3 py-2">#</div>
                  <div className="px-3 py-2">Outcome</div>
                  <div className="px-3 py-2 text-right">Success</div>
                  <div className="px-3 py-2">Ingredients</div>
                </div>
                <div
                  ref={containerRef}
                  onScroll={onScroll}
                  className="relative h-[min(70vh,640px)] overflow-y-auto"
                  role="rowgroup"
                  aria-label="Item combination formulas"
                >
                  <div className="relative" style={{ height: totalHeight }}>
                    {virtualRows.map(({ index, top }) => {
                      const { formula, formulaIndex } = filteredFormulas[index];
                      const visibleIngredients = formula.ingredients
                        .map((itemCode, ingredientIndex) => ({
                          ingredientIndex,
                          itemCode,
                        }))
                        .filter(({ itemCode }) => itemCode !== EMPTY_ITEM_CODE);

                      return (
                        <div
                          key={`${formula.outcome}-${formulaIndex}`}
                          className="absolute left-0 right-0 grid grid-cols-[4rem_16rem_7rem_1fr] items-start divide-x border-b"
                          style={{
                            height: FORMULA_ROW_HEIGHT,
                            transform: `translateY(${top}px)`,
                          }}
                          role="row"
                          aria-rowindex={formulaIndex + 1}
                        >
                          <div className="px-3 py-3 font-medium">
                            {formulaIndex + 1}
                          </div>
                          <div className="flex min-w-0 items-center gap-2 px-3 py-3">
                            <Package className="h-3 w-3 shrink-0 text-muted-foreground" />
                            <span className="truncate">
                              {formatItemDisplay(formula.outcome, itemLookup)}
                            </span>
                          </div>
                          <div className="px-3 py-3 text-right">
                            {formula.success_rate}%
                          </div>
                          <div className="px-3 py-3">
                            {visibleIngredients.length > 0 ? (
                              <div className="grid grid-cols-5 gap-1.5">
                                {visibleIngredients.map(
                                  ({ itemCode, ingredientIndex }) => (
                                    <Badge
                                      key={`${formulaIndex}-${ingredientIndex}-${itemCode}`}
                                      variant="secondary"
                                      className="justify-start gap-1 truncate"
                                    >
                                      <Hash className="h-3 w-3 shrink-0" />
                                      <span className="truncate">
                                        {ingredientIndex + 1}.{' '}
                                        {formatItemDisplay(
                                          itemCode,
                                          itemLookup,
                                        )}
                                      </span>
                                    </Badge>
                                  ),
                                )}
                              </div>
                            ) : (
                              <span className="text-muted-foreground">-</span>
                            )}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function formulaMatchesOutcomeSearch(
  formula: ItemCombinationDataFileAPIData['formulas'][number],
  query: string,
  itemLookup: Map<number, string>,
): boolean {
  if (query === '') {
    return true;
  }

  if (String(formula.outcome).includes(query)) {
    return true;
  }

  const itemName =
    formula.outcome === EMPTY_ITEM_CODE
      ? 'empty'
      : itemLookup.get(formula.outcome)?.toLowerCase();

  return itemName?.includes(query) ?? false;
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

function formatItemDisplay(
  itemCode: number,
  itemLookup: Map<number, string>,
): string {
  if (itemCode === EMPTY_ITEM_CODE) {
    return 'Empty';
  }

  const itemName = itemLookup.get(itemCode);
  if (itemName) {
    return `${itemName} (${itemCode})`;
  }

  return String(itemCode);
}
