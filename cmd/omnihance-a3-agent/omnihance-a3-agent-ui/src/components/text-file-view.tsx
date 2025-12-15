import { Card, CardContent } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';

interface TextFileViewProps {
  content: string;
}

export function TextFileView({ content }: TextFileViewProps) {
  return (
    <Card>
      <CardContent className="p-0">
        <ScrollArea className="h-[600px] w-full">
          <pre className="p-4 text-sm font-mono whitespace-pre-wrap break-words">
            {content}
          </pre>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
