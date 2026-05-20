import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Hash, Package, Percent, Tags } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { getItems, type DropFileAPIData } from '@/lib/api';
import { queryKeys } from '@/constants';

const EMPTY_ITEM_CODE = 0xffff;
const ITEM_ID_MASK = 0x3fff;

interface DropFileViewProps {
  data: DropFileAPIData;
}

export function DropFileView({ data }: DropFileViewProps) {
  const { data: items } = useQuery({
    queryKey: queryKeys.items,
    queryFn: () => getItems(),
  });

  const itemLookup = useMemo(() => {
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
  }, [items]);

  const getItemDisplay = (itemCode: number): string => {
    if (itemCode === EMPTY_ITEM_CODE) {
      return 'Empty slot';
    }

    const baseCode = itemCode & ITEM_ID_MASK;
    const itemName = itemLookup.get(baseCode);
    if (itemName) {
      return `${itemName} (${itemCode})`;
    }

    return String(itemCode);
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Package className="h-5 w-5" />
            Drops ({data.drops.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {data.drops.length > 0 ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>#</TableHead>
                    <TableHead>Item</TableHead>
                    <TableHead className="text-right">Item Code</TableHead>
                    <TableHead className="text-right">Drop Rate</TableHead>
                    <TableHead className="text-right">Drop Group</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.drops.map((drop, index) => (
                    <TableRow key={`${drop.item_code}-${index}`}>
                      <TableCell className="font-medium">{index + 1}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Package className="h-3 w-3 text-muted-foreground" />
                          {getItemDisplay(drop.item_code)}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Hash className="h-3 w-3 text-muted-foreground" />
                          {drop.item_code}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Percent className="h-3 w-3 text-muted-foreground" />
                          {drop.drop_rate}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Tags className="h-3 w-3 text-muted-foreground" />
                          {drop.drop_group}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="py-8 text-center text-muted-foreground">
              No drops configured
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
