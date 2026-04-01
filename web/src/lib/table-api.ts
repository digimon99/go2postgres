// Types
export interface TableInfo {
  table_name: string
  table_type: string
  row_count?: number
}

export interface ColumnInfo {
  column_name: string
  data_type: string
  is_nullable: boolean
  column_default: string | null
  is_primary: boolean
  is_unique: boolean
  is_array: boolean
}

export interface TableRow {
  [key: string]: any
}

export interface GetRowsParams {
  page?: number
  pageSize?: number
  sortColumn?: string
  sortDirection?: 'asc' | 'desc'
  filters?: { column: string; operator: string; value: string }[]
}

export interface GetRowsResponse {
  rows: TableRow[]
  total: number
}

export interface CreateTableColumn {
  name: string
  type: string
  default_value?: string
  is_primary: boolean
  is_nullable: boolean
  is_unique: boolean
  is_array: boolean
}

export interface CreateTableRequest {
  table_name: string
  description?: string
  columns: CreateTableColumn[]
}

// Helper to get base URL 
const API_BASE = '/api/v1'

async function tableRequest<T>(endpoint: string, options?: RequestInit): Promise<T> {
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

  if (response.status === 401) {
    // Try to refresh token
    const refreshToken = localStorage.getItem('refresh_token')
    if (refreshToken) {
      try {
        const refreshRes = await fetch(`${API_BASE}/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refreshToken }),
        })
        if (refreshRes.ok) {
          const tokens = await refreshRes.json()
          localStorage.setItem('access_token', tokens.access_token)
          localStorage.setItem('refresh_token', tokens.refresh_token)
          // Retry original request
          return tableRequest<T>(endpoint, options)
        }
      } catch {
        // Refresh failed
      }
    }
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    throw new Error('Unauthorized')
  }

  const data = await response.json()

  if (!response.ok) {
    throw new Error(data.error || 'An error occurred')
  }

  return data as T
}

export const tableApi = {
  // List all tables in the database
  listTables: async (instanceId: string): Promise<TableInfo[]> => {
    const res = await tableRequest<{ tables: TableInfo[] }>(`/instances/${instanceId}/tables`)
    return res.tables || []
  },

  // Get table schema (columns)
  getTableSchema: async (instanceId: string, tableName: string): Promise<ColumnInfo[]> => {
    const res = await tableRequest<{ columns: ColumnInfo[] }>(`/instances/${instanceId}/tables/${encodeURIComponent(tableName)}/schema`)
    return res.columns || []
  },

  // Get table rows with pagination/sorting/filtering
  getTableRows: async (instanceId: string, tableName: string, params: GetRowsParams = {}): Promise<GetRowsResponse> => {
    const query = new URLSearchParams()
    if (params.page) query.set('page', String(params.page))
    if (params.pageSize) query.set('page_size', String(params.pageSize))
    if (params.sortColumn) query.set('sort', params.sortColumn)
    if (params.sortDirection) query.set('order', params.sortDirection)
    if (params.filters && params.filters.length > 0) {
      query.set('filters', JSON.stringify(params.filters))
    }

    const queryString = query.toString()
    const url = `/instances/${instanceId}/tables/${encodeURIComponent(tableName)}/rows${queryString ? `?${queryString}` : ''}`
    return tableRequest<GetRowsResponse>(url)
  },

  // Create a new table
  createTable: async (instanceId: string, data: CreateTableRequest): Promise<void> => {
    await tableRequest<{ message: string }>(`/instances/${instanceId}/tables`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  // Update table schema (alter table)
  updateTableSchema: async (
    instanceId: string,
    tableName: string,
    data: { new_name?: string; description?: string; columns?: CreateTableColumn[] }
  ): Promise<void> => {
    await tableRequest<{ message: string }>(`/instances/${instanceId}/tables/${encodeURIComponent(tableName)}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    })
  },

  // Drop a table
  dropTable: async (instanceId: string, tableName: string): Promise<void> => {
    await tableRequest<{ message: string }>(`/instances/${instanceId}/tables/${encodeURIComponent(tableName)}`, {
      method: 'DELETE',
    })
  },

  // Insert a new row
  insertRow: async (instanceId: string, tableName: string, data: Record<string, any>): Promise<void> => {
    await tableRequest<{ message: string }>(`/instances/${instanceId}/tables/${encodeURIComponent(tableName)}/rows`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  // Update a row
  updateRow: async (
    instanceId: string,
    tableName: string,
    primaryKey: string,
    primaryKeyValue: any,
    data: Record<string, any>
  ): Promise<void> => {
    await tableRequest<{ message: string }>(
      `/instances/${instanceId}/tables/${encodeURIComponent(tableName)}/rows`,
      {
        method: 'PATCH',
        body: JSON.stringify({ ...data, _pk_column: primaryKey, _pk_value: primaryKeyValue }),
      }
    )
  },

  // Delete a row
  deleteRow: async (
    instanceId: string,
    tableName: string,
    primaryKey: string,
    primaryKeyValue: any
  ): Promise<void> => {
    await tableRequest<{ message: string }>(
      `/instances/${instanceId}/tables/${encodeURIComponent(tableName)}/rows`,
      {
        method: 'DELETE',
        body: JSON.stringify({ pk_column: primaryKey, pk_value: primaryKeyValue }),
      }
    )
  },
}
