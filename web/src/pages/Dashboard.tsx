import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { 
  Database, 
  Plus, 
  Trash2, 
  Eye, 
  EyeOff, 
  Copy, 
  Check, 
  LogOut, 
  User,
  Clock,
  AlertCircle,
  Loader2,
  RefreshCw
} from 'lucide-react'
import { api, ApiException, Instance } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

export default function Dashboard() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [instances, setInstances] = useState<Instance[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isCreating, setIsCreating] = useState(false)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [newDbName, setNewDbName] = useState('')
  const [error, setError] = useState('')
  const [visiblePasswords, setVisiblePasswords] = useState<Set<string>>(new Set())
  const [copiedFields, setCopiedFields] = useState<Set<string>>(new Set())
  const [deletingId, setDeletingId] = useState<string | null>(null)

  useEffect(() => {
    fetchInstances()
  }, [])

  async function fetchInstances() {
    try {
      setIsLoading(true)
      const data = await api.listInstances()
      setInstances(data)
    } catch (err) {
      if (err instanceof ApiException && err.status === 401) {
        logout()
        navigate('/signin')
      }
    } finally {
      setIsLoading(false)
    }
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setIsCreating(true)

    try {
      const instance = await api.createInstance(newDbName)
      setInstances([instance, ...instances])
      setShowCreateModal(false)
      setNewDbName('')
    } catch (err) {
      if (err instanceof ApiException) {
        setError(err.message)
      } else {
        setError('Failed to create database')
      }
    } finally {
      setIsCreating(false)
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Are you sure you want to delete this database? This action cannot be undone.')) {
      return
    }

    setDeletingId(id)
    try {
      await api.deleteInstance(id)
      setInstances(instances.filter((i) => i.id !== id))
    } catch (err) {
      if (err instanceof ApiException) {
        alert(err.message)
      }
    } finally {
      setDeletingId(null)
    }
  }

  async function handleRevealPassword(id: string) {
    if (visiblePasswords.has(id)) {
      setVisiblePasswords((prev) => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
      return
    }

    try {
      const password = await api.revealPassword(id)
      setInstances((prev) =>
        prev.map((i) => (i.id === id ? { ...i, password } : i))
      )
      setVisiblePasswords((prev) => new Set(prev).add(id))
      // Auto-hide after 30 seconds
      setTimeout(() => {
        setVisiblePasswords((prev) => {
          const next = new Set(prev)
          next.delete(id)
          return next
        })
      }, 30000)
    } catch (err) {
      if (err instanceof ApiException) {
        alert(err.message)
      }
    }
  }

  function copyToClipboard(text: string, fieldId: string) {
    navigator.clipboard.writeText(text)
    setCopiedFields((prev) => new Set(prev).add(fieldId))
    setTimeout(() => {
      setCopiedFields((prev) => {
        const next = new Set(prev)
        next.delete(fieldId)
        return next
      })
    }, 2000)
  }

  function getStatusColor(status: string) {
    switch (status) {
      case 'active':
        return 'bg-green-100 text-green-700'
      case 'suspended':
        return 'bg-red-100 text-red-700'
      case 'pending':
        return 'bg-yellow-100 text-yellow-700'
      default:
        return 'bg-slate-100 text-slate-700'
    }
  }

  function formatDate(date: string) {
    return new Date(date).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <header className="bg-white border-b border-slate-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <Link to="/" className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-gradient-to-br from-primary-500 to-accent-500 rounded-lg flex items-center justify-center">
                <span className="text-white font-bold text-sm">g2</span>
              </div>
              <span className="font-bold text-slate-900">go2postgres</span>
            </Link>
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2 text-slate-600">
                <User className="w-4 h-4" />
                <span className="text-sm">{user?.email}</span>
                {user?.role === 'admin' && (
                  <Link
                    to="/admin"
                    className="ml-2 px-2 py-1 bg-primary-100 text-primary-700 text-xs font-medium rounded"
                  >
                    Admin
                  </Link>
                )}
              </div>
              <button
                onClick={() => {
                  logout()
                  navigate('/')
                }}
                className="flex items-center space-x-1 text-slate-600 hover:text-slate-900"
              >
                <LogOut className="w-4 h-4" />
                <span className="text-sm">Logout</span>
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Page Header */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-2xl font-bold text-slate-900">My Databases</h1>
            <p className="text-slate-600">Manage your PostgreSQL instances</p>
          </div>
          <div className="flex items-center space-x-3">
            <button
              onClick={fetchInstances}
              disabled={isLoading}
              className="p-2 text-slate-600 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition-colors"
            >
              <RefreshCw className={`w-5 h-5 ${isLoading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={() => setShowCreateModal(true)}
              className="flex items-center space-x-2 px-4 py-2 bg-gradient-to-r from-primary-500 to-accent-500 text-white rounded-lg font-medium hover:opacity-90 transition-opacity"
            >
              <Plus className="w-5 h-5" />
              <span>New Database</span>
            </button>
          </div>
        </div>

        {/* Instances Grid */}
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <Loader2 className="w-8 h-8 animate-spin text-primary-500" />
          </div>
        ) : instances.length === 0 ? (
          <div className="bg-white rounded-xl border border-slate-200 p-12 text-center">
            <Database className="w-16 h-16 text-slate-300 mx-auto mb-4" />
            <h3 className="text-lg font-semibold text-slate-900 mb-2">No databases yet</h3>
            <p className="text-slate-600 mb-6">Create your first PostgreSQL database to get started.</p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="inline-flex items-center space-x-2 px-4 py-2 bg-gradient-to-r from-primary-500 to-accent-500 text-white rounded-lg font-medium hover:opacity-90 transition-opacity"
            >
              <Plus className="w-5 h-5" />
              <span>Create Database</span>
            </button>
          </div>
        ) : (
          <div className="grid gap-6">
            {instances.map((instance) => (
              <div
                key={instance.id}
                className="bg-white rounded-xl border border-slate-200 p-6 hover:border-primary-300 transition-colors"
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center space-x-3">
                    <div className="w-10 h-10 bg-primary-100 rounded-lg flex items-center justify-center">
                      <Database className="w-5 h-5 text-primary-600" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-slate-900">{instance.db_name}</h3>
                      <div className="flex items-center space-x-2 text-sm text-slate-500">
                        <Clock className="w-3 h-3" />
                        <span>Created {formatDate(instance.created_at)}</span>
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(instance.status)}`}>
                      {instance.status}
                    </span>
                    <button
                      onClick={() => handleDelete(instance.id)}
                      disabled={deletingId === instance.id}
                      className="p-2 text-slate-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors disabled:opacity-50"
                    >
                      {deletingId === instance.id ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <Trash2 className="w-4 h-4" />
                      )}
                    </button>
                  </div>
                </div>

                {/* Connection Details */}
                <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4 p-4 bg-slate-50 rounded-lg">
                  <div>
                    <label className="block text-xs font-medium text-slate-500 mb-1">Host</label>
                    <div className="flex items-center space-x-2">
                      <code className="text-sm text-slate-900">{instance.host}</code>
                      <button
                        onClick={() => copyToClipboard(instance.host, `${instance.id}-host`)}
                        className="p-1 text-slate-400 hover:text-slate-600"
                      >
                        {copiedFields.has(`${instance.id}-host`) ? (
                          <Check className="w-3 h-3 text-green-500" />
                        ) : (
                          <Copy className="w-3 h-3" />
                        )}
                      </button>
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-slate-500 mb-1">Port</label>
                    <div className="flex items-center space-x-2">
                      <code className="text-sm text-slate-900">{instance.port}</code>
                      <button
                        onClick={() => copyToClipboard(instance.port.toString(), `${instance.id}-port`)}
                        className="p-1 text-slate-400 hover:text-slate-600"
                      >
                        {copiedFields.has(`${instance.id}-port`) ? (
                          <Check className="w-3 h-3 text-green-500" />
                        ) : (
                          <Copy className="w-3 h-3" />
                        )}
                      </button>
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-slate-500 mb-1">Username</label>
                    <div className="flex items-center space-x-2">
                      <code className="text-sm text-slate-900">{instance.username}</code>
                      <button
                        onClick={() => copyToClipboard(instance.username, `${instance.id}-user`)}
                        className="p-1 text-slate-400 hover:text-slate-600"
                      >
                        {copiedFields.has(`${instance.id}-user`) ? (
                          <Check className="w-3 h-3 text-green-500" />
                        ) : (
                          <Copy className="w-3 h-3" />
                        )}
                      </button>
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-slate-500 mb-1">Password</label>
                    <div className="flex items-center space-x-2">
                      {visiblePasswords.has(instance.id) && instance.password ? (
                        <code className="text-sm text-slate-900">{instance.password}</code>
                      ) : (
                        <span className="text-sm text-slate-400">••••••••</span>
                      )}
                      <button
                        onClick={() => handleRevealPassword(instance.id)}
                        className="p-1 text-slate-400 hover:text-slate-600"
                      >
                        {visiblePasswords.has(instance.id) ? (
                          <EyeOff className="w-3 h-3" />
                        ) : (
                          <Eye className="w-3 h-3" />
                        )}
                      </button>
                      {visiblePasswords.has(instance.id) && instance.password && (
                        <button
                          onClick={() => copyToClipboard(instance.password!, `${instance.id}-pass`)}
                          className="p-1 text-slate-400 hover:text-slate-600"
                        >
                          {copiedFields.has(`${instance.id}-pass`) ? (
                            <Check className="w-3 h-3 text-green-500" />
                          ) : (
                            <Copy className="w-3 h-3" />
                          )}
                        </button>
                      )}
                    </div>
                  </div>
                </div>

                {/* Connection String */}
                <div className="mt-4">
                  <label className="block text-xs font-medium text-slate-500 mb-1">Connection String</label>
                  <div className="flex items-center space-x-2 p-3 bg-slate-900 rounded-lg overflow-x-auto">
                    <code className="text-sm text-green-400 whitespace-nowrap">
                      postgresql://{instance.username}:{visiblePasswords.has(instance.id) && instance.password ? instance.password : '******'}@{instance.host}:{instance.port}/{instance.db_name}
                    </code>
                    {visiblePasswords.has(instance.id) && instance.password && (
                      <button
                        onClick={() =>
                          copyToClipboard(
                            `postgresql://${instance.username}:${instance.password}@${instance.host}:${instance.port}/${instance.db_name}`,
                            `${instance.id}-conn`
                          )
                        }
                        className="p-1 text-slate-400 hover:text-white flex-shrink-0"
                      >
                        {copiedFields.has(`${instance.id}-conn`) ? (
                          <Check className="w-4 h-4 text-green-500" />
                        ) : (
                          <Copy className="w-4 h-4" />
                        )}
                      </button>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl max-w-md w-full p-6">
            <h2 className="text-xl font-bold text-slate-900 mb-4">Create New Database</h2>
            {error && (
              <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm flex items-center space-x-2">
                <AlertCircle className="w-4 h-4 flex-shrink-0" />
                <span>{error}</span>
              </div>
            )}
            <form onSubmit={handleCreate}>
              <div className="mb-4">
                <label htmlFor="dbName" className="block text-sm font-medium text-slate-700 mb-2">
                  Database Name
                </label>
                <input
                  id="dbName"
                  type="text"
                  value={newDbName}
                  onChange={(e) => setNewDbName(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, ''))}
                  className="w-full px-4 py-3 border border-slate-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none"
                  placeholder="my_database"
                  required
                  autoFocus
                  pattern="[a-z][a-z0-9_]*"
                  title="Must start with a letter, only lowercase letters, numbers, and underscores"
                />
                <p className="mt-1 text-xs text-slate-500">
                  Lowercase letters, numbers, and underscores only. Must start with a letter.
                </p>
              </div>
              <div className="flex space-x-3">
                <button
                  type="button"
                  onClick={() => {
                    setShowCreateModal(false)
                    setNewDbName('')
                    setError('')
                  }}
                  className="flex-1 py-2 border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-50 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isCreating || !newDbName}
                  className="flex-1 py-2 bg-gradient-to-r from-primary-500 to-accent-500 text-white rounded-lg font-medium hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center"
                >
                  {isCreating ? (
                    <Loader2 className="w-5 h-5 animate-spin" />
                  ) : (
                    'Create'
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
