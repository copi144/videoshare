const API_BASE = '';

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
  request<{ok: boolean; redirect?: string}>('POST', '/api/login', { username, totp_code: totpCode });

export const logout = () =>
  request<{ok: boolean; redirect?: string}>('POST', '/api/logout');

// Types
export interface Resource {
  id: string;
  title: string;
  filename: string;
  file_size: number;
  content_type: string;
  views: number;
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
export const listResources = () =>
  request<{ok: boolean; resources: Resource[]}>('GET', '/api/resources');

export const getResource = (id: string) =>
  request<ResourceDetail>('GET', `/api/resources/${id}`);

export const uploadVideo = (formData: FormData) =>
  request<{ok: boolean; redirect?: string}>('POST', '/api/upload', formData);

export const deleteResource = (id: string) =>
  request<{ok: boolean}>('DELETE', `/api/resource/${id}`);

export const updateReadme = (resourceId: string, readme: string) =>
  request<{ok: boolean}>('PUT', `/api/resources/${resourceId}/readme`, { readme });

export const shareAuth = (id: string, password: string) =>
  request<{ok: boolean; redirect?: string}>('POST', `/api/s/${id}/auth`, { password });

// Categories
export const listCategories = () =>
  request<{categories: any[]}>('GET', '/api/categories');

export const createCategory = (name: string, description: string) =>
  request<{ok: boolean; redirect?: string}>('POST', '/api/categories', { name, description });

export const deleteCategory = (id: string) =>
  request<{ok: boolean}>('DELETE', `/api/categories/${id}`);

export const assignUploaders = (categoryId: string, userIds: string[]) =>
  request<{ok: boolean; redirect?: string}>('POST', `/api/categories/${categoryId}/uploaders`, { user_ids: userIds });

// Playlists
export const listPlaylists = () =>
  request<{playlists: any[]}>('GET', '/api/playlists');

export const createPlaylist = (name: string, description: string, categoryId: string) =>
  request<{ok: boolean; redirect?: string}>('POST', '/api/playlists', { name, description, category_id: categoryId });

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
export const checkMe = () =>
  request<{authenticated: boolean; user?: {id: string; username: string; role: string}}>('GET', '/api/me');
