import { writable } from 'svelte/store';
import { checkMe, setApiToken } from '../lib/api';

export interface UserInfo {
  id: string;
  username: string;
  role: string;
}

export const user = writable<UserInfo | null>(null);
export const isAuthenticated = writable<boolean>(false);
export const apiToken = writable<string | null>(null);

export async function checkAuth(): Promise<void> {
  try {
    const data = await checkMe();
    if (data.authenticated && data.user) {
      user.set(data.user);
      isAuthenticated.set(true);
      if ((data as any).api_token) {
        setApiToken((data as any).api_token);
        apiToken.set((data as any).api_token);
      }
    } else {
      user.set(null);
      isAuthenticated.set(false);
    }
  } catch {
    user.set(null);
    isAuthenticated.set(false);
  }
}
