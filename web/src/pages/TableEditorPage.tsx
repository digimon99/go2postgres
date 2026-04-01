import { useState, useEffect } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import {
  ArrowLeft,
  Database,
  Table2,
  Plus,
  Search,
  MoreVertical,
  Loader2,
  RefreshCw,
  ChevronLeft,
  ChevronRight,
  Pencil,
  Trash2,
  Key,
} from 'lucide-react'
import { api, ApiException, Instance } from '../lib/api'
import { tableApi, TableInfo, ColumnInfo, TableRow } from '../lib/table-api'
import { useAuth } from '../contexts/AuthContext'
import EditSchemaModal from '../components/table-editor/EditSchemaModal'
import CreateTableModal from '../components/table-editor/CreateTableModal'
import RowModal from '../components/table-editor/RowModal'

const PAGE_SIZES = [25, 50, 100]

export default function TableEditorPage() {
  const { instanceId } = useParams<{ instanceId: string }>()
  const navigate = useNavigate()
  const { logout } = useAuth()

  // Instance info
  const [instance, setInstance] = useState<Instance | null>(null)
  const [instanceLoading, setInstanceLoading] = useState(true)

  // Tables
  const [tables, setTables] = useState<TableInfo[]>([])
  const [tablesLoading, setTablesLoading] = useState(true)
  const [selectedTable, setSelectedTable] = useState<string | null>(null)
  const [tableSearch, setTableSearch] = useState('')

  // Schema
  const [columns, setColumns] = useState<ColumnInfo[]>([])
  const [schemaLoading, setSchemaLoading] = useState(false)

  // Data
  const [rows, setRows] = useState<TableRow[]>([])
  const [totalRows, setTotalRows] = useState(0)
  const [dataLoading, setDataLoading] = useState(false)

  // Pagination & Filtering
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(25)
  const [sortColumn, setSortColumn] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [filters, setFilters] = useState<{ column: string; operator: string; value: string }[]>([])

  // Selection
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set())

  // Modals
  const [showCreateTable, setShowCreateTable] = useState(false)
  const [editSchemaTable, setEditSchemaTable] = useState<string | null>(null)
  const [tableMenuOpen, setTableMenuOpen] = useState<string | null>(null)

  // Inline editing
  const [editingCell, setEditingCell] = useState<{ row: number; col: string } | null>(null)
  const [editValue, setEditValue] = useState('')

  // Row modal
  const [rowModalMode, setRowModalMode] = useState<'insert' | 'edit' | null>(null)
  const [editingRow, setEditingRow] = useState<TableRow | null>(null)

  // Load instance info
  useEffect(() => {
    if (!instanceId) return
    loadInstance()
  }, [instanceId])

  // Load tables when instance is ready
  useEffect(() => {
    if (instance) {
      loadTables()
    }
  }, [instance])

  // Load data when table is selected
  useEffect(() => {
    if (selectedTable) {
      loadSchema()
      loadData()
    }
  }, [selectedTable, page, pageSize, sortColumn, sortDirection])

  async function loadInstance() {
    try {
      setInstanceLoading(true)
      const data = await api.getInstance(instanceId!)
      setInstance(data)
    } catch (err) {
      if (err instanceof ApiException && err.status === 401) {
        logout()
        navigate('/signin')
      } else {
        console.error('Failed to load instance:', err)
      }
    } finally {
      setInstanceLoading(false)
    }
  }

  async function loadTables() {
    try {
      setTablesLoading(true)
      const data = await tableApi.listTables(instanceId!)
      setTables(data)
      // Auto-select first table if none selected
      if (data.length > 0 && !selectedTable) {
        setSelectedTable(data[0].table_name)
      }
    } catch (err) {
      console.error('Error loading tables:', err)
    } finally {
      setTablesLoading(false)
    }
  }

  async function loadSchema() {
    if (!selectedTable) return
    try {
      setSchemaLoading(true)
      const data = await tableApi.getTableSchema(instanceId!, selectedTable)
      setColumns(data)
    } catch (err) {
      console.error('Error loading schema:', err)
    } finally {
      setSchemaLoading(false)
    }
  }

  async function loadData() {
    if (!selectedTable) return
    try {
      setDataLoading(true)
      const data = await tableApi.getTableRows(instanceId!, selectedTable, {
        page,
        pageSize,
        sortColumn: sortColumn || undefined,
        sortDirection,
        filters: filters.length > 0 ? filters : undefined,
      })
      setRows(data.rows)
      setTotalRows(data.total)
    } catch (err) {
      console.error('Error loading data:', err)
      setRows([])
      setTotalRows(0)
    } finally {
      setDataLoading(false)
    }
  }

  function handleTableSelect(tableName: string) {
    setSelectedTable(tableName)
    setSelectedRows(new Set())
    setPage(1)
    setFilters([])
    setSortColumn(null)
    setEditingCell(null)
  }

  function handleSort(column: string) {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortColumn(column)
      setSortDirection('asc')
    }
    setPage(1)
  }

  async function handleDeleteTable(tableName: string) {
    if (!confirm(`Are you sure you want to drop table "${tableName}"? This action cannot be undone.`)) {
      return
    }
    try {
      await tableApi.dropTable(instanceId!, tableName)
      await loadTables()
      if (selectedTable === tableName) {
        setSelectedTable(tables.length > 1 ? tables.find(t => t.table_name !== tableName)?.table_name || null : null)
      }
    } catch (err) {
      alert('Failed to drop table')
    }
    setTableMenuOpen(null)
  }

  async function handleDeleteRows() {
    if (selectedRows.size === 0) return
    if (!confirm(`Delete ${selectedRows.size} selected row(s)?`)) return
    
    const primaryKey = columns.find(c => c.is_primary)?.column_name
    if (!primaryKey) {
      alert('Cannot delete: no primary key found')
      return
    }

    try {
      for (const pkValue of selectedRows) {
        await tableApi.deleteRow(instanceId!, selectedTable!, primaryKey, pkValue)
      }
      setSelectedRows(new Set())
      await loadData()
    } catch (err) {
      alert('Failed to delete rows')
    }
  }

  async function handleCellEdit(rowIndex: number, column: string, newValue: string) {
    const row = rows[rowIndex]
    const primaryKey = columns.find(c => c.is_primary)?.column_name
    if (!primaryKey) return

    const pkValue = row[primaryKey]
    try {
      await tableApi.updateRow(instanceId!, selectedTable!, primaryKey, pkValue, { [column]: newValue })
      await loadData()
    } catch (err) {
      alert('Failed to update row')
    }
    setEditingCell(null)
  }

  const filteredTables = tables.filter(t => 
    t.table_name.toLowerCase().includes(tableSearch.toLowerCase())
  )

  const totalPages = Math.ceil(totalRows / pageSize)

  if (instanceLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50">
        <Loader2 className="w-8 h-8 animate-spin text-primary-500" />
      </div>
    )
  }

  if (!instance) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50">
        <div className="text-center">
          <h2 className="text-xl font-semibold text-slate-900 mb-2">Instance not found</h2>
          <Link to="/dashboard" className="text-primary-600 hover:underline">Back to Dashboard</Link>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      {/* Header */}
      <header className="bg-white border-b border-slate-200 flex-shrink-0">
        <div className="px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-14">
            <div className="flex items-center space-x-4">
              <Link
                to="/dashboard"
                className="flex items-center space-x-2 text-slate-600 hover:text-slate-900"
              >
                <ArrowLeft className="w-4 h-4" />
                <span className="text-sm">Dashboard</span>
              </Link>
              <div className="h-6 w-px bg-slate-200" />
              <div className="flex items-center space-x-2">
                <Database className="w-4 h-4 text-primary-500" />
                <span className="font-medium text-slate-900">{instance.database_name}</span>
                <span className="text-slate-400">/</span>
                <Table2 className="w-4 h-4 text-slate-400" />
                <span className="text-slate-600">Table Editor</span>
              </div>
            </div>
            <button
              onClick={() => { loadTables(); if (selectedTable) loadData() }}
              className="p-2 text-slate-600 hover:text-slate-900 hover:bg-slate-100 rounded-lg"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left Panel - Table List */}
        <div className="w-64 bg-white border-r border-slate-200 flex flex-col">
          {/* Search */}
          <div className="p-3 border-b border-slate-100">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <input
                type="text"
                placeholder="Search tables..."
                value={tableSearch}
                onChange={(e) => setTableSearch(e.target.value)}
                className="w-full pl-9 pr-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
            </div>
          </div>

          {/* Table List */}
          <div className="flex-1 overflow-y-auto p-2">
            {tablesLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="w-5 h-5 animate-spin text-slate-400" />
              </div>
            ) : filteredTables.length === 0 ? (
              <div className="text-center py-8 text-slate-500 text-sm">
                {tables.length === 0 ? 'No tables yet' : 'No matching tables'}
              </div>
            ) : (
              <div className="space-y-1">
                {filteredTables.map((table) => (
                  <div
                    key={table.table_name}
                    className={`group flex items-center justify-between px-3 py-2 rounded-lg cursor-pointer transition-colors ${
                      selectedTable === table.table_name
                        ? 'bg-primary-50 text-primary-700'
                        : 'hover:bg-slate-50 text-slate-700'
                    }`}
                    onClick={() => handleTableSelect(table.table_name)}
                  >
                    <div className="flex items-center space-x-2 min-w-0">
                      <Table2 className="w-4 h-4 flex-shrink-0" />
                      <span className="text-sm font-medium truncate">{table.table_name}</span>
                    </div>
                    <div className="relative">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          setTableMenuOpen(tableMenuOpen === table.table_name ? null : table.table_name)
                        }}
                        className="p-1 rounded opacity-0 group-hover:opacity-100 hover:bg-slate-200 transition-opacity"
                      >
                        <MoreVertical className="w-4 h-4" />
                      </button>
                      {tableMenuOpen === table.table_name && (
                        <div className="absolute right-0 top-full mt-1 w-40 bg-white border border-slate-200 rounded-lg shadow-lg z-10 py-1">
                          <button
                            onClick={(e) => {
                              e.stopPropagation()
                              setEditSchemaTable(table.table_name)
                              setTableMenuOpen(null)
                            }}
                            className="w-full flex items-center space-x-2 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
                          >
                            <Pencil className="w-4 h-4" />
                            <span>Edit Schema</span>
                          </button>
                          <button
                            onClick={(e) => {
                              e.stopPropagation()
                              handleDeleteTable(table.table_name)
                            }}
                            className="w-full flex items-center space-x-2 px-3 py-2 text-sm text-red-600 hover:bg-red-50"
                          >
                            <Trash2 className="w-4 h-4" />
                            <span>Drop Table</span>
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* New Table Button */}
          <div className="p-3 border-t border-slate-100">
            <button
              onClick={() => setShowCreateTable(true)}
              className="w-full flex items-center justify-center space-x-2 px-3 py-2 border border-dashed border-slate-300 rounded-lg text-sm text-slate-600 hover:border-primary-400 hover:text-primary-600 transition-colors"
            >
              <Plus className="w-4 h-4" />
              <span>New Table</span>
            </button>
          </div>
        </div>

        {/* Main Panel - Data Grid */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {selectedTable ? (
            <>
              {/* Toolbar */}
              <div className="bg-white border-b border-slate-200 px-4 py-3 flex items-center justify-between flex-shrink-0">
                <div className="flex items-center space-x-3">
                  <button
                    onClick={() => { setRowModalMode('insert'); setEditingRow(null) }}
                    className="flex items-center space-x-1 px-3 py-1.5 bg-primary-500 text-white rounded-lg text-sm font-medium hover:bg-primary-600"
                  >
                    <Plus className="w-4 h-4" />
                    <span>Insert Row</span>
                  </button>
                  {selectedRows.size > 0 && (
                    <button
                      onClick={handleDeleteRows}
                      className="flex items-center space-x-1 px-3 py-1.5 bg-red-500 text-white rounded-lg text-sm font-medium hover:bg-red-600"
                    >
                      <Trash2 className="w-4 h-4" />
                      <span>Delete ({selectedRows.size})</span>
                    </button>
                  )}
                </div>
                <div className="text-sm text-slate-500">
                  {totalRows.toLocaleString()} row{totalRows !== 1 ? 's' : ''}
                </div>
              </div>

              {/* Data Grid */}
              <div className="flex-1 overflow-auto">
                {dataLoading || schemaLoading ? (
                  <div className="flex items-center justify-center h-full">
                    <Loader2 className="w-6 h-6 animate-spin text-slate-400" />
                  </div>
                ) : columns.length === 0 ? (
                  <div className="flex items-center justify-center h-full text-slate-500">
                    No columns in this table
                  </div>
                ) : (
                  <table className="w-full text-sm">
                    <thead className="bg-slate-50 sticky top-0">
                      <tr>
                        <th className="w-10 px-3 py-2 border-b border-slate-200">
                          <input
                            type="checkbox"
                            checked={selectedRows.size === rows.length && rows.length > 0}
                            onChange={(e) => {
                              if (e.target.checked) {
                                const primaryKey = columns.find(c => c.is_primary)?.column_name
                                if (primaryKey) {
                                  setSelectedRows(new Set(rows.map(r => String(r[primaryKey]))))
                                }
                              } else {
                                setSelectedRows(new Set())
                              }
                            }}
                            className="rounded border-slate-300"
                          />
                        </th>
                        {columns.map((col) => (
                          <th
                            key={col.column_name}
                            className="px-3 py-2 text-left font-medium text-slate-700 border-b border-slate-200 cursor-pointer hover:bg-slate-100"
                            onClick={() => handleSort(col.column_name)}
                          >
                            <div className="flex items-center space-x-1">
                              {col.is_primary && <Key className="w-3 h-3 text-amber-500" />}
                              <span>{col.column_name}</span>
                              {sortColumn === col.column_name && (
                                <span className="text-primary-500">
                                  {sortDirection === 'asc' ? '↑' : '↓'}
                                </span>
                              )}
                            </div>
                            <div className="text-xs font-normal text-slate-400">{col.data_type}</div>
                          </th>
                        ))}
                        <th className="w-12 px-3 py-2 border-b border-slate-200"></th>
                      </tr>
                    </thead>
                    <tbody>
                      {rows.map((row, rowIndex) => {
                        const primaryKey = columns.find(c => c.is_primary)?.column_name
                        const pkValue = primaryKey ? String(row[primaryKey]) : String(rowIndex)
                        return (
                          <tr
                            key={pkValue}
                            className={`border-b border-slate-100 hover:bg-slate-50 ${
                              selectedRows.has(pkValue) ? 'bg-primary-50' : ''
                            }`}
                          >
                            <td className="px-3 py-2">
                              <input
                                type="checkbox"
                                checked={selectedRows.has(pkValue)}
                                onChange={(e) => {
                                  const next = new Set(selectedRows)
                                  if (e.target.checked) {
                                    next.add(pkValue)
                                  } else {
                                    next.delete(pkValue)
                                  }
                                  setSelectedRows(next)
                                }}
                                className="rounded border-slate-300"
                              />
                            </td>
                            {columns.map((col) => {
                              const value = row[col.column_name]
                              const isEditing = editingCell?.row === rowIndex && editingCell?.col === col.column_name
                              const displayValue = value === null ? (
                                <span className="text-slate-400 italic">NULL</span>
                              ) : typeof value === 'object' ? (
                                JSON.stringify(value)
                              ) : (
                                String(value)
                              )

                              return (
                                <td
                                  key={col.column_name}
                                  className="px-3 py-2 max-w-xs truncate"
                                  onDoubleClick={() => {
                                    if (!col.is_primary) {
                                      setEditingCell({ row: rowIndex, col: col.column_name })
                                      setEditValue(value === null ? '' : String(value))
                                    }
                                  }}
                                >
                                  {isEditing ? (
                                    <input
                                      type="text"
                                      value={editValue}
                                      onChange={(e) => setEditValue(e.target.value)}
                                      onBlur={() => handleCellEdit(rowIndex, col.column_name, editValue)}
                                      onKeyDown={(e) => {
                                        if (e.key === 'Enter') handleCellEdit(rowIndex, col.column_name, editValue)
                                        if (e.key === 'Escape') setEditingCell(null)
                                      }}
                                      className="w-full px-1 py-0.5 border border-primary-500 rounded text-sm focus:outline-none"
                                      autoFocus
                                    />
                                  ) : (
                                    displayValue
                                  )}
                                </td>
                              )
                            })}
                            <td className="px-3 py-2 text-right">
                              <button
                                onClick={() => {
                                  setEditingRow(row)
                                  setRowModalMode('edit')
                                }}
                                className="p-1 text-slate-400 hover:text-primary-600 hover:bg-slate-100 rounded"
                                title="Edit row"
                              >
                                <Pencil className="w-4 h-4" />
                              </button>
                            </td>
                          </tr>
                        )
                      })}
                      {rows.length === 0 && (
                        <tr>
                          <td colSpan={columns.length + 2} className="px-3 py-8 text-center text-slate-500">
                            No data in this table
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                )}
              </div>

              {/* Pagination */}
              <div className="bg-white border-t border-slate-200 px-4 py-3 flex items-center justify-between flex-shrink-0">
                <div className="flex items-center space-x-2 text-sm text-slate-600">
                  <span>Rows per page:</span>
                  <select
                    value={pageSize}
                    onChange={(e) => { setPageSize(Number(e.target.value)); setPage(1) }}
                    className="border border-slate-200 rounded px-2 py-1"
                  >
                    {PAGE_SIZES.map((size) => (
                      <option key={size} value={size}>{size}</option>
                    ))}
                  </select>
                </div>
                <div className="flex items-center space-x-4">
                  <span className="text-sm text-slate-600">
                    Page {page} of {totalPages || 1}
                  </span>
                  <div className="flex items-center space-x-1">
                    <button
                      onClick={() => setPage(p => Math.max(1, p - 1))}
                      disabled={page <= 1}
                      className="p-1 rounded hover:bg-slate-100 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <ChevronLeft className="w-5 h-5" />
                    </button>
                    <button
                      onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                      disabled={page >= totalPages}
                      className="p-1 rounded hover:bg-slate-100 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      <ChevronRight className="w-5 h-5" />
                    </button>
                  </div>
                </div>
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-slate-500">
              {tables.length === 0 ? (
                <div className="text-center">
                  <Table2 className="w-12 h-12 mx-auto mb-4 text-slate-300" />
                  <p className="mb-4">No tables in this database</p>
                  <button
                    onClick={() => setShowCreateTable(true)}
                    className="px-4 py-2 bg-primary-500 text-white rounded-lg text-sm font-medium hover:bg-primary-600"
                  >
                    Create your first table
                  </button>
                </div>
              ) : (
                'Select a table from the sidebar'
              )}
            </div>
          )}
        </div>
      </div>

      {/* Modals */}
      {showCreateTable && (
        <CreateTableModal
          instanceId={instanceId!}
          onClose={() => setShowCreateTable(false)}
          onCreated={() => {
            setShowCreateTable(false)
            loadTables()
          }}
        />
      )}

      {editSchemaTable && (
        <EditSchemaModal
          instanceId={instanceId!}
          tableName={editSchemaTable}
          onClose={() => setEditSchemaTable(null)}
          onSaved={() => {
            setEditSchemaTable(null)
            loadTables()
            if (selectedTable === editSchemaTable) {
              loadSchema()
              loadData()
            }
          }}
        />
      )}

      {rowModalMode && selectedTable && (
        <RowModal
          instanceId={instanceId!}
          tableName={selectedTable}
          columns={columns}
          mode={rowModalMode}
          editRow={editingRow}
          onClose={() => {
            setRowModalMode(null)
            setEditingRow(null)
          }}
          onSaved={() => {
            setRowModalMode(null)
            setEditingRow(null)
            loadData()
          }}
        />
      )}

      {/* Click outside handler for table menu */}
      {tableMenuOpen && (
        <div
          className="fixed inset-0 z-0"
          onClick={() => setTableMenuOpen(null)}
        />
      )}
    </div>
  )
}
