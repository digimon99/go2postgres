import { useState, useEffect } from 'react'
import { X, Loader2, Save } from 'lucide-react'
import { tableApi, ColumnInfo, TableRow } from '../../lib/table-api'

interface RowModalProps {
  instanceId: string
  tableName: string
  columns: ColumnInfo[]
  mode: 'insert' | 'edit'
  editRow?: TableRow | null // The row data when editing
  onClose: () => void
  onSaved: () => void
}

export default function RowModal({
  instanceId,
  tableName,
  columns,
  mode,
  editRow,
  onClose,
  onSaved,
}: RowModalProps) {
  const [values, setValues] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Initialize values
  useEffect(() => {
    const initialValues: Record<string, string> = {}
    columns.forEach((col) => {
      if (mode === 'edit' && editRow) {
        const val = editRow[col.column_name]
        initialValues[col.column_name] = val === null ? '' : typeof val === 'object' ? JSON.stringify(val) : String(val)
      } else {
        // For insert, set empty string or use default hint
        initialValues[col.column_name] = ''
      }
    })
    setValues(initialValues)
  }, [columns, mode, editRow])

  async function handleSave() {
    setError(null)
    setSaving(true)

    try {
      // Build the data object, converting values to appropriate types
      const data: Record<string, any> = {}
      
      for (const col of columns) {
        const value = values[col.column_name]
        
        // Skip primary key for insert if it has a default (like serial)
        if (mode === 'insert' && col.is_primary && col.column_default && value === '') {
          continue
        }
        
        // Skip empty values for nullable columns
        if (value === '' && col.is_nullable) {
          data[col.column_name] = null
          continue
        }
        
        // Skip empty values for columns with defaults
        if (value === '' && col.column_default) {
          continue
        }

        // Skip primary key when editing (can't change PK)
        if (mode === 'edit' && col.is_primary) {
          continue
        }

        // Parse value based on type
        data[col.column_name] = parseValue(value, col.data_type)
      }

      if (mode === 'insert') {
        await tableApi.insertRow(instanceId, tableName, data)
      } else {
        // Find primary key
        const pkCol = columns.find((c) => c.is_primary)
        if (!pkCol || !editRow) {
          throw new Error('Cannot update: no primary key found')
        }
        const pkValue = editRow[pkCol.column_name]
        await tableApi.updateRow(instanceId, tableName, pkCol.column_name, pkValue, data)
      }

      onSaved()
    } catch (err: any) {
      console.error('Error saving row:', err)
      setError(err.message || 'Failed to save row')
    } finally {
      setSaving(false)
    }
  }

  function parseValue(value: string, dataType: string): any {
    if (value === '') return null
    
    const lowerType = dataType.toLowerCase()
    
    // Integer types
    if (lowerType.includes('int') || lowerType === 'serial' || lowerType === 'bigserial' || lowerType === 'smallserial') {
      const parsed = parseInt(value, 10)
      return isNaN(parsed) ? value : parsed
    }
    
    // Float types
    if (lowerType.includes('float') || lowerType.includes('double') || lowerType === 'real' || lowerType === 'numeric' || lowerType === 'decimal') {
      const parsed = parseFloat(value)
      return isNaN(parsed) ? value : parsed
    }
    
    // Boolean
    if (lowerType === 'boolean' || lowerType === 'bool') {
      return value.toLowerCase() === 'true' || value === '1'
    }
    
    // JSON/JSONB
    if (lowerType === 'json' || lowerType === 'jsonb') {
      try {
        return JSON.parse(value)
      } catch {
        return value
      }
    }
    
    // Arrays
    if (lowerType.includes('[]') || lowerType === 'array') {
      try {
        return JSON.parse(value)
      } catch {
        return value
      }
    }
    
    // Default: return as string
    return value
  }

  function getInputType(dataType: string): string {
    const lowerType = dataType.toLowerCase()
    
    if (lowerType.includes('int') || lowerType === 'serial' || lowerType === 'bigserial' || lowerType === 'smallserial') {
      return 'number'
    }
    if (lowerType.includes('float') || lowerType.includes('double') || lowerType === 'real' || lowerType === 'numeric' || lowerType === 'decimal') {
      return 'number'
    }
    if (lowerType === 'date') {
      return 'date'
    }
    if (lowerType === 'time' || lowerType.startsWith('time ')) {
      return 'time'
    }
    if (lowerType.includes('timestamp')) {
      return 'datetime-local'
    }
    
    return 'text'
  }

  function shouldUseTextarea(dataType: string): boolean {
    const lowerType = dataType.toLowerCase()
    return lowerType === 'text' || lowerType === 'json' || lowerType === 'jsonb' || lowerType.includes('[]')
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-2xl max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200">
          <h2 className="text-lg font-semibold text-slate-900">
            {mode === 'insert' ? 'Insert Row' : 'Edit Row'}
          </h2>
          <button
            onClick={onClose}
            className="p-1 text-slate-400 hover:text-slate-600 rounded"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
              {error}
            </div>
          )}

          <div className="space-y-4">
            {columns.map((col) => {
              const isDisabled = mode === 'edit' && col.is_primary
              const hasDefault = !!col.column_default
              const inputType = getInputType(col.data_type)
              const useTextarea = shouldUseTextarea(col.data_type)

              return (
                <div key={col.column_name}>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    <span>{col.column_name}</span>
                    <span className="ml-2 text-xs text-slate-400">({col.data_type})</span>
                    {col.is_primary && (
                      <span className="ml-2 text-xs text-amber-600 font-normal">Primary Key</span>
                    )}
                    {!col.is_nullable && !col.is_primary && (
                      <span className="ml-1 text-red-500">*</span>
                    )}
                  </label>
                  
                  {useTextarea ? (
                    <textarea
                      value={values[col.column_name] || ''}
                      onChange={(e) => setValues({ ...values, [col.column_name]: e.target.value })}
                      disabled={isDisabled}
                      placeholder={
                        hasDefault 
                          ? `Default: ${col.column_default}` 
                          : col.is_nullable 
                            ? 'NULL' 
                            : ''
                      }
                      rows={3}
                      className={`w-full px-3 py-2 text-sm border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent ${
                        isDisabled ? 'bg-slate-100 text-slate-500 cursor-not-allowed' : 'border-slate-300'
                      }`}
                    />
                  ) : col.data_type.toLowerCase() === 'boolean' || col.data_type.toLowerCase() === 'bool' ? (
                    <select
                      value={values[col.column_name] || ''}
                      onChange={(e) => setValues({ ...values, [col.column_name]: e.target.value })}
                      disabled={isDisabled}
                      className={`w-full px-3 py-2 text-sm border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent ${
                        isDisabled ? 'bg-slate-100 text-slate-500 cursor-not-allowed' : 'border-slate-300'
                      }`}
                    >
                      <option value="">
                        {hasDefault ? `Default: ${col.column_default}` : col.is_nullable ? 'NULL' : 'Select...'}
                      </option>
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </select>
                  ) : (
                    <input
                      type={inputType}
                      value={values[col.column_name] || ''}
                      onChange={(e) => setValues({ ...values, [col.column_name]: e.target.value })}
                      disabled={isDisabled}
                      placeholder={
                        hasDefault 
                          ? `Default: ${col.column_default}` 
                          : col.is_nullable 
                            ? 'NULL' 
                            : ''
                      }
                      step={inputType === 'number' ? 'any' : undefined}
                      className={`w-full px-3 py-2 text-sm border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent ${
                        isDisabled ? 'bg-slate-100 text-slate-500 cursor-not-allowed' : 'border-slate-300'
                      }`}
                    />
                  )}

                  {mode === 'edit' && col.is_primary && (
                    <p className="mt-1 text-xs text-slate-500">Primary key cannot be modified</p>
                  )}
                </div>
              )
            })}
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-slate-200 bg-slate-50">
          <button
            onClick={onClose}
            disabled={saving}
            className="px-4 py-2 text-sm font-medium text-slate-700 hover:text-slate-900 disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center space-x-2 px-4 py-2 bg-primary-500 text-white rounded-lg text-sm font-medium hover:bg-primary-600 disabled:opacity-50"
          >
            {saving ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>Saving...</span>
              </>
            ) : (
              <>
                <Save className="w-4 h-4" />
                <span>{mode === 'insert' ? 'Insert' : 'Save Changes'}</span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
