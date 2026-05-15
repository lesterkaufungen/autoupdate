export interface App {
    id: string;
    name: string;
    description: string;
    icon?: string;
    min_version: string;
    created_at: string;
    latest_version?: string;
    last_release_at?: string;
    release_count?: number;
    today_installed?: number;
    total_installed?: number;
}

export interface DashboardStats {
    today_installed: number;
    total_installed: number;
    total_downloads: number;
    total_nodes: number;
}

export interface TrendPoint {
    date: string;
    checks: number;
    downloads: number;
}

export interface Release {
    id: number;
    app_id: string;
    version: string;
    os: string;
    arch: string;
    sha256: string;
    size: number;
    scheduled_at: string | null;
    is_active: boolean;
    created_at: string;
}

export interface Node {
    id: string;
    app_id: string;
    first_seen: string;
    last_seen: string;
    os: string;
    arch: string;
    version: string;
}

export interface Log {
    id: number;
    app_id: string;
    node_id: string;
    ip: string;
    country: string;
    os: string;
    arch: string;
    version: string;
    action: string;
    status: string;
    error_message?: string;
    created_at: string;
    node?: Node;
}

export interface Paginated<T> {
    items: T[];
    total: number;
    page: number;
    limit: number;
}

export interface Stats {
    version: string;
    new: number;
    active: number;
    platforms: Record<string, { new: number; active: number }>;
    countries: Record<string, Record<string, { new: number; active: number }>>;
}

async function request<T>(path: string, options?: RequestInit & { onProgress?: (percent: number) => void }): Promise<T> {
    const adminToken = localStorage.getItem('admin_token') || '';
    
    // If we need progress tracking, we use XMLHttpRequest
    if (options?.onProgress && options.method === 'POST' && options.body instanceof FormData) {
        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();
            xhr.open('POST', path);
            
            xhr.setRequestHeader('X-Admin-Token', adminToken);
            if (options.headers) {
                Object.entries(options.headers).forEach(([key, value]) => {
                    if (key !== 'Content-Type') { // XHR sets this automatically for FormData
                        xhr.setRequestHeader(key, value as string);
                    }
                });
            }

            xhr.upload.onprogress = (event) => {
                if (event.lengthComputable) {
                    const percent = Math.round((event.loaded / event.total) * 100);
                    options.onProgress?.(percent);
                }
            };

            xhr.onload = () => {
                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        if (xhr.status === 204) resolve({} as T);
                        else resolve(JSON.parse(xhr.responseText));
                    } catch (e) {
                        reject(new Error('Failed to parse response'));
                    }
                } else {
                    reject(new Error(xhr.responseText || `Request failed with status ${xhr.status}`));
                }
            };

            xhr.onerror = () => reject(new Error('Network error'));
            xhr.send(options.body as FormData);
        });
    }

    const res = await fetch(path, {
        ...options,
        headers: {
            'X-Admin-Token': adminToken,
            ...options?.headers,
        },
    });

    if (!res.ok) {
        if (res.status === 401) {
            // Handle unauthorized (redirect to login if we had one)
        }
        throw new Error(await res.text());
    }

    if (res.status === 204) return {} as T;
    return res.json();
}

export const api = {
    login: (username: string, password: string) => 
        request<{ token: string }>('/admin/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
            headers: { 'Content-Type': 'application/json' }
        }),
    apps: {
        list: () => request<App[]>('/admin/apps'),
        create: (name: string, description: string, icon?: string) => 
            request<App>('/admin/apps', {
                method: 'POST',
                body: JSON.stringify({ name, description, icon }),
                headers: { 'Content-Type': 'application/json' }
            }),
        update: (id: string, name: string, description: string, icon?: string) =>
            request<App>('/admin/apps', {
                method: 'PUT',
                body: JSON.stringify({ id, name, description, icon }),
                headers: { 'Content-Type': 'application/json' }
            }),
    },
    dashboard: {
        stats: () => request<DashboardStats>('/admin/dashboard/stats'),
        trends: (options?: { appId?: string, days?: number }) => {
            const params = new URLSearchParams();
            if (options?.appId) params.append('app_id', options.appId);
            if (options?.days) params.append('days', options.days.toString());
            return request<TrendPoint[]>(`/admin/dashboard/trends?${params.toString()}`);
        }
    },
    releases: {
        list: (options?: { appId?: string, version?: string, page?: number, limit?: number }) => {
            const params = new URLSearchParams();
            if (options?.appId) params.append('app_id', options.appId);
            if (options?.version) params.append('version', options.version);
            if (options?.page) params.append('page', options.page.toString());
            if (options?.limit) params.append('limit', options.limit.toString());
            return request<Paginated<Release>>(`/admin/releases?${params.toString()}`);
        },
        upload: (formData: FormData, onProgress?: (percent: number) => void) => 
            request<Release[]>('/admin/releases', {
                method: 'POST',
                body: formData,
                onProgress,
            }),
    },
    logs: (options?: { 
        appId?: string, 
        from?: string, 
        to?: string, 
        page?: number, 
        limit?: number,
        action?: string,
        status?: string,
        search?: string,
        sortBy?: string,
        order?: 'asc' | 'desc'
    }) => {
        const params = new URLSearchParams();
        if (options?.appId) params.append('app_id', options.appId);
        if (options?.from) params.append('from', options.from);
        if (options?.to) params.append('to', options.to);
        if (options?.action) params.append('action', options.action);
        if (options?.status) params.append('status', options.status);
        if (options?.search) params.append('search', options.search);
        if (options?.sortBy) params.append('sort_by', options.sortBy);
        if (options?.order) params.append('order', options.order);
        
        params.append('page', (options?.page || 1).toString());
        params.append('limit', (options?.limit || 50).toString());
        
        return request<Paginated<Log>>(`/admin/logs?${params.toString()}`);
    },
    stats: (appId: string, from?: string, to?: string) => {
        const params = new URLSearchParams({ app_id: appId });
        if (from) params.append('from', from);
        if (to) params.append('to', to);
        return request<Stats[]>(`/admin/stats?${params.toString()}`);
    }
};
