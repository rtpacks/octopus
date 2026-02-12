'use client';

import { useState } from 'react';
import { motion } from 'motion/react';
import { KeyRound, Mail, Clock, RefreshCw, Trash2, Power, PowerOff, AlertTriangle, CheckCircle } from 'lucide-react';
import { Card as CardPrimitive, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import {
    useUpdateOAuthToken,
    useDeleteOAuthToken,
    useRefreshOAuthToken,
    type OAuthTokenResponse,
} from '@/api/endpoints/oauth';
import { formatTime } from '@/lib/utils';

interface CardProps {
    token: OAuthTokenResponse;
}

export function Card({ token }: CardProps) {
    const [showDeleteDialog, setShowDeleteDialog] = useState(false);
    const updateToken = useUpdateOAuthToken();
    const deleteToken = useDeleteOAuthToken();
    const refreshToken = useRefreshOAuthToken();

    const handleToggleEnabled = () => {
        updateToken.mutate({
            id: token.id,
            enabled: !token.enabled,
        });
    };

    const handleRefresh = () => {
        refreshToken.mutate(token.id);
    };

    const handleDelete = () => {
        deleteToken.mutate(token.id, {
            onSuccess: () => {
                setShowDeleteDialog(false);
            },
        });
    };

    const getTypeBadge = () => {
        const typeColors: Record<string, string> = {
            codex: 'bg-green-500/10 text-green-500 border-green-500/20',
            antigravity: 'bg-blue-500/10 text-blue-500 border-blue-500/20',
        };
        return (
            <Badge variant="outline" className={typeColors[token.type] || ''}>
                {token.type.toUpperCase()}
            </Badge>
        );
    };

    const getStatusBadge = () => {
        if (!token.enabled) {
            return <Badge variant="outline" className="bg-gray-500/10 text-gray-500">Disabled</Badge>;
        }
        if (token.is_expired) {
            return <Badge variant="outline" className="bg-red-500/10 text-red-500">Expired</Badge>;
        }
        return <Badge variant="outline" className="bg-green-500/10 text-green-500">Active</Badge>;
    };

    const getExpiresIn = () => {
        if (!token.expires_at) return 'Unknown';
        const expiresAt = new Date(token.expires_at).getTime();
        const now = Date.now();
        const diff = expiresAt - now;

        if (diff <= 0) return 'Expired';
        if (diff < 60 * 60 * 1000) return `${Math.floor(diff / 60000)}m`;
        if (diff < 24 * 60 * 60 * 1000) return `${Math.floor(diff / 3600000)}h`;
        return `${Math.floor(diff / 86400000)}d`;
    };

    return (
        <>
            <CardPrimitive className="group relative overflow-hidden">
                <CardHeader className="pb-2">
                    <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                            <KeyRound className="h-4 w-4 text-muted-foreground" />
                            <CardTitle className="text-sm font-medium">
                                {token.email || `Token #${token.id}`}
                            </CardTitle>
                        </div>
                        <div className="flex items-center gap-2">
                            {getTypeBadge()}
                            {getStatusBadge()}
                        </div>
                    </div>
                </CardHeader>
                <CardContent className="space-y-3">
                    {/* Email */}
                    {token.email && (
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            <Mail className="h-3.5 w-3.5" />
                            <span className="truncate">{token.email}</span>
                        </div>
                    )}

                    {/* Expiry Info */}
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                        <Clock className="h-3.5 w-3.5" />
                        <span>Expires in: {getExpiresIn()}</span>
                        {token.is_expired && (
                            <AlertTriangle className="h-3.5 w-3.5 text-red-500 ml-1" />
                        )}
                    </div>

                    {/* Last Refresh */}
                    {token.last_refresh && (
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            <RefreshCw className="h-3.5 w-3.5" />
                            <span>Last refresh: {formatTime(new Date(token.last_refresh).getTime() / 1000).formatted.value}{formatTime(new Date(token.last_refresh).getTime() / 1000).formatted.unit} ago</span>
                        </div>
                    )}

                    {/* Remark */}
                    {token.remark && (
                        <div className="text-xs text-muted-foreground truncate">
                            {token.remark}
                        </div>
                    )}

                    {/* Actions */}
                    <div className="flex items-center justify-between pt-2 border-t border-border/50">
                        <div className="flex items-center gap-2">
                            <Switch
                                checked={token.enabled}
                                onCheckedChange={handleToggleEnabled}
                                disabled={updateToken.isPending}
                            />
                            <span className="text-xs text-muted-foreground">
                                {token.enabled ? <Power className="h-3.5 w-3.5" /> : <PowerOff className="h-3.5 w-3.5" />}
                            </span>
                        </div>
                        <div className="flex items-center gap-1">
                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                onClick={handleRefresh}
                                disabled={refreshToken.isPending || token.is_expired}
                            >
                                <RefreshCw className={`h-4 w-4 ${refreshToken.isPending ? 'animate-spin' : ''}`} />
                            </Button>
                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8 text-destructive hover:text-destructive"
                                onClick={() => setShowDeleteDialog(true)}
                            >
                                <Trash2 className="h-4 w-4" />
                            </Button>
                        </div>
                    </div>
                </CardContent>
            </CardPrimitive>

            {/* Delete Confirmation Dialog */}
            <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete OAuth Token</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to delete this OAuth token? This action cannot be undone.
                            {token.email && (
                                <div className="mt-2 text-sm">
                                    Token: <span className="font-medium">{token.email}</span>
                                </div>
                            )}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={handleDelete}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                            Delete
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </>
    );
}
