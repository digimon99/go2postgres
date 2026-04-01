// SQL API client for the SQL Editor

const API_BASE = '/api/v1'

export interface QueryRequest {
  sql: string
  params?: any[]
  mode?: 'transaction' | 'pipeline'
}

export interface StatementResult {
  columns?: string[]
  rows?: any[][]
  row_count?: number
  rows_affected?: number
  error?: string
}

export interface QueryResponse {
  results: StatementResult[]
  elapsed_ms: number
}

class ApiException extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'ApiException'
  }
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const accessToken = localStorage.getItem('access_token')
  
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(accessToken && { Authorization: `Bearer ${accessToken}` }),
    ...(options?.headers as Record<string, string>),
  }

  const response = await fetch(`${API_BASE}${url}`, {
    ...options,
    headers,
  })

  const data = await response.json()

  if (!response.ok) {
    const error = data as { error: string }
    throw new ApiException(response.status, error.error || 'An error occurred')
  }

  return data as T
}

export const sqlApi = {
  /**
   * Execute SQL query against an instance
   */
  execute: (instanceId: string, req: QueryRequest): Promise<QueryResponse> =>
    request<QueryResponse>(`/instances/${instanceId}/query`, {
      method: 'POST',
      body: JSON.stringify(req),
    }),
}
