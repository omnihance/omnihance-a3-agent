import { useMemo, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Loader2, Save } from 'lucide-react';
import { toast } from 'sonner';
import { APIError, updateZoneDataFile } from '@/lib/api';
import type { ZoneDataField, ZoneDataFile, ZoneDataOperation } from '@/lib/api';
import { queryKeys } from '@/constants';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
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
import { ZoneMapCanvas, fieldsForScope } from './zone-data-file-view';

const pageSize = 25;

export function ZoneDataFileEdit({
  filePath,
  defaultData,
}: {
  filePath: string;
  defaultData: ZoneDataFile;
}) {
  const [operations, setOperations] = useState<ZoneDataOperation[]>([]);
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: () =>
      updateZoneDataFile(
        { path: filePath },
        { source_hash: defaultData.source_hash, operations },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.zoneDataFile(filePath),
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.fileTree(filePath) });
      queryClient.invalidateQueries({
        queryKey: queryKeys.revisionSummary(filePath),
      });
      toast.success('ZoneData file updated');
      setOperations([]);
    },
    onError: (error) => {
      toast.error(
        error instanceof APIError
          ? error.getErrorMessage()
          : error instanceof Error
            ? error.message
            : 'Failed to update ZoneData file',
      );
    },
  });

  const setValue = (
    scope: ZoneDataField['scope'],
    row: number,
    field: string,
    value: number | string | boolean,
  ) => {
    setOperations((current) => {
      const next = current.filter(
        (operation) =>
          !(
            operation.scope === scope &&
            operation.row === row &&
            operation.field === field
          ),
      );
      next.push({ scope, row, field, value });
      return next;
    });
  };

  const valueFor = (
    scope: ZoneDataField['scope'],
    row: number,
    field: string,
    original: number | string | boolean,
  ) =>
    operations.find(
      (operation) =>
        operation.scope === scope &&
        operation.row === row &&
        operation.field === field,
    )?.value ?? original;

  return (
    <div className="space-y-6">
      {defaultData.map ? (
        <ZoneMapEdit
          data={defaultData}
          operations={operations}
          setValue={setValue}
          valueFor={valueFor}
        />
      ) : (
        <ZoneDataTableEdit
          data={defaultData}
          setValue={setValue}
          valueFor={valueFor}
        />
      )}
      <div className="sticky bottom-4 flex items-center justify-between gap-4 rounded-lg border bg-background/95 p-4 shadow-lg backdrop-blur">
        <div>
          <div className="font-medium">{operations.length} pending changes</div>
          <div className="text-sm text-muted-foreground">
            Opaque bytes stay read-only and are checked before saving.
          </div>
        </div>
        <Button
          onClick={() => mutation.mutate()}
          disabled={operations.length === 0 || mutation.isPending}
        >
          {mutation.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Save className="mr-2 h-4 w-4" />
          )}
          Save changes
        </Button>
      </div>
    </div>
  );
}

