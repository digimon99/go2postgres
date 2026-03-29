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

async function request<T>(endpoint: string, options?: RequestInit): Promise<T> {
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
  id: string
  db_name: string
  host: string
  port: number
  username: string
  status: string
  created_at: string
  password?: string
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
  id: string
  db_name: string
  username: string
  host: string
  port: number
  status: string
  created_at: string
  user_id: string
  user_email: string
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

  createInstance: (dbName: string) =>
    request<Instance>('/instances', {
      method: 'POST',
      body: JSON.stringify({ db_name: dbName }),
    }),

  getInstance: (id: string) => request<Instance>(`/instances/${id}`),

  deleteInstance: (id: string) =>
    request<void>(`/instances/${id}`, { method: 'DELETE' }),

  revealPassword: async (id: string) => {
    const res = await request<{ password: string }>(`/instances/${id}/password`)
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
}
