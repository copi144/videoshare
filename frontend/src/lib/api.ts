const API_BASE = '';

let _apiToken: string | null = null;

export function setApiToken(token: string | null) {
  _apiToken = token;
  if (token) {
    localStorage.setItem('videoshare_api_token', token);
  } else {
    localStorage.removeItem('videoshare_api_token');
  }
}

export function getApiToken(): string | null {
  return _apiToken;
}

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

async function request<T>(method: string, path: string, body?: any): Promise<T> {
  const opts: RequestInit = {
    method,
    headers: { 'Accept': 'application/json' },
    credentials: 'same-origin',
  };
  
  // Add API token if available (not needed for login/share-auth which bootstrap it)
  if (_apiToken && path !== '/api/login' && !path.startsWith('/api/s/')) {
    (opts.headers as Record<string, string>)['Authorization'] = `Bearer ${_apiToken}`;
  }
  
  if (body !== undefined && method !== 'GET') {
    if (body instanceof FormData) {
      opts.body = body;
    } else {
      (opts.headers as Record<string, string>)['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
  }
  
  const res = await fetch(`${API_BASE}${path}`, opts);
  const data = await res.json();

  if (data.ok === false) {
    throw new ApiError(res.status, data.error || 'Request failed');
  }

  if (!res.ok) {
    throw new ApiError(res.status, data.error || `HTTP ${res.status}`);
  }
  
  return data as T;
}

// Auth
export const login = (name: string, totpCode: string) =>
  request<{ok: boolean; redirect?: string; api_token?: string}>('POST', '/api/login', { name, totp_code: totpCode });

export const checkMe = () =>
  request<{authenticated: boolean; user?: {name: string; is_admin: boolean}; api_token?: string}>('GET', '/api/me');

export const logout = () =>
  request<{ok: boolean; redirect?: string}>('POST', '/api/logout');

export const heartbeat = () =>
  request<{ok: boolean}>('POST', '/api/heartbeat');

// Types
export interface Resource {
  id: string;
  title: string;
  filename: string;
  file_size: number;
  content_type: string;
  resource_type?: string;
  views: number;
  banned?: boolean;
  created_at: string;
  updated_at: string;
  uploaded_by: string;
  category_name: string;
  uploaded_username?: string;
}

export interface ResourceDetail extends Resource {
  readme?: string;
}

// Resources
export const listResources = (params?: {limit?: number; offset?: number; category_name?: string; playlist_id?: string; resource_type?: string}) => {
  let path = '/api/resources';
  if (params) {
    const qs = new URLSearchParams();
    if (params.limit !== undefined) qs.set('limit', String(params.limit));
    if (params.offset !== undefined) qs.set('offset', String(params.offset));
    if (params.category_name !== undefined) qs.set('category_name', params.category_name);
    if (params.playlist_id !== undefined) qs.set('playlist_id', params.playlist_id);
    if (params.resource_type !== undefined) qs.set('resource_type', params.resource_type);
    const query = qs.toString();
    if (query) path += `?${query}`;
  }
  return request<{ok: boolean; resources: Resource[]; total: number; limit: number; offset: number}>('GET', path);
};

export const getResource = (id: string) =>
  request<ResourceDetail>('GET', `/api/resources/${id}`);

export const uploadVideo = (formData: FormData) =>
  request<{ok: boolean; redirect?: string}>('POST', '/api/upload', formData);

export const deleteResource = (id: string) =>
  request<{ok: boolean}>('DELETE', `/api/resource/${id}`);

export const retranscode = (id: string) =>
  request<{ok: boolean}>('POST', `/api/resources/${id}/retranscode`);

export const banResource = (id: string) =>
  request<{ok: boolean}>('POST', `/api/resources/${id}/ban`);

export const updateReadme = (resourceId: string, readme: string) =>
  request<{ok: boolean}>('PUT', `/api/resources/${resourceId}/readme`, { readme });

export const createSession = (type: 'user' | 'share' | 'token', data: Record<string, any>) =>
  request<{ok: boolean; redirect?: string; api_token?: string; user?: {name: string; is_admin: boolean}}>('POST', '/api/session', { type, ...data });

// Categories
export const listCategories = (params?: {limit?: number; offset?: number}) => {
  let path = '/api/categories';
  if (params) {
    const qs = new URLSearchParams();
    if (params.limit !== undefined) qs.set('limit', String(params.limit));
    if (params.offset !== undefined) qs.set('offset', String(params.offset));
    const query = qs.toString();
    if (query) path += `?${query}`;
  }
  return request<{categories: any[]; total: number; limit: number; offset: number}>('GET', path);
};

export const createCategory = (name: string, displayName: string, description: string) =>
  request<{ok: boolean; redirect?: string}>('POST', '/api/categories', { name, display_name: displayName, description });

export const deleteCategory = (name: string) =>
  request<{ok: boolean}>('DELETE', `/api/categories/${name}`);

export const assignUploaders = (categoryName: string, userIds: string[]) =>
  request<{ok: boolean; redirect?: string}>('POST', `/api/categories/${categoryName}/uploaders`, { user_ids: userIds });

// Playlists
export const listPlaylists = (params?: {limit?: number; offset?: number; category_name?: string; playlist_type?: string}) => {
  let path = '/api/playlists';
  if (params) {
    const qs = new URLSearchParams();
    if (params.limit !== undefined) qs.set('limit', String(params.limit));
    if (params.offset !== undefined) qs.set('offset', String(params.offset));
    if (params.category_name !== undefined) qs.set('category_name', params.category_name);
    if (params.playlist_type !== undefined) qs.set('playlist_type', params.playlist_type);
    const query = qs.toString();
    if (query) path += `?${query}`;
  }
  return request<{playlists: any[]; total: number; limit: number; offset: number}>('GET', path);
};

export const createPlaylist = (name: string, displayName: string, description: string, categoryName: string, playlistType?: string) =>
  request<{ok: boolean; redirect?: string}>('POST', '/api/playlists', { name, display_name: displayName, description, category_name: categoryName, playlist_type: playlistType });

export const deletePlaylist = (id: string) =>
  request<{ok: boolean}>('DELETE', `/api/playlists/${id}`);

export const addVideoToPlaylist = (playlistId: string, resourceId: string) =>
  request<{ok: boolean; redirect?: string}>('POST', `/api/playlists/${playlistId}/videos`, { resource_id: resourceId });

export const removeVideoFromPlaylist = (playlistId: string, resourceId: string) =>
  request<{ok: boolean}>('DELETE', `/api/playlists/${playlistId}/videos/${resourceId}`);

// Resource Share Links — API: /api/share-resources
export const createShareLink = (resourceId: string, expiresInMinutes: number) =>
  request<{ok: boolean; url: string; password: string; expires_at: string}>('POST', '/api/share-resources', { resource_id: resourceId, expires_in_minutes: expiresInMinutes });

export const listShareLinks = (resourceId: string) =>
  request<{share_links: Array<{resource_id: string; password: string; expires_at: string | null; created_by: string; created_at: string}>}>('GET', `/api/share-resources?resource_id=${resourceId}`);

export const deleteShareLink = (resourceId: string, password: string) =>
  request<{ok: boolean}>('DELETE', `/api/share-resources/${encodeURIComponent(resourceId)}/${encodeURIComponent(password)}`);

// Category/Playlist Share Links — API: /api/share-links
export const createTargetShareLink = (targetType: string, targetId: string, expiresInMinutes: number) =>
  request<{ok: boolean; url: string; id: string; password: string; target_type: string; target_id: string; expires_at: string}>('POST', '/api/share-links', { target_type: targetType, target_id: targetId, expires_in_minutes: expiresInMinutes });

export const listTargetShareLinks = (targetType: string, targetId: string) =>
  request<{share_links: Array<{id: string; target_type: string; target_id: string; expires_at: string | null; created_by: string; created_at: string}>}>('GET', `/api/share-links?target_type=${targetType}&target_id=${targetId}`);

export const deleteTargetShareLink = (id: string) =>
  request<{ok: boolean}>('DELETE', `/api/share-links/${id}`);

// Auth for /#/s/{id}/{password} links
export const authenticateShareLink = (id: string, password: string) =>
  request<{ok: boolean; redirect: string; target_type: string; target_id: string}>('POST', `/api/share-links/${id}/auth`, { password });

// Users
export const createUser = (name: string, isAdmin: boolean, displayName: string) =>
  request<{ok: boolean; totp_secret: string; totp_uri: string; qr_image: string; redirect?: string}>('POST', '/api/users', { name, is_admin: isAdmin, display_name: displayName });

export const listUsers = () =>
  request<{users: Array<{name: string; display_name: string; is_admin: boolean; created_at: string}>}>('GET', '/api/users');

export const deleteUser = (name: string) =>
  request<{ok: boolean}>('DELETE', `/api/users/${name}`);

export const resetTOTP = (name: string) =>
  request<{ok: boolean; totp_secret: string; totp_uri: string; qr_image: string}>('POST', `/api/users/${name}/reset-totp`);

// Me

// Restore API token from localStorage on module load
const saved = localStorage.getItem('videoshare_api_token');
if (saved) {
  setApiToken(saved);
}
