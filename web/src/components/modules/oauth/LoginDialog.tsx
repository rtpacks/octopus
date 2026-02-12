'use client';

import { useState } from 'react';
import { ExternalLink, Loader2 } from 'lucide-react';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { useOAuthStart, type OAuthProvider } from '@/api/endpoints/oauth';
import { logger } from '@/lib/logger';

interface LoginDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    provider: OAuthProvider | null;
}

const PROVIDER_INFO: Record<OAuthProvider, { name: string; description: string }> = {
    codex: {
        name: 'Codex / ChatGPT',
        description: 'Login with your ChatGPT account to access Codex API',
    },
    antigravity: {
        name: 'Antigravity / Gemini',
        description: 'Login with your Google account to access Gemini API via Antigravity',
    },
};

export function LoginDialog({ open, onOpenChange, provider }: LoginDialogProps) {
    const [authStarted, setAuthStarted] = useState(false);
    const startOAuth = useOAuthStart();

    const handleStartAuth = () => {
        if (!provider) return;

        startOAuth.mutate(provider, {
            onSuccess: (data) => {
                logger.log('OAuth started:', data);
                setAuthStarted(true);
                // Open the auth URL in a new window
                window.open(data.auth_url, '_blank', 'width=600,height=800');
            },
            onError: (error) => {
                logger.error('OAuth start failed:', error);
            },
        });
    };

    const handleClose = () => {
        setAuthStarted(false);
        onOpenChange(false);
    };

    if (!provider) return null;

    const info = PROVIDER_INFO[provider];

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>OAuth Login - {info.name}</DialogTitle>
                    <DialogDescription>
                        {info.description}
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4">
                    {!authStarted ? (
                        <>
                            <div className="text-sm text-muted-foreground">
                                <p className="mb-2">Click the button below to start the OAuth login process.</p>
                                <p>A new window will open for you to authenticate with the provider.</p>
                            </div>

                            <div className="flex gap-2">
                                <Button
                                    onClick={handleStartAuth}
                                    disabled={startOAuth.isPending}
                                    className="flex-1"
                                >
                                    {startOAuth.isPending && (
                                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                    )}
                                    Start Login
                                    <ExternalLink className="ml-2 h-4 w-4" />
                                </Button>
                                <Button variant="outline" onClick={handleClose}>
                                    Cancel
                                </Button>
                            </div>
                        </>
                    ) : (
                        <>
                            <div className="flex flex-col items-center justify-center py-4 space-y-4">
                                <div className="flex items-center justify-center w-12 h-12 rounded-full bg-green-500/10">
                                    <svg
                                        className="w-6 h-6 text-green-500"
                                        fill="none"
                                        stroke="currentColor"
                                        viewBox="0 0 24 24"
                                    >
                                        <path
                                            strokeLinecap="round"
                                            strokeLinejoin="round"
                                            strokeWidth={2}
                                            d="M5 13l4 4L19 7"
                                        />
                                    </svg>
                                </div>
                                <div className="text-center">
                                    <p className="font-medium">Authentication window opened</p>
                                    <p className="text-sm text-muted-foreground mt-1">
                                        Complete the login in the opened window.
                                        <br />
                                        The token will appear in the list after successful authentication.
                                    </p>
                                </div>
                            </div>

                            <div className="flex justify-end">
                                <Button variant="outline" onClick={handleClose}>
                                    Close
                                </Button>
                            </div>
                        </>
                    )}
                </div>
            </DialogContent>
        </Dialog>
    );
}
