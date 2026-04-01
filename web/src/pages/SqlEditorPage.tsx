import { useState, useEffect, useRef, useCallback } from 'react'
import { Link, useParams, useNavigate } from 'react-router-dom'
import {
  ArrowLeft,
  Database,
  Play,
  Trash2,
  Loader2,
  AlertCircle,
  CheckCircle2,
  Clock,
  Rows3,
  ChevronLeft,
  ChevronRight,
  Copy,
  Check,
  Download,
} from 'lucide-react'
import { api, ApiException, Instance } from '../lib/api'
import { sqlApi, QueryRequest, StatementResult, QueryResponse } from '../lib/sql-api'
import { useAuth } from '../contexts/AuthContext'

const EXAMPLE_QUERIES = [
  { label: 'Show all tables', sql: "SELECT tablename FROM pg_tables WHERE schemaname = 'public'" },
  { label: 'Table row counts', sql: "SELECT relname AS table_name, n_live_tup AS row_count FROM pg_stat_user_tables ORDER BY n_live_tup DESC" },
  { label: 'Database size', sql: "SELECT pg_size_pretty(pg_database_size(current_database())) AS size" },
]

export default function SqlEditorPage() {
  const { instanceId } = useParams<{ instanceId: string }>()
  const navigate = useNavigate()
  const { logout } = useAuth()
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // Instance info
  const [instance, setInstance] = useState<Instance | null>(null)
  const [instanceLoading, setInstanceLoading] = useState(true)

  // SQL Editor state
  const [sql, setSql] = useState('')
  const [mode, setMode] = useState<'transaction' | 'pipeline'>('transaction')
  const [executing, setExecuting] = useState(false)

  // Results
  const [results, setResults] = useState<StatementResult[]>([])
  const [elapsedMs, setElapsedMs] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [activeResultTab, setActiveResultTab] = useState(0)

  // Pagination for results
  const [resultPage, setResultPage] = useState(1)
  const resultsPerPage = 100

  // Copy state
  const [copiedCell, setCopiedCell] = useState<string | null>(null)

  // Load instance info
  useEffect(() => {
    if (!instanceId) return
    loadInstance()
  }, [instanceId])

  async function loadInstance() {
    setInstanceLoading(true)
    try {
      const inst = await api.getInstance(instanceId!)
      setInstance(inst)
    } catch (err) {
      if (err instanceof ApiException) {
        if (err.status === 401) {
          logout()
          navigate('/signin')
        } else if (err.status === 404) {
          navigate('/dashboard')
        }
      }
    } finally {
      setInstanceLoading(false)
    }
  }

  // Execute query
  const executeQuery = useCallback(async () => {
    if (!sql.trim() || !instanceId) return

    setExecuting(true)
    setError(null)
    setResults([])
    setElapsedMs(null)
    setActiveResultTab(0)
    setResultPage(1)

    try {
      const req: QueryRequest = {
        sql: sql.trim(),
        mode,
      }
      const response: QueryResponse = await sqlApi.execute(instanceId, req)
      setResults(response.results)
      setElapsedMs(response.elapsed_ms)
    } catch (err) {
      if (err instanceof ApiException) {
        if (err.status === 401) {
          logout()
          navigate('/signin')
        } else {
          setError(err.message)
        }
      } else {
        setError('An unexpected error occurred')
      }
    } finally {
      setExecuting(false)
    }
  }, [sql, mode, instanceId, logout, navigate])

  // Keyboard shortcut: Ctrl+Enter to run
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault()
        executeQuery()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [executeQuery])

  // Copy cell value
  function copyValue(value: any, cellId: string) {
    const text = value === null ? 'NULL' : String(value)
    navigator.clipboard.writeText(text)
    setCopiedCell(cellId)
    setTimeout(() => setCopiedCell(null), 1500)
  }

  // Export results as CSV
  function exportCsv() {
    const result = results[activeResultTab]
    if (!result || !result.columns || !result.rows) return

    const header = result.columns.join(',')
    const rows = result.rows.map(row =>
      row.map(cell => {
        if (cell === null) return ''
        const str = String(cell)
        if (str.includes(',') || str.includes('"') || str.includes('\n')) {
          return `"${str.replace(/"/g, '""')}"`
        }
        return str
      }).join(',')
    )
    const csv = [header, ...rows].join('\n')
    
    const blob = new Blob([csv], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `query_result_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  // Format cell value for display
  function formatCell(value: any): string {
    if (value === null) return 'NULL'
    if (typeof value === 'object') return JSON.stringify(value)
    return String(value)
  }

  // Get current result for display
  const currentResult = results[activeResultTab]
  const totalResultRows = currentResult?.rows?.length || 0
  const totalResultPages = Math.ceil(totalResultRows / resultsPerPage)
  const paginatedRows = currentResult?.rows?.slice(
    (resultPage - 1) * resultsPerPage,
    resultPage * resultsPerPage
  ) || []

  if (instanceLoading) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-primary-500 animate-spin" />
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      {/* Header */}
      <header className="bg-white border-b border-slate-200 px-4 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <Link
              to="/dashboard"
              className="flex items-center text-slate-500 hover:text-slate-700 transition-colors"
            >
              <ArrowLeft className="w-4 h-4 mr-1" />
              Dashboard
            </Link>
            <div className="h-6 w-px bg-slate-200" />
            <div className="flex items-center space-x-2">
              <Database className="w-5 h-5 text-primary-500" />
              <span className="font-semibold text-slate-900">
                {instance?.database_name || 'Database'}
              </span>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <Link
              to={`/instances/${instanceId}/tables`}
              className="px-3 py-1.5 text-sm text-slate-600 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition-colors"
            >
              Table Editor
            </Link>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <div className="flex-1 flex flex-col p-4 space-y-4">
        {/* SQL Editor Section */}
        <div className="bg-white rounded-lg border border-slate-200 shadow-sm">
          {/* Toolbar */}
          <div className="flex items-center justify-between px-4 py-2 border-b border-slate-200 bg-slate-50 rounded-t-lg">
            <div className="flex items-center space-x-2">
              <button
                onClick={executeQuery}
                disabled={executing || !sql.trim()}
                className="flex items-center space-x-1.5 px-3 py-1.5 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors text-sm font-medium"
              >
                {executing ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <Play className="w-4 h-4" />
                )}
                <span>Run</span>
              </button>
              <span className="text-xs text-slate-400">Ctrl+Enter</span>
              <div className="h-4 w-px bg-slate-200 mx-2" />
              <select
                value={mode}
                onChange={(e) => setMode(e.target.value as 'transaction' | 'pipeline')}
                className="text-sm border border-slate-200 rounded px-2 py-1 bg-white"
              >
                <option value="transaction">Transaction (atomic)</option>
                <option value="pipeline">Pipeline (independent)</option>
              </select>
            </div>
            <div className="flex items-center space-x-2">
              <select
                onChange={(e) => {
                  if (e.target.value) {
                    setSql(e.target.value)
                    e.target.value = ''
                  }
                }}
                className="text-sm border border-slate-200 rounded px-2 py-1 bg-white text-slate-600"
                defaultValue=""
              >
                <option value="" disabled>Example queries...</option>
                {EXAMPLE_QUERIES.map((q, i) => (
                  <option key={i} value={q.sql}>{q.label}</option>
                ))}
              </select>
              <button
                onClick={() => {
                  setSql('')
                  setResults([])
                  setError(null)
                  setElapsedMs(null)
                }}
                className="p-1.5 text-slate-400 hover:text-slate-600 hover:bg-slate-100 rounded transition-colors"
                title="Clear"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* SQL Textarea */}
          <div className="p-0">
            <textarea
              ref={textareaRef}
              value={sql}
              onChange={(e) => setSql(e.target.value)}
              placeholder="Enter your SQL query here...

Example:
SELECT * FROM users LIMIT 10;

-- Multiple statements (in transaction mode):
INSERT INTO logs (message) VALUES ('test');
SELECT COUNT(*) FROM logs;"
              className="w-full h-48 p-4 font-mono text-sm resize-y border-0 focus:ring-0 focus:outline-none bg-slate-900 text-slate-100 rounded-b-lg"
              spellCheck={false}
            />
          </div>
        </div>

        {/* Results Section */}
        <div className="flex-1 bg-white rounded-lg border border-slate-200 shadow-sm flex flex-col min-h-[300px]">
          {/* Results Header */}
          <div className="flex items-center justify-between px-4 py-2 border-b border-slate-200 bg-slate-50 rounded-t-lg">
            <div className="flex items-center space-x-4">
              <span className="text-sm font-medium text-slate-700">Results</span>
              {results.length > 1 && (
                <div className="flex space-x-1">
                  {results.map((_, idx) => (
                    <button
                      key={idx}
                      onClick={() => {
                        setActiveResultTab(idx)
                        setResultPage(1)
                      }}
                      className={`px-2 py-0.5 text-xs rounded transition-colors ${
                        activeResultTab === idx
                          ? 'bg-primary-100 text-primary-700'
                          : 'text-slate-500 hover:bg-slate-100'
                      }`}
                    >
                      Query {idx + 1}
                    </button>
                  ))}
                </div>
              )}
            </div>
            <div className="flex items-center space-x-3 text-xs text-slate-500">
              {elapsedMs !== null && (
                <span className="flex items-center space-x-1">
                  <Clock className="w-3 h-3" />
                  <span>{elapsedMs}ms</span>
                </span>
              )}
              {currentResult && currentResult.rows && (
                <>
                  <span className="flex items-center space-x-1">
                    <Rows3 className="w-3 h-3" />
                    <span>{totalResultRows} rows</span>
                  </span>
                  <button
                    onClick={exportCsv}
                    className="flex items-center space-x-1 px-2 py-1 hover:bg-slate-100 rounded transition-colors"
                    title="Export as CSV"
                  >
                    <Download className="w-3 h-3" />
                    <span>CSV</span>
                  </button>
                </>
              )}
              {currentResult && currentResult.rows_affected !== undefined && (
                <span className="flex items-center space-x-1">
                  <CheckCircle2 className="w-3 h-3 text-green-500" />
                  <span>{currentResult.rows_affected} rows affected</span>
                </span>
              )}
            </div>
          </div>

          {/* Results Content */}
          <div className="flex-1 overflow-auto">
            {/* Error State */}
            {error && (
              <div className="p-4">
                <div className="flex items-start space-x-3 p-4 bg-red-50 border border-red-200 rounded-lg">
                  <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium text-red-800">Query Error</p>
                    <p className="text-sm text-red-600 mt-1 font-mono">{error}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Statement-level error */}
            {currentResult?.error && (
              <div className="p-4">
                <div className="flex items-start space-x-3 p-4 bg-red-50 border border-red-200 rounded-lg">
                  <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium text-red-800">Statement Error</p>
                    <p className="text-sm text-red-600 mt-1 font-mono">{currentResult.error}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Success message for non-SELECT queries */}
            {currentResult && currentResult.rows_affected !== undefined && !currentResult.columns && (
              <div className="p-4">
                <div className="flex items-center space-x-3 p-4 bg-green-50 border border-green-200 rounded-lg">
                  <CheckCircle2 className="w-5 h-5 text-green-500" />
                  <p className="text-sm text-green-800">
                    Query executed successfully. {currentResult.rows_affected} row(s) affected.
                  </p>
                </div>
              </div>
            )}

            {/* Data Table */}
            {currentResult?.columns && currentResult.rows && (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-slate-50 sticky top-0">
                    <tr>
                      <th className="px-3 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider border-b border-r border-slate-200 bg-slate-100 w-12">
                        #
                      </th>
                      {currentResult.columns.map((col, idx) => (
                        <th
                          key={idx}
                          className="px-3 py-2 text-left text-xs font-medium text-slate-700 border-b border-r border-slate-200 bg-slate-50 whitespace-nowrap"
                        >
                          {col}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {paginatedRows.map((row, rowIdx) => {
                      const actualRowNum = (resultPage - 1) * resultsPerPage + rowIdx + 1
                      return (
                        <tr key={rowIdx} className="hover:bg-slate-50">
                          <td className="px-3 py-2 text-xs text-slate-400 border-r border-slate-100 bg-slate-50">
                            {actualRowNum}
                          </td>
                          {row.map((cell, cellIdx) => {
                            const cellId = `${rowIdx}-${cellIdx}`
                            const isNull = cell === null
                            return (
                              <td
                                key={cellIdx}
                                className="px-3 py-2 font-mono text-xs border-r border-slate-100 max-w-xs truncate group relative"
                                title={formatCell(cell)}
                              >
                                <span className={isNull ? 'text-slate-400 italic' : 'text-slate-900'}>
                                  {formatCell(cell)}
                                </span>
                                <button
                                  onClick={() => copyValue(cell, cellId)}
                                  className="absolute right-1 top-1/2 -translate-y-1/2 p-1 opacity-0 group-hover:opacity-100 hover:bg-slate-200 rounded transition-opacity"
                                >
                                  {copiedCell === cellId ? (
                                    <Check className="w-3 h-3 text-green-500" />
                                  ) : (
                                    <Copy className="w-3 h-3 text-slate-400" />
                                  )}
                                </button>
                              </td>
                            )
                          })}
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            )}

            {/* Empty State */}
            {!executing && !error && results.length === 0 && (
              <div className="flex flex-col items-center justify-center h-full py-12 text-slate-400">
                <Database className="w-12 h-12 mb-4" />
                <p className="text-sm">Run a query to see results</p>
                <p className="text-xs mt-1">Press Ctrl+Enter or click Run</p>
              </div>
            )}

            {/* Loading State */}
            {executing && (
              <div className="flex flex-col items-center justify-center h-full py-12">
                <Loader2 className="w-8 h-8 text-primary-500 animate-spin mb-4" />
                <p className="text-sm text-slate-500">Executing query...</p>
              </div>
            )}
          </div>

          {/* Pagination */}
          {totalResultPages > 1 && (
            <div className="flex items-center justify-between px-4 py-2 border-t border-slate-200 bg-slate-50 rounded-b-lg">
              <span className="text-xs text-slate-500">
                Showing {(resultPage - 1) * resultsPerPage + 1} - {Math.min(resultPage * resultsPerPage, totalResultRows)} of {totalResultRows}
              </span>
              <div className="flex items-center space-x-2">
                <button
                  onClick={() => setResultPage(p => Math.max(1, p - 1))}
                  disabled={resultPage === 1}
                  className="p-1 rounded hover:bg-slate-200 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <ChevronLeft className="w-4 h-4" />
                </button>
                <span className="text-xs text-slate-600">
                  Page {resultPage} of {totalResultPages}
                </span>
                <button
                  onClick={() => setResultPage(p => Math.min(totalResultPages, p + 1))}
                  disabled={resultPage === totalResultPages}
                  className="p-1 rounded hover:bg-slate-200 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <ChevronRight className="w-4 h-4" />
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