function ZoneDataTableEdit({
  data,
  setValue,
  valueFor,
}: {
  data: ZoneDataFile;
  setValue: SetValue;
  valueFor: ValueFor;
}) {
  const [page, setPage] = useState(0);
  const fields = fieldsForScope(data, 'row');
  const allRows = data.rows ?? [];
  const pageCount = Math.max(1, Math.ceil(allRows.length / pageSize));
  const rows = allRows.slice(page * pageSize, (page + 1) * pageSize);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Decoded fields</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Row</TableHead>
                {fields.map((field) => (
                  <TableHead key={field.key} className="whitespace-nowrap">
                    {field.label}
                  </TableHead>
                ))}
                <TableHead>Opaque bytes</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.index}>
                  <TableCell className="font-mono text-xs">
                    {row.index}
                  </TableCell>
                  {fields.map((field) => (
                    <TableCell key={field.key} className="min-w-36">
                      <FieldInput
                        field={field}
                        value={valueFor(
                          'row',
                          row.index,
                          field.key,
                          row.values[field.key] ?? '',
                        )}
                        onChange={(value) =>
                          setValue('row', row.index, field.key, value)
                        }
                      />
                    </TableCell>
                  ))}
                  <TableCell>
                    <code className="block max-w-44 truncate text-xs text-muted-foreground">
                      {row.opaque_bytes}
                    </code>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
        <div className="mt-4 flex items-center justify-between">
          <span className="text-sm text-muted-foreground">
            Page {page + 1} of {pageCount}
          </span>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((value) => Math.max(0, value - 1))}
              disabled={page === 0}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                setPage((value) => Math.min(pageCount - 1, value + 1))
              }
              disabled={page >= pageCount - 1}
            >
              Next
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function ZoneMapEdit({
  data,
  operations,
  setValue,
  valueFor,
}: ZoneMapEditProps) {
  const map = data.map!;
  const [selectedCell, setSelectedCell] = useState(0);
  const [zoom, setZoom] = useState(2);
  const patchedCells = useMemo(() => {
    const cells = [...map.cells];
    for (const operation of operations) {
      if (operation.scope !== 'cell') {
        continue;
      }

      const raw = cells[operation.row] ?? 0;
      if (operation.field === 'can_move') {
        cells[operation.row] = operation.value ? raw | 1 : raw & ~1;
      }
      if (operation.field === 'pk_level') {
        cells[operation.row] =
          (raw & ~(3 << 15)) | (Number(operation.value) << 15);
      }
      if (operation.field === 'warp_index') {
        cells[operation.row] =
          (raw & ~(15 << 11)) | (Number(operation.value) << 11);
      }
    }
    return cells;
  }, [map.cells, operations]);
  const raw = patchedCells[selectedCell] ?? 0;
  const x = selectedCell % map.width;
  const y = Math.floor(selectedCell / map.width);

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <Card>
        <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <Label htmlFor="zone-map-name">Map name</Label>
            <Input
              id="zone-map-name"
              value={String(valueFor('map', 0, 'name', map.name))}
              maxLength={2}
              onChange={(event) =>
                setValue('map', 0, 'name', event.target.value)
              }
              className="mt-2 w-28"
            />
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setZoom((value) => Math.max(1, value - 1))}
              aria-label="Zoom out"
            >
              −
            </Button>
            <Badge variant="secondary">{zoom}×</Badge>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setZoom((value) => Math.min(4, value + 1))}
              aria-label="Zoom in"
            >
              +
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="max-h-[70vh] overflow-auto rounded-md border bg-muted/40 p-3">
            <ZoneMapCanvas
              cells={patchedCells}
              width={map.width}
              height={map.height}
              selectedCell={selectedCell}
              zoom={zoom}
              onSelect={setSelectedCell}
            />
          </div>
        </CardContent>
      </Card>
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>
              Cell {x}, {y}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-3">
              <Checkbox
                id="cell-can-move"
                checked={(raw & 1) !== 0}
                onCheckedChange={(checked) =>
                  setValue('cell', selectedCell, 'can_move', checked === true)
                }
              />
              <Label htmlFor="cell-can-move">Can move</Label>
            </div>
            <LabeledNumber
              id="cell-pk-level"
              label="PK level"
              value={(raw >>> 15) & 3}
              min={0}
              max={3}
              onChange={(value) =>
                setValue('cell', selectedCell, 'pk_level', value)
              }
            />
            <LabeledNumber
              id="cell-warp-index"
              label="Warp index (15 means none)"
              value={(raw >>> 11) & 15}
              min={0}
              max={15}
              onChange={(value) =>
                setValue('cell', selectedCell, 'warp_index', value)
              }
            />
            <div>
              <Label>Raw bytes</Label>
              <div className="mt-2 rounded-md border bg-muted px-3 py-2 font-mono text-sm text-muted-foreground">
                0x{raw.toString(16).padStart(8, '0')}
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Warps</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {map.warps.map((warp) => (
              <div
                key={warp.index}
                className="grid grid-cols-3 gap-2 rounded-md border p-3"
              >
                {fieldsForScope(data, 'warp').map((field) => (
                  <div key={field.key}>
                    <Label
                      htmlFor={`warp-${warp.index}-${field.key}`}
                      className="text-xs"
                    >
                      {field.label}
                    </Label>
                    <Input
                      id={`warp-${warp.index}-${field.key}`}
                      type="number"
                      min={field.min}
                      max={field.max}
                      value={Number(
                        valueFor(
                          'warp',
                          warp.index,
                          field.key,
                          warp.values[field.key] ?? 0,
                        ),
                      )}
                      onChange={(event) =>
                        setValue(
                          'warp',
                          warp.index,
                          field.key,
                          Number(event.target.value),
                        )
                      }
                    />
                  </div>
                ))}
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function FieldInput({
  field,
  value,
  onChange,
}: {
  field: ZoneDataField;
  value: number | string | boolean;
  onChange: (value: number | string | boolean) => void;
}) {
  if (field.type === 'boolean') {
    return (
      <Checkbox
        checked={Boolean(value)}
        onCheckedChange={(checked) => onChange(checked === true)}
        aria-label={field.label}
      />
    );
  }

  return (
    <Input
      type={field.type === 'integer' ? 'number' : 'text'}
      min={field.min}
      max={field.max}
      value={String(value)}
      aria-label={field.label}
      onChange={(event) =>
        onChange(
          field.type === 'integer'
            ? Number(event.target.value)
            : event.target.value,
        )
      }
    />
  );
}

function LabeledNumber({
  id,
  label,
  value,
  min,
  max,
  onChange,
}: LabeledNumberProps) {
  return (
    <div>
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="number"
        value={value}
        min={min}
        max={max}
        onChange={(event) => onChange(Number(event.target.value))}
        className="mt-2"
      />
    </div>
  );
}

type SetValue = (
  scope: ZoneDataField['scope'],
  row: number,
  field: string,
  value: number | string | boolean,
) => void;

type ValueFor = (
  scope: ZoneDataField['scope'],
  row: number,
  field: string,
  original: number | string | boolean,
) => number | string | boolean;

interface ZoneMapEditProps {
  data: ZoneDataFile;
  operations: ZoneDataOperation[];
  setValue: SetValue;
  valueFor: ValueFor;
}

interface LabeledNumberProps {
  id: string;
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (value: number) => void;
}
