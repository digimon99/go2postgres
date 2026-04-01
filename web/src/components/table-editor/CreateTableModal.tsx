import { useState } from 'react'
import { X, Plus, Trash2, Settings, Loader2, ChevronDown } from 'lucide-react'
import { tableApi, CreateTableColumn } from '../../lib/table-api'
import { POSTGRES_TYPES } from './postgres-types'

interface CreateTableModalProps {
  instanceId: string
  onClose: () => void
  onCreated: () => void
}

interface NewColumn extends CreateTableColumn {
  id: number
}

let columnIdCounter = 0

export default function CreateTableModal({ instanceId, onClose, onCreated }: CreateTableModalProps) {
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [tableName, setTableName] = useState('')
  const [description, setDescription] = useState('')
  const [columns, setColumns] = useState<NewColumn[]>([
    {
      id: ++columnIdCounter,
      name: 'id',
      type: 'serial',
      default_value: '',
      is_primary: true,
      is_nullable: false,
      is_unique: false,
      is_array: false,
    },
  ])

  const [showOptions, setShowOptions] = useState<number | null>(null)

  function addColumn() {
    setColumns([...columns, {
      id: ++columnIdCounter,
      name: '',
      type: 'text',
      default_value: '',
      is_primary: false,
      is_nullable: true,
      is_unique: false,
      is_array: false,
    }])
  }

  function updateColumn(id: number, updates: Partial<NewColumn>) {
    setColumns(columns.map(c => c.id === id ? { ...c, ...updates } : c))
  }

  function deleteColumn(id: number) {
    setColumns(columns.filter(c => c.id !== id))
  }

  function setPrimaryKey(id: number) {
    setColumns(columns.map(c => ({ ...c, is_primary: c.id === id })))
  }

  async function handleCreate() {
    setError(null)

    // Validation
    if (!tableName.trim()) {
      setError('Table name is required')
      return
    }
    if (columns.length === 0) {
      setError('At least one column is required')
      return
    }
    for (const col of columns) {
      if (!col.name.trim()) {
        setError('All columns must have a name')
        return
      }
    }

    setSaving(true)

    try {
      await tableApi.createTable(instanceId, {
        table_name: tableName.trim(),
        description: description.trim() || undefined,
        columns: columns.map(c => ({
          name: c.name.trim(),
          type: c.is_array ? `${c.type}[]` : c.type,
          default_value: c.default_value?.trim() || undefined,
          is_primary: c.is_primary,
          is_nullable: c.is_nullable,
          is_unique: c.is_unique,
          is_array: c.is_array,
        })),
      })
      onCreated()
    } catch (err: any) {
      setError(err.message || 'Failed to create table')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-4xl max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200">
          <h2 className="text-lg font-semibold text-slate-900">Create New Table</h2>
          <button onClick={onClose} className="p-1 hover:bg-slate-100 rounded">
            <X className="w-5 h-5 text-slate-500" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          <div className="space-y-6">
            {/* Table Info */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Table Name <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={tableName}
                  onChange={(e) => setTableName(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, '_'))}
                  placeholder="my_table"
                  className="w-full px-3 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Description (optional)</label>
                <input
                  type="text"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Table description..."
                  className="w-full px-3 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
            </div>

            {/* Columns */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <label className="text-sm font-medium text-slate-700">Columns</label>
                <button
                  onClick={addColumn}
                  className="flex items-center space-x-1 px-2 py-1 text-sm text-primary-600 hover:bg-primary-50 rounded"
                >
                  <Plus className="w-4 h-4" />
                  <span>Add Column</span>
                </button>
              </div>

              <div className="border border-slate-200 rounded-lg overflow-hidden">
                {/* Table Header */}
                <div className="grid grid-cols-[1fr_150px_120px_70px_80px] gap-2 px-3 py-2 bg-slate-50 text-xs font-medium text-slate-500 uppercase">
                  <div>Name</div>
                  <div>Type</div>
                  <div>Default</div>
                  <div className="text-center">Primary</div>
                  <div className="text-center">Actions</div>
                </div>

                {/* Column Rows */}
                {columns.length === 0 ? (
                  <div className="px-3 py-8 text-center text-slate-500 text-sm">
                    No columns. Add one to get started.
                  </div>
                ) : (
                  <div className="divide-y divide-slate-100">
                    {columns.map((col, idx) => (
                      <div key={col.id} className="grid grid-cols-[1fr_150px_120px_70px_80px] gap-2 px-3 py-2 items-center">
                        <input
                          type="text"
                          value={col.name}
                          onChange={(e) => updateColumn(col.id, { name: e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, '_') })}
                          placeholder="column_name"
                          className="px-2 py-1.5 border border-slate-200 rounded text-sm focus:outline-none focus:ring-1 focus:ring-primary-500"
                        />
                        <div className="relative">
                          <select
                            value={col.type}
                            onChange={(e) => updateColumn(col.id, { type: e.target.value })}
                            className="w-full px-2 py-1.5 border border-slate-200 rounded text-sm focus:outline-none focus:ring-1 focus:ring-primary-500 appearance-none bg-white"
                          >
                            {Object.entries(POSTGRES_TYPES).map(([group, types]) => (
                              <optgroup key={group} label={group}>
                                {types.map(t => (
                                  <option key={t} value={t}>{t}</option>
                                ))}
                              </optgroup>
                            ))}
                          </select>
                          <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400 pointer-events-none" />
                        </div>
                        <input
                          type="text"
                          value={col.default_value || ''}
                          onChange={(e) => updateColumn(col.id, { default_value: e.target.value })}
                          placeholder="default"
                          className="px-2 py-1.5 border border-slate-200 rounded text-sm focus:outline-none focus:ring-1 focus:ring-primary-500"
                        />
                        <div className="text-center">
                          <input
                            type="radio"
                            name="primary_key"
                            checked={col.is_primary}
                            onChange={() => setPrimaryKey(col.id)}
                            className="w-4 h-4 text-primary-500"
                          />
                        </div>
                        <div className="flex items-center justify-center space-x-1">
                          <div className="relative">
                            <button
                              onClick={() => setShowOptions(showOptions === idx ? null : idx)}
                              className="p-1.5 hover:bg-slate-100 rounded"
                              title="Extra options"
                            >
                              <Settings className="w-4 h-4 text-slate-500" />
                            </button>
                            {showOptions === idx && (
                              <div className="absolute right-0 top-full mt-1 w-48 bg-white border border-slate-200 rounded-lg shadow-lg z-10 p-3 space-y-2">
                                <label className="flex items-center space-x-2 text-sm">
                                  <input
                                    type="checkbox"
                                    checked={col.is_nullable}
                                    onChange={(e) => updateColumn(col.id, { is_nullable: e.target.checked })}
                                    className="rounded border-slate-300"
                                  />
                                  <span>Is Nullable</span>
                                </label>
                                <label className="flex items-center space-x-2 text-sm">
                                  <input
                                    type="checkbox"
                                    checked={col.is_unique}
                                    onChange={(e) => updateColumn(col.id, { is_unique: e.target.checked })}
                                    className="rounded border-slate-300"
                                  />
                                  <span>Is Unique</span>
                                </label>
                                <label className="flex items-center space-x-2 text-sm">
                                  <input
                                    type="checkbox"
                                    checked={col.is_array}
                                    onChange={(e) => updateColumn(col.id, { is_array: e.target.checked })}
                                    className="rounded border-slate-300"
                                  />
                                  <span>Define as Array</span>
                                </label>
                              </div>
                            )}
                          </div>
                          <button
                            onClick={() => deleteColumn(col.id)}
                            disabled={columns.length === 1}
                            className="p-1.5 hover:bg-red-50 rounded text-slate-500 hover:text-red-500 disabled:opacity-30 disabled:cursor-not-allowed"
                            title="Delete column"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {error && (
              <div className="p-3 bg-red-50 text-red-700 rounded-lg text-sm">
                {error}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end space-x-3 px-6 py-4 border-t border-slate-200">
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-700 hover:bg-slate-100 rounded-lg font-medium"
          >
            Cancel
          </button>
          <button
            onClick={handleCreate}
            disabled={saving}
            className="px-4 py-2 bg-primary-500 text-white rounded-lg font-medium hover:bg-primary-600 disabled:opacity-50"
          >
            {saving ? (
              <span className="flex items-center space-x-2">
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>Creating...</span>
              </span>
            ) : (
              'Create Table'
            )}
          </button>
        </div>
      </div>

      {/* Click outside to close options */}
      {showOptions !== null && (
        <div className="fixed inset-0 z-0" onClick={() => setShowOptions(null)} />
      )}
    </div>
  )
}
