---
name: react-frontend
description: Use when writing or editing React/TypeScript frontend code — pages, components, Axios API client, Zustand store, Ant Design usage, or Vite config. Do NOT trigger for Go backend files.
---

# React Frontend Conventions

## Page → Route Mapping (1-to-1, no exceptions)

| File | Route | Purpose |
|------|-------|---------|
| `pages/Dashboard.tsx` | `/` | Scan summary, exposure counts |
| `pages/Ports.tsx` | `/ports` | Filtered port list |
| `pages/Iptables.tsx` | `/iptables` | Rules by chain |
| `pages/Nft.tsx` | `/nft` | nftables raw + parsed |
| `pages/Firewall.tsx` | `/firewall` | Add/delete custom rules |
| `pages/Login.tsx` | `/login` | Auth form |

## API Client — only use this, never fetch() directly

```typescript
// src/api/client.ts
import axios from 'axios';

const client = axios.create({ baseURL: '/api/v1' });

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('sgw_token');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

client.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('sgw_token');
      window.location.href = '/login';
    }
    return Promise.reject(err);
  }
);

export default client;
```

## Auth Store (Zustand)

```typescript
// src/store/auth.ts
import { create } from 'zustand';

interface AuthState {
  token: string | null;
  setToken: (token: string) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem('sgw_token'),
  setToken: (t) => { localStorage.setItem('sgw_token', t); set({ token: t }); },
  logout: () => { localStorage.removeItem('sgw_token'); set({ token: null }); },
}));
```

## TypeScript Rules

- Strict mode enabled — no `any`, no `@ts-ignore`
- Named exports only — no `export default` for components
- Response types always defined:

```typescript
interface ApiResponse<T> { code: number; message: string; data: T; }

interface PortEntry {
  protocol: 'tcp' | 'udp';
  local_addr: string;
  port: number;
  process_name: string;
  pid: number;
  exposure_level: 'public' | 'private' | 'loopback' | 'specific';
  source_type: 'docker' | 'system' | 'user';
}
```

## Ant Design — sole UI library

Exposure level Tag colors: `public`→red, `private`→orange, `loopback`→green, `specific`→blue  
Source type Tag colors: `docker`→cyan, `system`→default, `user`→purple

Use `message.success` / `message.error` for feedback.  
Use `Modal.confirm` for destructive actions (rule deletion).

## Vite Config (dev proxy)

```typescript
// frontend/vite.config.ts
export default defineConfig({
  plugins: [react()],
  server: { proxy: { '/api': 'http://localhost:8080' } },
  build: { outDir: 'dist' },  // Go embeds this path
});
```

## Rules

- All API calls → `src/api/client.ts` only
- No mixing UI libraries — Ant Design only
- No `export default` for components
- Token in `localStorage` key `sgw_token`
