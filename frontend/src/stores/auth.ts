import { writable } from 'svelte/store';
import { checkMe, setApiToken, getApiToken, heartbeat } from '../lib/api';

export interface UserInfo {
  id: string;
  username: string;
  role: string;
}

export const user = writable<UserInfo | null>(null);
export const isAuthenticated = writable<boolean>(false);
export const apiToken = writable<string | null>(null);

let heartbeatInterval: ReturnType<typeof setInterval> | null = null;

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

export async function startHeartbeat(): Promise<void> {
  // Send heartbeat immediately on first call
  try {
    await heartbeat();
  } catch {
    // Silently fail — user may have api_token available from this call
  }
  
  // Clear any existing interval before setting a new one
  stopHeartbeat();
  
  // Then every 5 minutes
  heartbeatInterval = setInterval(async () => {
    try {
      await heartbeat();
    } catch {
      // Silently fail — don't disrupt user experience
    }
  }, 5 * 60 * 1000);
}

export function stopHeartbeat(): void {
  if (heartbeatInterval !== null) {
    clearInterval(heartbeatInterval);
    heartbeatInterval = null;
  }
}
