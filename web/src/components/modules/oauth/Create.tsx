'use client';

import { useState } from 'react';
import { Plus, KeyRound, ArrowLeft, Loader2, ExternalLink, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import {
    MorphingDialogTitle,
    MorphingDialogDescription,
    MorphingDialogClose,
} from '@/components/ui/morphing-dialog';
import { useCreateOAuthToken, useOAuthStart, useSubmitOAuthCallback, type OAuthProvider } from '@/api/endpoints/oauth';
import { toast } from '@/components/common/Toast';

type View = 'menu' | 'oauth' | 'manual';

// Provider configurations
const PROVIDER_CONFIG: Record<OAuthProvider, {
    name: string;
    description: string;
    callbackUrl: string;
    authUrl: string;
}> = {
    codex: {
        name: 'Codex / ChatGPT',
        description: 'Login with your ChatGPT account to access Codex API',
        callbackUrl: 'http://localhost:1455/auth/callback',
        authUrl: 'https://auth.openai.com/oauth/authorize',
    },
    antigravity: {
        name: 'Antigravity / Gemini',
        description: 'Login with your Google account to access Gemini API',
        callbackUrl: 'http://localhost:51121/oauth-callback',
        authUrl: 'https://accounts.google.com/o/oauth2/v2/auth',
    },
};

export function CreateDialogContent() {
    const [view, setView] = useState<View>('menu');
    const createToken = useCreateOAuthToken();
    const startOAuth = useOAuthStart();
    const submitCallback = useSubmitOAuthCallback();

    // OAuth form state
    const [oauthProvider, setOAuthProvider] = useState<OAuthProvider>('codex');
    const [callbackUrl, setCallbackUrl] = useState('');
    const [authStarted, setAuthStarted] = useState(false);
    const [authUrl, setAuthUrl] = useState('');

    // Manual form state
    const [manualProvider, setManualProvider] = useState<OAuthProvider>('codex');
    const [email, setEmail] = useState('');
    const [accessToken, setAccessToken] = useState('');
    const [refreshToken, setRefreshToken] = useState('');
    const [expiresIn, setExpiresIn] = useState('');
    const [remark, setRemark] = useState('');

    // Handle OAuth login start
    const handleStartOAuth = async () => {
        startOAuth.mutate(oauthProvider, {
            onSuccess: (data) => {
                setAuthUrl(data.auth_url);
                setAuthStarted(true);
                // Open auth window
                window.open(data.auth_url, '_blank', 'width=600,height=800');
            },
            onError: (error) => {
                toast.error('Failed to start OAuth login');
                console.error(error);
            },
        });
    };

    // Handle OAuth callback submission
    const handleOAuthSubmit = () => {
        // Parse the callback URL to extract code
        try {
            const url = new URL(callbackUrl.trim());
            const code = url.searchParams.get('code');

            if (!code) {
                toast.error('Invalid callback URL: missing code parameter');
                return;
            }

            // Submit to backend to exchange code for tokens
            submitCallback.mutate({
                provider: oauthProvider,
                callback_url: callbackUrl.trim(),
            }, {
                onSuccess: () => {
                    toast.success('OAuth Token created successfully');
                    // Reset and go back to menu
                    setView('menu');
                    setCallbackUrl('');
                    setAuthStarted(false);
                },
                onError: (error: any) {
                    const errorMsg = error?.response?.data?.error || error?.message || 'Failed to submit callback URL';
                    toast.error('Failed to create token: ' + errorMsg);
                },
            });
        } catch (error) {
            toast.error('Invalid callback URL format');
        }
    };

    // Handle manual token submission
    const handleManualSubmit = () => {
        const expiresInNum = expiresIn ? parseInt(expiresIn, 10) : undefined;

        createToken.mutate({
            type: manualProvider,
            email: email || undefined,
            access_token: accessToken,
            refresh_token: refreshToken || undefined,
            expires_in: expiresInNum,
            remark: remark || undefined,
        }, {
            onSuccess: () => {
                toast.success('OAuth Token created successfully');
                // Reset form
                setEmail('');
                setAccessToken('');
                setRefreshToken('');
                setExpiresIn('');
                setRemark('');
                setView('menu');
            },
            onError: (error) => {
                toast.error('Failed to create OAuth Token');
                console.error(error);
            },
        });
    };

    const handleBack = () => {
        setView('menu');
        setAuthStarted(false);
        setCallbackUrl('');
    };

    const isValidManual = accessToken.trim() !== '';
    const isValidOAuth = callbackUrl.trim() !== '';

    if (view === 'menu') {
        return (
            <div className="w-screen max-w-full md:max-w-md h-full min-h-0 flex flex-col">
                <MorphingDialogTitle className="shrink-0">
                    <header className="mb-4 flex items-center justify-between">
                        <div className="flex items-center gap-2">
                            <KeyRound className="h-5 w-5" />
                            <h2 className="text-xl font-bold text-card-foreground">Add OAuth Token</h2>
                        </div>
                        <MorphingDialogClose
                            className="relative right-0 top-0"
                            variants={{
                                initial: { opacity: 0, scale: 0.8 },
                                animate: { opacity: 1, scale: 1 },
                                exit: { opacity: 0, scale: 0.8 }
                            }}
                        />
                    </header>
                </MorphingDialogTitle>
                <MorphingDialogDescription disableLayoutAnimation className="flex-1 min-h-0">
                    <div className="space-y-4">
                        <p className="text-sm text-muted-foreground">
                            Choose how you want to add an OAuth token:
                        </p>

                        <div className="grid gap-2">
                            {/* OAuth Login Option */}
                            <Button
                                variant="outline"
                                className="justify-start h-auto py-3"
                                onClick={() => setView('oauth')}
                                type="button"
                            >
                                <div className="flex flex-col items-start gap-1">
                                    <span className="flex items-center gap-2 font-medium">
                                        <KeyRound className="h-4 w-4" />
                                        OAuth Login
                                    </span>
                                    <span className="text-xs text-muted-foreground pl-6">
                                        Login with ChatGPT or Google account
                                    </span>
                                </div>
                            </Button>

                            {/* Manual Option */}
                            <Button
                                variant="outline"
                                className="justify-start h-auto py-3"
                                onClick={() => setView('manual')}
                                type="button"
                            >
                                <div className="flex flex-col items-start gap-1">
                                    <span className="flex items-center gap-2 font-medium">
                                        <Plus className="h-4 w-4" />
                                        Add Token Manually
                                    </span>
                                    <span className="text-xs text-muted-foreground pl-6">
                                        Paste your access token directly
                                    </span>
                                </div>
                            </Button>
                        </div>

                        {/* Info */}
                        <div className="pt-2 border-t text-xs text-muted-foreground">
                            <p>
                                <strong>OAuth Login:</strong> Click to open authentication window, then copy the callback URL.<br/>
                                <strong>Manual:</strong> Get access token from browser developer tools.
                            </p>
                        </div>
                    </div>
                </MorphingDialogDescription>
            </div>
        );
    }

    // OAuth Login view
    if (view === 'oauth') {
        const config = PROVIDER_CONFIG[oauthProvider];

        return (
            <div className="w-screen max-w-full md:max-w-md h-full min-h-0 flex flex-col">
                <MorphingDialogTitle className="shrink-0">
                    <header className="mb-4 flex items-center gap-2">
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8"
                            onClick={handleBack}
                            type="button"
                        >
                            <ArrowLeft className="h-4 w-4" />
                        </Button>
                        <h2 className="text-xl font-bold text-card-foreground">OAuth Login</h2>
                    </header>
                </MorphingDialogTitle>
                <MorphingDialogDescription disableLayoutAnimation className="flex-1 min-h-0 overflow-auto">
                    <div className="space-y-4">
                        {/* Provider Selection */}
                        <div className="space-y-2">
                            <Label>Provider</Label>
                            <Select value={oauthProvider} onValueChange={(v) => setOAuthProvider(v as OAuthProvider)}>
                                <SelectTrigger>
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="codex">Codex / ChatGPT</SelectItem>
                                    <SelectItem value="antigravity">Antigravity / Gemini</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>

                        {/* Provider Info */}
                        <div className="p-3 rounded-lg bg-muted/50 border border-border/50">
                            <p className="text-sm font-medium">{config.name}</p>
                            <p className="text-xs text-muted-foreground mt-1">{config.description}</p>
                        </div>

                        {/* Step 1: Start OAuth */}
                        <div className="space-y-2">
                            <div className="flex items-center gap-2">
                                <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-bold">1</div>
                                <Label className="font-medium">Start OAuth Login</Label>
                            </div>
                            <Button
                                onClick={handleStartOAuth}
                                disabled={startOAuth.isPending}
                                className="w-full"
                                type="button"
                            >
                                {startOAuth.isPending && (
                                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                )}
                                Open Authentication Window
                                <ExternalLink className="ml-2 h-4 w-4" />
                            </Button>
                            <p className="text-xs text-muted-foreground">
                                A new window will open for you to authenticate with {config.name}.
                            </p>
                        </div>

                        {/* Step 2: Copy Callback URL */}
                        {authStarted && (
                            <div className="space-y-2">
                                <div className="flex items-center gap-2">
                                    <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-primary-foreground text-xs font-bold">2</div>
                                    <Label className="font-medium">Copy Callback URL</Label>
                                </div>
                                <div className="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
                                    <div className="flex items-start gap-2">
                                        <AlertCircle className="h-4 w-4 text-amber-500 mt-0.5 shrink-0" />
                                        <div className="text-xs text-muted-foreground">
                                            <p className="mb-2">After completing authentication:</p>
                                            <ol className="list-decimal list-inside space-y-1">
                                                <li>Copy the callback URL from your browser address bar</li>
                                                <li>Paste it in the input box below</li>
                                                <li>Click "Submit Callback URL" to get your access token</li>
                                            </ol>
                                            <p className="mt-2 text-amber-600 font-medium">
                                                Expected callback: {config.callbackUrl}
                                            </p>
                                        </div>
                                    </div>
                                </div>
                                <Input
                                    placeholder="Paste callback URL here..."
                                    value={callbackUrl}
                                    onChange={(e) => setCallbackUrl(e.target.value)}
                                    className="font-mono text-xs"
                                />
                            </div>
                        )}

                        {/* Submit Button */}
                        {authStarted && (
                            <Button
                                onClick={handleOAuthSubmit}
                                disabled={!isValidOAuth || submitCallback.isPending}
                                className="w-full"
                                type="button"
                            >
                                {submitCallback.isPending && (
                                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                )}
                                Submit Callback URL
                            </Button>
                        )}
                    </div>
                </MorphingDialogDescription>
            </div>
        );
    }

    // Manual form view
    return (
        <div className="w-screen max-w-full md:max-w-md h-full min-h-0 flex flex-col">
            <MorphingDialogTitle className="shrink-0">
                <header className="mb-4 flex items-center gap-2">
                    <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={handleBack}
                        type="button"
                    >
                        <ArrowLeft className="h-4 w-4" />
                    </Button>
                    <h2 className="text-xl font-bold text-card-foreground">Add Token Manually</h2>
                </header>
            </MorphingDialogTitle>
            <MorphingDialogDescription disableLayoutAnimation className="flex-1 min-h-0 overflow-auto">
                <div className="space-y-4">
                    {/* Provider */}
                    <div className="space-y-2">
                        <Label htmlFor="manualProvider">Provider</Label>
                        <Select value={manualProvider} onValueChange={(v) => setManualProvider(v as OAuthProvider)}>
                            <SelectTrigger id="manualProvider">
                                <SelectValue placeholder="Select provider" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="codex">Codex / ChatGPT</SelectItem>
                                <SelectItem value="antigravity">Antigravity / Gemini</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>

                    {/* Email */}
                    <div className="space-y-2">
                        <Label htmlFor="email">Email (optional)</Label>
                        <Input
                            id="email"
                            type="email"
                            placeholder="user@example.com"
                            value={email}
                            onChange={(e) => setEmail(e.target.value)}
                        />
                    </div>

                    {/* Access Token */}
                    <div className="space-y-2">
                        <Label htmlFor="accessToken">Access Token *</Label>
                        <Input
                            id="accessToken"
                            type="password"
                            placeholder="Enter access token"
                            value={accessToken}
                            onChange={(e) => setAccessToken(e.target.value)}
                        />
                    </div>

                    {/* Refresh Token */}
                    <div className="space-y-2">
                        <Label htmlFor="refreshToken">Refresh Token (optional)</Label>
                        <Input
                            id="refreshToken"
                            type="password"
                            placeholder="Enter refresh token"
                            value={refreshToken}
                            onChange={(e) => setRefreshToken(e.target.value)}
                        />
                    </div>

                    {/* Expires In */}
                    <div className="space-y-2">
                        <Label htmlFor="expiresIn">Expires In (seconds, optional)</Label>
                        <Input
                            id="expiresIn"
                            type="number"
                            placeholder="3600"
                            value={expiresIn}
                            onChange={(e) => setExpiresIn(e.target.value)}
                        />
                    </div>

                    {/* Remark */}
                    <div className="space-y-2">
                        <Label htmlFor="remark">Remark (optional)</Label>
                        <Input
                            id="remark"
                            type="text"
                            placeholder="e.g., Production account"
                            value={remark}
                            onChange={(e) => setRemark(e.target.value)}
                        />
                    </div>

                    {/* Submit Button */}
                    <div className="flex gap-2 pt-2">
                        <Button
                            variant="outline"
                            onClick={handleBack}
                            className="flex-1"
                            type="button"
                        >
                            Cancel
                        </Button>
                        <Button
                            onClick={handleManualSubmit}
                            disabled={!isValidManual || createToken.isPending}
                            className="flex-1"
                            type="button"
                        >
                            {createToken.isPending && (
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            )}
                            Add Token
                        </Button>
                    </div>
                </div>
            </MorphingDialogDescription>
        </div>
    );
}
