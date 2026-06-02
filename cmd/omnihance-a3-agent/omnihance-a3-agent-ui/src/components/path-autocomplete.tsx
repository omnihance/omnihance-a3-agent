import { useState } from 'react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { FileArchive, Folder, Loader2 } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { queryKeys } from '@/constants';
import { searchBackupPaths } from '@/lib/api';
import { useDebouncedValue } from '@/lib/util';

const pathSearchDebounceMs = 250;

export function PathAutocomplete({
  id,
  label,
  value,
  kind,
  onChange,
  disabled = false,
}: {
  id: string;
  label: string;
  value: string;
  kind: 'input' | 'directory';
  onChange: (value: string) => void;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const debouncedValue = useDebouncedValue(value, pathSearchDebounceMs);
  const waitingForDebounce = value !== debouncedValue;
  const { data: results = [], isFetching } = useQuery({
    queryKey: queryKeys.backupPathSearch(debouncedValue, kind),
    queryFn: () => searchBackupPaths({ query: debouncedValue, kind }),
    enabled: open && !disabled,
    placeholderData: keepPreviousData,
    staleTime: 5000,
  });
  const showLoading = waitingForDebounce || isFetching;
  const showDropdown =
    open &&
    !disabled &&
    (showLoading || results.length > 0 || value.trim() !== '');
  const suggestionsID = `${id}-suggestions`;

  return (
    <div
      className="relative space-y-2"
      onFocus={() => setOpen(true)}
      onBlur={(event) => {
        if (event.currentTarget.contains(event.relatedTarget)) {
          return;
        }

        setOpen(false);
      }}
    >
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        value={value}
        onChange={(event) => {
          setOpen(true);
          onChange(event.target.value);
        }}
        autoComplete="off"
        disabled={disabled}
        aria-autocomplete="list"
        aria-controls={showDropdown ? suggestionsID : undefined}
        aria-expanded={showDropdown}
        aria-haspopup="listbox"
      />
      {showDropdown && (
        <div
          id={suggestionsID}
          role="listbox"
          aria-label={`${label} path suggestions`}
          className="absolute z-[70] max-h-56 w-full overflow-auto rounded-md border bg-popover p-1 shadow-md"
        >
          {showLoading && results.length === 0 && (
            <div className="flex items-center gap-2 px-2 py-3 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
              Searching paths
            </div>
          )}
          {!showLoading && results.length === 0 && (
            <div className="px-2 py-3 text-sm text-muted-foreground">
              No paths found
            </div>
          )}
          {results.map((result) => (
            <button
              key={result.path}
              type="button"
              role="option"
              aria-selected={value === result.path}
              className="flex w-full items-center gap-2 rounded-sm px-2 py-2 text-left text-sm hover:bg-accent"
              onClick={() => {
                onChange(result.path);
                setOpen(false);
              }}
              aria-label={`Select ${result.path}`}
            >
              {result.kind === 'directory' ? (
                <Folder className="h-4 w-4 shrink-0" />
              ) : (
                <FileArchive className="h-4 w-4 shrink-0" />
              )}
              <span className="min-w-0 flex-1 truncate">{result.path}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
