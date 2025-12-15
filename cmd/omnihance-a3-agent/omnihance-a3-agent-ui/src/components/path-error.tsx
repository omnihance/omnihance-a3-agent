import { Link } from '@tanstack/react-router';
import { AlertCircle, Home } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';

interface PathErrorProps {
  title?: string;
  description?: string;
  showBackToDashboard?: boolean;
}

export function PathError({
  title = 'File Path Required',
  description = 'No file path was provided. Please select a file from the project directory to view or edit.',
  showBackToDashboard = true,
}: PathErrorProps) {
  return (
    <div className="flex min-h-[calc(100vh-8rem)] items-center justify-center p-4 lg:p-6">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-destructive/10">
            <AlertCircle className="h-8 w-8 text-destructive" />
          </div>
          <CardTitle className="text-2xl">{title}</CardTitle>
          <CardDescription className="mt-2 text-base">
            {description}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              The file path query parameter is missing or empty. Please navigate
              to a file from the project directory.
            </AlertDescription>
          </Alert>
          {showBackToDashboard && (
            <Button asChild className="w-full">
              <Link to="/dashboard">
                <Home className="mr-2 h-4 w-4" />
                Go to Dashboard
              </Link>
            </Button>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
