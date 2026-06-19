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
export const login = (username: string, totpCode: string) =>
  request<{ok: boolean; redirect?: string; api_token?: string}>('POST', '/api/login', { username, totp_code: totpCode });

export const checkMe = () =>
  request<{authenticated: boolean; user?: {id: string; username: string; role: string}; api_token?: string}>('GET', '/api/me');

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
  category_id: string;
  uploaded_username?: string;
  category_name?: string;
}

export interface ResourceDetail extends Resource {
  readme?: string;
}

// Resources
export const listResources = (params?: {limit?: number; offset?: number; category_id?: string; playlist_id?: string; resource_type?: string}) => {
  let path = '/api/resources';
  if (params) {
    const qs = new URLSearchParams();
    if (params.limit !== undefined) qs.set('limit', String(params.limit));
    if (params.offset !== undefined) qs.set('offset', String(params.offset));
    if (params.category_id !== undefined) qs.set('category_id', params.category_id);
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

export const shareAuth = (id: string, password: string) =>
  request<{ok: boolean; redirect?: string}>('POST', `/api/s/${id}/auth`, { password });

export const createSession = (type: 'user' | 'share' | 'token', data: Record<string, any>) =>
  request<{ok: boolean; redirect?: string; api_token?: string; user?: {id: string; username: string; role: string}}>('POST', '/api/session', { type, ...data });

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

export const createCategory = (name: string, description: string) =>
  request<{ok: boolean; redirect?: string}>('POST', '/api/categories', { name, description });

export const deleteCategory = (id: string) =>
  request<{ok: boolean}>('DELETE', `/api/categories/${id}`);

export const assignUploaders = (categoryId: string, userIds: string[]) =>
  request<{ok: boolean; redirect?: string}>('POST', `/api/categories/${categoryId}/uploaders`, { user_ids: userIds });

// Playlists
export const listPlaylists = (params?: {limit?: number; offset?: number; category_id?: string; playlist_type?: string}) => {
  let path = '/api/playlists';
  if (params) {
    const qs = new URLSearchParams();
    if (params.limit !== undefined) qs.set('limit', String(params.limit));
    if (params.offset !== undefined) qs.set('offset', String(params.offset));
    if (params.category_id !== undefined) qs.set('category_id', params.category_id);
    if (params.playlist_type !== undefined) qs.set('playlist_type', params.playlist_type);
    const query = qs.toString();
    if (query) path += `?${query}`;
  }
  return request<{playlists: any[]; total: number; limit: number; offset: number}>('GET', path);
};

export const createPlaylist = (name: string, description: string, categoryId: string, playlistType?: string) =>
  request<{ok: boolean; redirect?: string}>('POST', '/api/playlists', { name, description, category_id: categoryId, playlist_type: playlistType });

export const deletePlaylist = (id: string) =>
  request<{ok: boolean}>('DELETE', `/api/playlists/${id}`);

export const addVideoToPlaylist = (playlistId: string, resourceId: string) =>
  request<{ok: boolean; redirect?: string}>('POST', `/api/playlists/${playlistId}/videos`, { resource_id: resourceId });

export const removeVideoFromPlaylist = (playlistId: string, resourceId: string) =>
  request<{ok: boolean}>('DELETE', `/api/playlists/${playlistId}/videos/${resourceId}`);

// Users
export const createUser = (username: string) =>
  request<{ok: boolean; totp_secret: string; totp_uri: string; qr_image: string; redirect?: string}>('POST', '/api/users', { username });

// Me

// Restore API token from localStorage on module load
const saved = localStorage.getItem('videoshare_api_token');
if (saved) {
  setApiToken(saved);
}
