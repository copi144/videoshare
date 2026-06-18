import { writable } from 'svelte/store';

export interface UserInfo {
  id: string;
  username: string;
  role: string;
}

export interface PageInfo {
  name: string;
  params?: Record<string, string>;
}

export const user = writable<UserInfo | null>(null);
export const isAuthenticated = writable<boolean>(false);
export const page = writable<PageInfo>({ name: 'loading' });

// Simple route parser
export function navigate(hash: string) {
  // Remove leading /
  const path = hash.startsWith('/') ? hash : '/' + hash;
  
  // Match routes
  const loginMatch = path.match(/^\/login$/);
  const adminMatch = path.match(/^\/admin\/?$/);
  const shareMatch = path.match(/^\/s\/([^\/]+)\/?$/);
  const watchMatch = path.match(/^\/s\/([^\/]+)\/watch\/?$/);
  const categoriesMatch = path.match(/^\/admin\/categories\/?$/);
  const playlistsMatch = path.match(/^\/admin\/playlists\/?$/);
  const usersMatch = path.match(/^\/admin\/users\/?$/);
  
  if (loginMatch) {
    page.set({ name: 'login' });
  } else if (adminMatch) {
    page.set({ name: 'admin' });
  } else if (shareMatch) {
    page.set({ name: 'share', params: { id: shareMatch[1] } });
  } else if (watchMatch) {
    page.set({ name: 'watch', params: { id: watchMatch[1] } });
  } else if (categoriesMatch) {
    page.set({ name: 'categories' });
  } else if (playlistsMatch) {
    page.set({ name: 'playlists' });
  } else if (usersMatch) {
    page.set({ name: 'users' });
  } else {
    page.set({ name: 'notfound' });
  }

  // Update window hash (only if different, to avoid hashchange loop)
  const fullHash = path;
  const currentHash = window.location.hash.slice(1) || '/';
  if (fullHash !== currentHash) {
    window.location.hash = fullHash;
  }
}

// API endpoint for checking auth
const API_BASE = '';

export async function checkAuth(): Promise<void> {
  try {
    const res = await fetch(`${API_BASE}/api/me`, { credentials: 'same-origin' });
    if (res.ok) {
      const data = await res.json();
      if (data.authenticated) {
        user.set(data.user);
        isAuthenticated.set(true);
        return;
      }
    }
  } catch (e) {
    // Not authenticated
  }
  user.set(null);
  isAuthenticated.set(false);
}
