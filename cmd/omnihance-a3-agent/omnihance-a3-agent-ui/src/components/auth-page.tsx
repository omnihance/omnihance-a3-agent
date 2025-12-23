import { useState, useEffect, useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Link, useRouter } from '@tanstack/react-router';
import { useMutation } from '@tanstack/react-query';
import { Server, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ThemeToggle } from '@/components/theme-toggle';
import { signIn, signUp, APIError } from '@/lib/api';
import { APP_NAME } from '@/constants';
import { useStatus } from '@/hooks/use-status';

const loginSchema = z.object({
  email: z.email('Please enter a valid email'),
  password: z.string().min(6, 'Password must be at least 6 characters'),
});

const signUpSchema = z
  .object({
    email: z.email('Please enter a valid email'),
    password: z.string().min(6, 'Password must be at least 6 characters'),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords don't match",
    path: ['confirmPassword'],
  });

type LoginFormData = z.infer<typeof loginSchema>;
type SignUpFormData = z.infer<typeof signUpSchema>;

export function AuthPage() {
  const { status, isLoading: isStatusLoading } = useStatus();
  const [manualSignUpOverride, setManualSignUpOverride] = useState<
    boolean | null
  >(null);
  const [error, setError] = useState<string | null>(null);
  const router = useRouter();

  const isSignUp = useMemo(() => {
    if (manualSignUpOverride !== null) {
      return manualSignUpOverride;
    }

    if (status && !status.setup_done) {
      return true;
    }

    return false;
  }, [status, manualSignUpOverride]);

  useEffect(() => {
    document.title = isSignUp
      ? `Sign Up - ${APP_NAME}`
      : `Sign In - ${APP_NAME}`;
  }, [isSignUp]);

  const loginForm = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: '',
      password: '',
    },
  });

  const signUpForm = useForm<SignUpFormData>({
    resolver: zodResolver(signUpSchema),
    defaultValues: {
      email: '',
      password: '',
      confirmPassword: '',
    },
  });

  const signInMutation = useMutation({
    mutationFn: signIn,
    onSuccess: () => {
      router.navigate({ to: '/dashboard' });
    },
    onError: (err: unknown) => {
      if (err instanceof APIError) {
        setError(err.getErrorMessage());
      } else {
        setError(err instanceof Error ? err.message : 'Login failed');
      }
    },
  });

  const signUpMutation = useMutation({
    mutationFn: signUp,
    onSuccess: (data) => {
      toast.success(data.message);
      signUpForm.reset();
      setManualSignUpOverride(false);
    },
    onError: (err: unknown) => {
      if (err instanceof APIError) {
        setError(err.getErrorMessage());
      } else {
        setError(err instanceof Error ? err.message : 'Sign up failed');
      }
    },
  });

  const onLogin = async (data: LoginFormData) => {
    setError(null);
    signInMutation.mutate({
      email: data.email,
      password: data.password,
    });
  };

  const onSignUp = async (data: SignUpFormData) => {
    setError(null);
    signUpMutation.mutate({
      email: data.email,
      password: data.password,
    });
  };

  const handleToggleMode = (newMode: boolean) => {
    setError(null);
    setManualSignUpOverride(newMode);
  };

  return (
    <div className="flex min-h-screen flex-col bg-background">
      {/* Header */}
      <header className="border-b">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
          <Link to="/" className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
              <Server className="h-5 w-5 text-primary-foreground" />
            </div>
            <span className="text-xl font-bold">Omnihance</span>
          </Link>
          <ThemeToggle />
        </div>
      </header>

      {/* Auth Form */}
      <main className="flex flex-1 items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <CardTitle className="text-2xl">
              {isSignUp ? 'Create an account' : 'Welcome back'}
            </CardTitle>
            <CardDescription>
              {isSignUp
                ? 'Enter your details to get started'
                : 'Enter your credentials to access your account'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {isStatusLoading && (
              <div className="mb-4 flex items-center justify-center py-4">
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              </div>
            )}

            {error && (
              <div className="mb-4 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {error}
              </div>
            )}

            {isSignUp ? (
              <form
                onSubmit={signUpForm.handleSubmit(onSignUp)}
                className="space-y-4"
              >
                <div className="space-y-2">
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    placeholder="Enter your email"
                    {...signUpForm.register('email')}
                  />
                  {signUpForm.formState.errors.email && (
                    <p className="text-xs text-destructive">
                      {signUpForm.formState.errors.email.message}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="password">Password</Label>
                  <Input
                    id="password"
                    type="password"
                    placeholder="Create a password"
                    {...signUpForm.register('password')}
                  />
                  {signUpForm.formState.errors.password && (
                    <p className="text-xs text-destructive">
                      {signUpForm.formState.errors.password.message}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="confirmPassword">Confirm Password</Label>
                  <Input
                    id="confirmPassword"
                    type="password"
                    placeholder="Confirm your password"
                    {...signUpForm.register('confirmPassword')}
                  />
                  {signUpForm.formState.errors.confirmPassword && (
                    <p className="text-xs text-destructive">
                      {signUpForm.formState.errors.confirmPassword.message}
                    </p>
                  )}
                </div>
                <Button
                  type="submit"
                  className="w-full"
                  disabled={signUpMutation.isPending}
                >
                  {signUpMutation.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Create account
                </Button>
              </form>
            ) : (
              <form
                onSubmit={loginForm.handleSubmit(onLogin)}
                className="space-y-4"
              >
                <div className="space-y-2">
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    placeholder="Enter your email"
                    {...loginForm.register('email')}
                  />
                  {loginForm.formState.errors.email && (
                    <p className="text-xs text-destructive">
                      {loginForm.formState.errors.email.message}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="password">Password</Label>
                  <Input
                    id="password"
                    type="password"
                    placeholder="Enter your password"
                    {...loginForm.register('password')}
                  />
                  {loginForm.formState.errors.password && (
                    <p className="text-xs text-destructive">
                      {loginForm.formState.errors.password.message}
                    </p>
                  )}
                </div>
                <Button
                  type="submit"
                  className="w-full"
                  disabled={signInMutation.isPending}
                >
                  {signInMutation.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Sign in
                </Button>
              </form>
            )}

            {status?.setup_done && (
              <div className="mt-6 text-center text-sm">
                {isSignUp ? (
                  <>
                    Already have an account?{' '}
                    <button
                      type="button"
                      onClick={() => handleToggleMode(false)}
                      className="font-medium text-primary hover:underline"
                    >
                      Sign in
                    </button>
                  </>
                ) : (
                  <>
                    Don&apos;t have an account?{' '}
                    <button
                      type="button"
                      onClick={() => handleToggleMode(true)}
                      className="font-medium text-primary hover:underline"
                    >
                      Sign up
                    </button>
                  </>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </main>
    </div>
  );
}
