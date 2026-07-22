import {
  memo,
  useDeferredValue,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { Badge } from '@/components/ui/badge';
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
import type { ZoneDataFile, ZoneDataField } from '@/lib/api';

const pageSize = 50;

export function ZoneDataFileView({ data }: { data: ZoneDataFile }) {
  if (data.map) {
    return <ZoneMapView data={data} />;
  }

  return <ZoneDataTable data={data} />;
}

function ZoneDataTable({ data }: { data: ZoneDataFile }) {
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(0);
  const deferredQuery = useDeferredValue(query.toLowerCase());
  const fields = data.schema.filter((field) => field.scope === 'row');
  const filteredRows = useMemo(() => {
    if (!deferredQuery) {
      return data.rows ?? [];
    }

    return (data.rows ?? []).filter((row) =>
      JSON.stringify(row.values).toLowerCase().includes(deferredQuery),
    );
  }, [data.rows, deferredQuery]);
  const pageCount = Math.max(1, Math.ceil(filteredRows.length / pageSize));
  const currentPage = Math.min(page, pageCount - 1);
  const rows = filteredRows.slice(
    currentPage * pageSize,
    (currentPage + 1) * pageSize,
  );

  return (
    <Card>
      <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <CardTitle>{formatName(data.format)}</CardTitle>
          <p className="mt-1 text-sm text-muted-foreground">
            {filteredRows.length.toLocaleString()} decoded rows
          </p>
        </div>
        <Input
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setPage(0);
          }}
          aria-label="Search decoded rows"
          placeholder="Search rows"
          className="sm:max-w-xs"
        />
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">Row</TableHead>
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
                    <TableCell key={field.key} className="whitespace-nowrap">
                      {String(row.values[field.key] ?? '')}
                    </TableCell>
                  ))}
                  <TableCell>
                    <code className="block max-w-48 truncate text-xs text-muted-foreground">
                      {row.opaque_bytes}
                    </code>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
        <div className="mt-4 flex items-center justify-between gap-3">
          <span className="text-sm text-muted-foreground">
            Page {currentPage + 1} of {pageCount}
          </span>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((value) => Math.max(0, value - 1))}
              disabled={currentPage === 0}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                setPage((value) => Math.min(pageCount - 1, value + 1))
              }
              disabled={currentPage >= pageCount - 1}
            >
              Next
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function ZoneMapView({ data }: { data: ZoneDataFile }) {
  const map = data.map!;
  const [selectedCell, setSelectedCell] = useState(0);
  const [zoom, setZoom] = useState(2);
  const raw = map.cells[selectedCell] ?? 0;
  const x = selectedCell % map.width;
  const y = Math.floor(selectedCell / map.width);
  const warpIndex = (raw >>> 11) & 0x0f;

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_20rem]">
      <Card>
        <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <CardTitle>{map.name || 'Zone map'}</CardTitle>
            <p className="mt-1 text-sm text-muted-foreground">
              {map.width}×{map.height} cells · {map.warps.length} warps
            </p>
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
              cells={map.cells}
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
            <CardTitle>Selected cell</CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-4 text-sm">
            <MapValue label="X" value={x} />
            <MapValue label="Y" value={y} />
            <MapValue
              label="Raw"
              value={`0x${raw.toString(16).padStart(8, '0')}`}
            />
            <MapValue label="Can move" value={(raw & 1) !== 0 ? 'Yes' : 'No'} />
            <MapValue label="PK level" value={(raw >>> 15) & 0x03} />
            <MapValue
              label="Warp"
              value={warpIndex === 15 ? 'None' : warpIndex}
            />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Warp overlay</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            {map.warps.length === 0 && (
              <p className="text-muted-foreground">No warp rows.</p>
            )}
            {map.warps.map((warp) => (
              <div
                key={warp.index}
                className="flex items-center justify-between gap-3 rounded-md border px-3 py-2"
              >
                <span>Warp {warp.index}</span>
                <span className="text-muted-foreground">
                  Map {String(warp.values.map_id)} · Cell{' '}
                  {String(warp.values.cell)}
                </span>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export const ZoneMapCanvas = memo(function ZoneMapCanvas({
  cells,
  width,
  height,
  selectedCell,
  zoom,
  onSelect,
}: {
  cells: number[];
  width: number;
  height: number;
  selectedCell: number;
  zoom: number;
  onSelect: (index: number) => void;
}) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    const context = canvas?.getContext('2d');
    if (!canvas || !context) {
      return;
    }

    const image = context.createImageData(width, height);
    for (let index = 0; index < cells.length; index += 1) {
      const raw = cells[index] ?? 0;
      const warp = ((raw >>> 11) & 0x0f) !== 15;
      const movable = (raw & 1) !== 0;
      const offset = index * 4;
      image.data[offset] = warp ? 245 : movable ? 83 : 30;
      image.data[offset + 1] = warp ? 158 : movable ? 178 : 41;
      image.data[offset + 2] = warp ? 11 : movable ? 109 : 59;
      image.data[offset + 3] = 255;
    }
    context.putImageData(image, 0, 0);
    context.strokeStyle = '#ffffff';
    context.lineWidth = 1;
    context.strokeRect(
      selectedCell % width,
      Math.floor(selectedCell / width),
      1,
      1,
    );
  }, [cells, height, selectedCell, width]);

  const selectFromPointer = (clientX: number, clientY: number) => {
    const canvas = canvasRef.current;
    if (!canvas) {
      return;
    }

    const rect = canvas.getBoundingClientRect();
    const x = Math.min(
      width - 1,
      Math.max(0, Math.floor(((clientX - rect.left) / rect.width) * width)),
    );
    const y = Math.min(
      height - 1,
      Math.max(0, Math.floor(((clientY - rect.top) / rect.height) * height)),
    );
    onSelect(y * width + x);
  };

  return (
    <canvas
      ref={canvasRef}
      width={width}
      height={height}
      tabIndex={0}
      role="grid"
      aria-label="Zone map cell grid. Use arrow keys to move the selected cell."
      className="block cursor-crosshair bg-background [image-rendering:pixelated] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      style={{ width: width * zoom, height: height * zoom }}
      onClick={(event) => selectFromPointer(event.clientX, event.clientY)}
      onKeyDown={(event) => {
        const delta =
          event.key === 'ArrowLeft'
            ? -1
            : event.key === 'ArrowRight'
              ? 1
              : event.key === 'ArrowUp'
                ? -width
                : event.key === 'ArrowDown'
                  ? width
                  : 0;
        if (delta !== 0) {
          event.preventDefault();
          onSelect(
            Math.min(cells.length - 1, Math.max(0, selectedCell + delta)),
          );
        }
      }}
    />
  );
});

function MapValue({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <div className="text-muted-foreground">{label}</div>
      <div className="mt-1 font-mono font-medium">{value}</div>
    </div>
  );
}

function formatName(value: string) {
  return value
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export function fieldsForScope(
  data: ZoneDataFile,
  scope: ZoneDataField['scope'],
) {
  return data.schema.filter((field) => field.scope === scope);
}
