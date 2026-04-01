import { useState, useEffect } from 'react'
import { X, Plus, Trash2, Settings, Loader2, ChevronDown } from 'lucide-react'
import { tableApi, CreateTableColumn } from '../../lib/table-api'
import { POSTGRES_TYPES } from './postgres-types'

interface EditSchemaModalProps {
  instanceId: string
  tableName: string
  onClose: () => void
  onSaved: () => void
}

interface EditableColumn extends CreateTableColumn {
  original_name?: string
  is_new?: boolean
  is_deleted?: boolean
}

export default function EditSchemaModal({ instanceId, tableName, onClose, onSaved }: EditSchemaModalProps) {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [newTableName, setNewTableName] = useState(tableName)
  const [description, setDescription] = useState('')
  const [columns, setColumns] = useState<EditableColumn[]>([])

  const [showOptions, setShowOptions] = useState<number | null>(null)

  useEffect(() => {
    loadSchema()
  }, [])

  async function loadSchema() {
    try {
      setLoading(true)
      const cols = await tableApi.getTableSchema(instanceId, tableName)
      setColumns(cols.map(c => ({
        name: c.column_name,
        type: c.data_type,
        default_value: c.column_default || '',
        is_primary: c.is_primary,
        is_nullable: c.is_nullable,
        is_unique: c.is_unique,
        is_array: c.is_array,
        original_name: c.column_name,
      })))
    } catch (err) {
      setError('Failed to load schema')
    } finally {
      setLoading(false)
    }
  }

  function addColumn() {
    setColumns([...columns, {
      name: '',
      type: 'text',
      default_value: '',
      is_primary: false,
      is_nullable: true,
      is_unique: false,
      is_array: false,
      is_new: true,
    }])
  }

  function updateColumn(index: number, updates: Partial<EditableColumn>) {
    const next = [...columns]
    next[index] = { ...next[index], ...updates }
    setColumns(next)
  }

  function deleteColumn(index: number) {
    const col = columns[index]
    if (col.is_new) {
      setColumns(columns.filter((_, i) => i !== index))
    } else {
      updateColumn(index, { is_deleted: true })
    }
  }

  async function handleSave() {
    setError(null)
    setSaving(true)

    try {
      // Filter out deleted columns for new columns, mark existing for deletion
      const activeColumns = columns.filter(c => !c.is_deleted)
      
      await tableApi.updateTableSchema(instanceId, tableName, {
        new_name: newTableName !== tableName ? newTableName : undefined,
        description: description || undefined,
        columns: activeColumns.map(c => ({
          name: c.name,
          type: c.is_array ? `${c.type}[]` : c.type,
          default_value: c.default_value || undefined,
          is_primary: c.is_primary,
          is_nullable: c.is_nullable,
          is_unique: c.is_unique,
          is_array: c.is_array,
        })),
      })
      onSaved()
    } catch (err: any) {
      setError(err.message || 'Failed to save changes')
    } finally {
      setSaving(false)
    }
  }

  const activeColumns = columns.filter(c => !c.is_deleted)

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-4xl max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200">
          <h2 className="text-lg font-semibold text-slate-900">Edit Table: {tableName}</h2>
          <button onClick={onClose} className="p-1 hover:bg-slate-100 rounded">
            <X className="w-5 h-5 text-slate-500" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-6 h-6 animate-spin text-slate-400" />
            </div>
          ) : (
            <div className="space-y-6">
              {/* Table Info */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">Table Name</label>
                  <input
                    type="text"
                    value={newTableName}
                    onChange={(e) => setNewTableName(e.target.value)}
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
                  {activeColumns.length === 0 ? (
                    <div className="px-3 py-8 text-center text-slate-500 text-sm">
                      No columns. Add one to get started.
                    </div>
                  ) : (
                    <div className="divide-y divide-slate-100">
                      {activeColumns.map((col, idx) => {
                        const originalIndex = columns.findIndex(c => c === col)
                        return (
                          <div key={idx} className="grid grid-cols-[1fr_150px_120px_70px_80px] gap-2 px-3 py-2 items-center">
                            <input
                              type="text"
                              value={col.name}
                              onChange={(e) => updateColumn(originalIndex, { name: e.target.value })}
                              placeholder="column_name"
                              className="px-2 py-1.5 border border-slate-200 rounded text-sm focus:outline-none focus:ring-1 focus:ring-primary-500"
                            />
                            <div className="relative">
                              <select
                                value={col.type.replace('[]', '')}
                                onChange={(e) => updateColumn(originalIndex, { type: e.target.value })}
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
                              onChange={(e) => updateColumn(originalIndex, { default_value: e.target.value })}
                              placeholder="default"
                              className="px-2 py-1.5 border border-slate-200 rounded text-sm focus:outline-none focus:ring-1 focus:ring-primary-500"
                            />
                            <div className="text-center">
                              <input
                                type="radio"
                                name="primary_key"
                                checked={col.is_primary}
                                onChange={() => {
                                  // Clear other primary keys and set this one
                                  setColumns(columns.map((c, i) => ({
                                    ...c,
                                    is_primary: i === originalIndex,
                                  })))
                                }}
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
                                        onChange={(e) => updateColumn(originalIndex, { is_nullable: e.target.checked })}
                                        className="rounded border-slate-300"
                                      />
                                      <span>Is Nullable</span>
                                    </label>
                                    <label className="flex items-center space-x-2 text-sm">
                                      <input
                                        type="checkbox"
                                        checked={col.is_unique}
                                        onChange={(e) => updateColumn(originalIndex, { is_unique: e.target.checked })}
                                        className="rounded border-slate-300"
                                      />
                                      <span>Is Unique</span>
                                    </label>
                                    <label className="flex items-center space-x-2 text-sm">
                                      <input
                                        type="checkbox"
                                        checked={col.is_array}
                                        onChange={(e) => updateColumn(originalIndex, { is_array: e.target.checked })}
                                        className="rounded border-slate-300"
                                      />
                                      <span>Define as Array</span>
                                    </label>
                                  </div>
                                )}
                              </div>
                              <button
                                onClick={() => deleteColumn(originalIndex)}
                                className="p-1.5 hover:bg-red-50 rounded text-slate-500 hover:text-red-500"
                                title="Delete column"
                              >
                                <Trash2 className="w-4 h-4" />
                              </button>
                            </div>
                          </div>
                        )
                      })}
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
          )}
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
            onClick={handleSave}
            disabled={saving || loading}
            className="px-4 py-2 bg-primary-500 text-white rounded-lg font-medium hover:bg-primary-600 disabled:opacity-50"
          >
            {saving ? (
              <span className="flex items-center space-x-2">
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>Saving...</span>
              </span>
            ) : (
              'Save Changes'
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
