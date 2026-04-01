const API_BASE = '/api/v1'

interface ApiError {
  error: string
  details?: string
}

export class ApiException extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'ApiException'
  }
}

// Session management - allows components to react to session expiry
type SessionExpiredCallback = () => void
let sessionExpiredCallback: SessionExpiredCallback | null = null

export function onSessionExpired(callback: SessionExpiredCallback) {
  sessionExpiredCallback = callback
}

// Token refresh state to prevent concurrent refreshes
let isRefreshing = false
let refreshPromise: Promise<boolean> | null = null

// Attempt to refresh the access token
async function tryRefreshToken(): Promise<boolean> {
  // If already refreshing, wait for that to complete
  if (isRefreshing && refreshPromise) {
    return refreshPromise
  }

  const refreshToken = localStorage.getItem('refresh_token')
  if (!refreshToken) {
    return false
  }

  isRefreshing = true
  refreshPromise = (async () => {
    try {
      const response = await fetch(`${API_BASE}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      })

      if (!response.ok) {
        throw new Error('Refresh failed')
      }

      const data = await response.json()
      localStorage.setItem('access_token', data.access_token)
      localStorage.setItem('refresh_token', data.refresh_token)
      return true
    } catch {
      // Refresh failed - session is truly expired
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      sessionExpiredCallback?.()
      return false
    } finally {
      isRefreshing = false
      refreshPromise = null
    }
  })()

  return refreshPromise
}

// List of endpoints that should not trigger refresh (they are auth endpoints themselves)
const NO_REFRESH_ENDPOINTS = ['/auth/refresh', '/auth/logout', '/auth/otp/send', '/auth/otp/verify']

async function request<T>(endpoint: string, options?: RequestInit, isRetry = false): Promise<T> {
  const token = localStorage.getItem('access_token')
  
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(token && { Authorization: `Bearer ${token}` }),
    ...options?.headers,
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  })

  // Handle 401 - try to refresh token and retry (unless this is already a retry)
  if (response.status === 401 && !isRetry && !NO_REFRESH_ENDPOINTS.includes(endpoint)) {
    const refreshed = await tryRefreshToken()
    if (refreshed) {
      // Retry the original request with new token
      return request<T>(endpoint, options, true)
    }
    // Refresh failed - throw the original error
  }

  const data = await response.json()

  if (!response.ok) {
    const errorData = data as ApiError
    throw new ApiException(response.status, errorData.error || 'An error occurred')
  }

  return data as T
}

// Auth
export interface SendOTPResponse {
  message: string
  email: string
  is_new_user: boolean
  action: string
}

export interface VerifyOTPResponse {
  access_token: string
  refresh_token: string
  token_type: string
  is_new_user: boolean
  user: User
}

export interface User {
  id: string
  email: string
  role: string
  status?: string
  created_at?: string
}

export interface Instance {
  instance_id: string
  project_id: string
  database_name: string
  host: string
  port: number
  username: string
  status: string
  created_at: string
  password?: string
  password_encoded?: string
}

export interface AdminStats {
  total_users: number
  active_users: number
  total_instances: number
  active_instances: number
}

export interface AdminUser {
  id: string
  email: string
  role: string
  status: string
  created_at: string
  instance_count: number
}

export interface AdminInstance {
  instance_id: string
  project_id: string
  database_name: string
  username: string
  host: string
  port: number
  status: string
  created_at: string
  user_id: string
  user_email: string
}

export interface APIKey {
  key_id: string
  name: string
  key_preview: string
  key_type: 'readonly' | 'fullaccess'
  ip_allowlist: string
  is_active: boolean
  last_used_at: string | null
  created_at: string
}

export interface CreateAPIKeyResponse {
  key: string
  key_id: string
  name: string
  key_type: string
  key_preview: string
  created_at: string
}

export const api = {
  // Auth
  sendOTP: (email: string) =>
    request<SendOTPResponse>('/auth/otp/send', {
      method: 'POST',
      body: JSON.stringify({ email }),
    }),

  verifyOTP: (email: string, code: string) =>
    request<VerifyOTPResponse>('/auth/otp/verify', {
      method: 'POST',
      body: JSON.stringify({ email, code }),
    }),

  refresh: (refreshToken: string) =>
    request<{ access_token: string; refresh_token: string }>('/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    }),

  logout: (refreshToken: string) =>
    request<void>('/auth/logout', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    }),

  // User
  getProfile: () => request<User>('/me'),

  // Instances
  listInstances: async () => {
    const res = await request<{ instances: Instance[] }>('/instances')
    return res.instances || []
  },

  createInstance: (dbName: string, projectId: string) =>
    request<Instance>('/instances', {
      method: 'POST',
      body: JSON.stringify({ db_name: dbName, project_id: projectId }),
    }),

  getInstance: (id: string) => request<Instance>(`/instances/${id}`),

  deleteInstance: (id: string) =>
    request<void>(`/instances/${id}`, { method: 'DELETE' }),

  revealPassword: async (id: string) => {
    const res = await request<{ password: string }>(`/instances/${id}/get-db-config`, {
      method: 'POST',
    })
    return res.password
  },

  // Admin
  adminStats: () => request<AdminStats>('/admin/stats'),

  adminListUsers: async () => {
    const res = await request<{ users: AdminUser[] }>('/admin/users')
    return res.users || []
  },

  adminListInstances: async () => {
    const res = await request<{ instances: AdminInstance[] }>('/admin/instances')
    return res.instances || []
  },

  adminUpdateUser: (userId: string, data: { status?: string; role?: string }) =>
    request<void>(`/admin/users/${userId}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),

  adminUpdateInstance: (instanceId: string, data: { status?: string }) =>
    request<void>(`/admin/instances/${instanceId}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),

  // API Keys
  listAPIKeys: async (instanceId: string) => {
    const res = await request<{ keys: APIKey[] }>(`/instances/${instanceId}/keys`)
    return res.keys || []
  },

  createAPIKey: (instanceId: string, name: string, keyType: 'readonly' | 'fullaccess', ipAllowlist?: string) =>
    request<CreateAPIKeyResponse>(`/instances/${instanceId}/keys`, {
      method: 'POST',
      body: JSON.stringify({ name, key_type: keyType, ip_allowlist: ipAllowlist || '[]' }),
    }),

  revokeAPIKey: (keyId: string) =>
    request<void>(`/keys/${keyId}`, { method: 'DELETE' }),
}
