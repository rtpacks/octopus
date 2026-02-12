'use client';

import { useState } from 'react';
import { Plus, KeyRound, ArrowLeft, Loader2 } from 'lucide-react';
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
import { useCreateOAuthToken, type OAuthProvider } from '@/api/endpoints/oauth';
import { toast } from '@/components/common/Toast';

type View = 'menu' | 'manual';

export function CreateDialogContent() {
    const [view, setView] = useState<View>('menu');
    const createToken = useCreateOAuthToken();

    // Manual form state
    const [provider, setProvider] = useState<OAuthProvider>('codex');
    const [email, setEmail] = useState('');
    const [accessToken, setAccessToken] = useState('');
    const [refreshToken, setRefreshToken] = useState('');
    const [expiresIn, setExpiresIn] = useState('');
    const [remark, setRemark] = useState('');

    const handleManualSubmit = () => {
        const expiresInNum = expiresIn ? parseInt(expiresIn, 10) : undefined;

        createToken.mutate({
            type: provider,
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
    };

    const isValid = accessToken.trim() !== '';

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

                        {/* Manual Option - Recommended */}
                        <div className="grid gap-2">
                            <Button
                                variant="outline"
                                className="justify-start h-auto py-3"
                                onClick={() => setView('manual')}
                                type="button"
                            >
                                <div className="flex flex-col items-start gap-1">
                                    <span className="flex items-center gap-2 font-medium">
                                        <Plus className="h-4 w-4" />
                                        Add Token Manually (Recommended)
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
                                <strong>Tip:</strong> You can get your access token from your browser&apos;s developer tools
                                after logging in to ChatGPT or Google.
                            </p>
                        </div>
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
                        <Label htmlFor="provider">Provider</Label>
                        <Select value={provider} onValueChange={(v) => setProvider(v as OAuthProvider)}>
                            <SelectTrigger id="provider">
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
                            disabled={!isValid || createToken.isPending}
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
