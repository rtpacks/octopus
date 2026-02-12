import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';
import { logger } from '@/lib/logger';

/**
 * OAuth 提供商类型
 */
export type OAuthProvider = 'codex' | 'antigravity';

/**
 * 认证类型
 */
export type AuthType = 'api_key' | 'oauth_codex' | 'oauth_antigravity';

/**
 * OAuth Token 响应类型
 */
export interface OAuthTokenResponse {
    id: number;
    type: OAuthProvider;
    email: string;
    expires_at: string | null;
    last_refresh: string | null;
    created_at: string;
    updated_at: string;
    enabled: boolean;
    remark: string;
    is_expired: boolean;
}

/**
 * OAuth 启动响应
 */
export interface OAuthStartResponse {
    auth_url: string;
    session_id: string;
    provider: OAuthProvider;
}

/**
 * OAuth Token 状态
 */
export interface OAuthTokenStatus {
    id: number;
    type: OAuthProvider;
    email: string;
    enabled: boolean;
    is_expired: boolean;
    needs_refresh: boolean;
    expires_at?: string;
    expires_in?: number;
    last_refresh?: string;
}

/**
 * 创建 OAuth Token 请求
 */
export interface CreateOAuthTokenRequest {
    type: OAuthProvider;
    email?: string;
    access_token: string;
    refresh_token?: string;
    id_token?: string;
    expires_in?: number;
    remark?: string;
}

/**
 * 更新 OAuth Token 请求
 */
export interface UpdateOAuthTokenRequest {
    id: number;
    enabled?: boolean;
    remark?: string;
    access_token?: string;
    refresh_token?: string;
    expires_in?: number;
}

/**
 * 启动 OAuth 登录流程 Hook
 *
 * @example
 * const startOAuth = useOAuthStart();
 *
 * startOAuth.mutate('codex', {
 *   onSuccess: (data) => {
 *     window.open(data.auth_url, '_blank');
 *   }
 * });
 */
export function useOAuthStart() {
    return useMutation({
        mutationFn: async (provider: OAuthProvider) => {
            return apiClient.post<OAuthStartResponse>(`/api/v1/oauth/start/${provider}`, {});
        },
        onSuccess: (data) => {
            logger.log('OAuth 流程启动成功:', data);
        },
        onError: (error) => {
            logger.error('OAuth 流程启动失败:', error);
        },
    });
}

/**
 * 获取 OAuth Token 列表 Hook
 *
 * @example
 * const { data: tokens, isLoading } = useOAuthTokenList();
 */
export function useOAuthTokenList() {
    return useQuery({
        queryKey: ['oauth', 'tokens'],
        queryFn: async () => {
            return apiClient.get<OAuthTokenResponse[]>('/api/v1/oauth/token/list');
        },
        refetchInterval: 60000,
    });
}

/**
 * 手动创建 OAuth Token Hook
 *
 * @example
 * const createToken = useCreateOAuthToken();
 *
 * createToken.mutate({
 *   type: 'codex',
 *   access_token: 'xxx',
 *   refresh_token: 'xxx',
 *   expires_in: 3600,
 * });
 */
export function useCreateOAuthToken() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: CreateOAuthTokenRequest) => {
            return apiClient.post<OAuthTokenResponse>('/api/v1/oauth/token/create', data);
        },
        onSuccess: (data) => {
            logger.log('OAuth Token 创建成功:', data);
            queryClient.invalidateQueries({ queryKey: ['oauth', 'tokens'] });
        },
        onError: (error) => {
            logger.error('OAuth Token 创建失败:', error);
        },
    });
}

/**
 * 更新 OAuth Token Hook
 *
 * @example
 * const updateToken = useUpdateOAuthToken();
 *
 * updateToken.mutate({
 *   id: 1,
 *   enabled: false,
 *   remark: '备用账号',
 * });
 */
export function useUpdateOAuthToken() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: UpdateOAuthTokenRequest) => {
            return apiClient.post<OAuthTokenResponse>('/api/v1/oauth/token/update', data);
        },
        onSuccess: (data) => {
            logger.log('OAuth Token 更新成功:', data);
            queryClient.invalidateQueries({ queryKey: ['oauth', 'tokens'] });
        },
        onError: (error) => {
            logger.error('OAuth Token 更新失败:', error);
        },
    });
}

/**
 * 删除 OAuth Token Hook
 *
 * @example
 * const deleteToken = useDeleteOAuthToken();
 *
 * deleteToken.mutate(1);
 */
export function useDeleteOAuthToken() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (id: number) => {
            return apiClient.delete<null>(`/api/v1/oauth/token/delete/${id}`);
        },
        onSuccess: () => {
            logger.log('OAuth Token 删除成功');
            queryClient.invalidateQueries({ queryKey: ['oauth', 'tokens'] });
        },
        onError: (error) => {
            logger.error('OAuth Token 删除失败:', error);
        },
    });
}

/**
 * 刷新 OAuth Token Hook
 *
 * @example
 * const refreshToken = useRefreshOAuthToken();
 *
 * refreshToken.mutate(1);
 */
export function useRefreshOAuthToken() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (id: number) => {
            return apiClient.post<OAuthTokenResponse>(`/api/v1/oauth/token/refresh/${id}`, {});
        },
        onSuccess: (data) => {
            logger.log('OAuth Token 刷新成功:', data);
            queryClient.invalidateQueries({ queryKey: ['oauth', 'tokens'] });
        },
        onError: (error) => {
            logger.error('OAuth Token 刷新失败:', error);
        },
    });
}

/**
 * 获取 OAuth Token 状态 Hook
 *
 * @example
 * const { data: status } = useOAuthTokenStatus(1);
 *
 * if (status?.needs_refresh) {
 *   // Token 需要刷新
 * }
 */
export function useOAuthTokenStatus(id: number | null) {
    return useQuery({
        queryKey: ['oauth', 'status', id],
        queryFn: async () => {
            if (!id) return null;
            return apiClient.get<OAuthTokenStatus>(`/api/v1/oauth/status/${id}`);
        },
        enabled: id !== null,
        refetchInterval: 30000,
    });
}
