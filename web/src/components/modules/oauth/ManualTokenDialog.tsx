'use client';

import { useState } from 'react';
import { Loader2 } from 'lucide-react';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from '@/components/ui/dialog';
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
import { useCreateOAuthToken, type OAuthProvider } from '@/api/endpoints/oauth';

interface ManualTokenDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export function ManualTokenDialog({ open, onOpenChange }: ManualTokenDialogProps) {
    const [provider, setProvider] = useState<OAuthProvider>('codex');
    const [email, setEmail] = useState('');
    const [accessToken, setAccessToken] = useState('');
    const [refreshToken, setRefreshToken] = useState('');
    const [expiresIn, setExpiresIn] = useState('');
    const [remark, setRemark] = useState('');

    const createToken = useCreateOAuthToken();

    const handleSubmit = () => {
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
                // Reset form
                setEmail('');
                setAccessToken('');
                setRefreshToken('');
                setExpiresIn('');
                setRemark('');
                onOpenChange(false);
            },
        });
    };

    const isValid = accessToken.trim() !== '';

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Add Token Manually</DialogTitle>
                    <DialogDescription>
                        Enter the OAuth token details manually. You can get these values from your browser's developer tools during an OAuth login.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4">
                    {/* Provider */}
                    <div className="space-y-2">
                        <Label htmlFor="provider">Provider</Label>
                        <Select value={provider} onValueChange={(v) => setProvider(v as OAuthProvider)}>
                            <SelectTrigger>
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
                </div>

                <DialogFooter>
                    <Button variant="outline" onClick={() => onOpenChange(false)}>
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSubmit}
                        disabled={!isValid || createToken.isPending}
                    >
                        {createToken.isPending && (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        )}
                        Add Token
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
