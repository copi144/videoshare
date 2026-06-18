import { writable } from 'svelte/store';

export interface UserInfo {
  id: string;
  username: string;
  role: string;
}

export const user = writable<UserInfo | null>(null);
export const isAuthenticated = writable<boolean>(false);

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
