const HISTORY_KEY = 'videoshare_watch_history';
const SEARCH_KEY = 'videoshare_search_history';
const MAX_WATCH = 50;
const MAX_SEARCH = 20;

export interface WatchRecord {
  id: string;
  title: string;
  watchedAt: string; // ISO timestamp
}

export interface SearchRecord {
  query: string;
  searchedAt: string; // ISO timestamp
}

function safeLocalStorage(): Storage | null {
  try {
    const key = '__videoshare_test__';
    localStorage.setItem(key, '1');
    localStorage.removeItem(key);
    return localStorage;
  } catch {
    return null;
  }
}

function readFromStorage<T>(key: string): T[] {
  const storage = safeLocalStorage();
  if (!storage) return [];

  try {
    const raw = storage.getItem(key);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed as T[];
  } catch {
    return [];
  }
}

function writeToStorage<T>(key: string, data: T[]): void {
  const storage = safeLocalStorage();
  if (!storage) return;

  try {
    storage.setItem(key, JSON.stringify(data));
  } catch {
    // Storage full or unavailable — silently fail
  }
}

export function getWatchHistory(): WatchRecord[] {
  return readFromStorage<WatchRecord>(HISTORY_KEY);
}

export function addWatchHistory(record: WatchRecord): void {
  if (!record.id || !record.watchedAt) return;

  const history = getWatchHistory();
  const filtered = history.filter((r) => r.id !== record.id);
  filtered.unshift(record);
  writeToStorage(HISTORY_KEY, filtered.slice(0, MAX_WATCH));
}

export function clearWatchHistory(): void {
  const storage = safeLocalStorage();
  if (!storage) return;

  try {
    storage.removeItem(HISTORY_KEY);
  } catch {
    // Silently fail
  }
}

export function removeWatchEntry(id: string): void {
  const history = getWatchHistory();
  writeToStorage(
    HISTORY_KEY,
    history.filter((r) => r.id !== id),
  );
}

export function getSearchHistory(): SearchRecord[] {
  return readFromStorage<SearchRecord>(SEARCH_KEY);
}

export function addSearchHistory(query: string): void {
  const trimmed = query.trim();
  if (!trimmed) return;

  const history = getSearchHistory();
  const filtered = history.filter(
    (r) => r.query.toLowerCase() !== trimmed.toLowerCase(),
  );
  filtered.unshift({ query: trimmed, searchedAt: new Date().toISOString() });
  writeToStorage(SEARCH_KEY, filtered.slice(0, MAX_SEARCH));
}

export function clearSearchHistory(): void {
  const storage = safeLocalStorage();
  if (!storage) return;

  try {
    storage.removeItem(SEARCH_KEY);
  } catch {
    // Silently fail
  }
}
