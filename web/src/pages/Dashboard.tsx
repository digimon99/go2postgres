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
  RefreshCw,
  Key,
  ChevronDown,
  ChevronUp,
  Table2,
  SquareCode
} from 'lucide-react'
import { api, ApiException, Instance, APIKey } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

export default function Dashboard() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [instances, setInstances] = useState<Instance[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isCreating, setIsCreating] = useState(false)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [newDbName, setNewDbName] = useState('')
  const [newProjectId, setNewProjectId] = useState('')
  const [error, setError] = useState('')
  const [visiblePasswords, setVisiblePasswords] = useState<Set<string>>(new Set())
  const [copiedFields, setCopiedFields] = useState<Set<string>>(new Set())
  const [deletingId, setDeletingId] = useState<string | null>(null)


  // API Keys state
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set())
  const [apiKeys, setApiKeys] = useState<Record<string, APIKey[]>>({})
  const [loadingKeys, setLoadingKeys] = useState<Set<string>>(new Set())
  const [showKeyModal, setShowKeyModal] = useState<string | null>(null)
  const [newKeyName, setNewKeyName] = useState('')
  const [newKeyType, setNewKeyType] = useState<'fullaccess' | 'readonly'>('fullaccess')
  const [newKeyIps, setNewKeyIps] = useState('')
  const [createdKey, setCreatedKey] = useState<string | null>(null)
  const [keyError, setKeyError] = useState('')
  const [creatingKey, setCreatingKey] = useState(false)

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
      const instance = await api.createInstance(newDbName, newProjectId)
      setInstances([instance, ...instances])
      setShowCreateModal(false)
      setNewDbName('')
      setNewProjectId('')
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
      setInstances(instances.filter((i) => i.instance_id !== id))
    } catch (err) {
      if (err instanceof ApiException) {
        alert(err.message)
      }
    } finally {
      setDeletingId(null)
    }
  }

  function handleRevealPassword(id: string) {
    if (visiblePasswords.has(id)) {
      setVisiblePasswords((prev) => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
      return
    }

    // Decode password client-side from base64 (avoids AJAX call that triggers Cloudflare WAF)
    const instance = instances.find((i) => i.instance_id === id)
    if (instance?.password_encoded) {
      try {
        const password = atob(instance.password_encoded)
        setInstances((prev) =>
          prev.map((i) => (i.instance_id === id ? { ...i, password } : i))
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
      } catch {
        alert('Failed to decode password')
      }
    } else {
      alert('Password not available')
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

  // ---- API Key Functions ----
  async function toggleExpandKeys(instanceId: string) {
    if (expandedKeys.has(instanceId)) {
      setExpandedKeys((prev) => {
        const n = new Set(prev)
        n.delete(instanceId)
        return n
      })
      return
    }
    setExpandedKeys((prev) => new Set(prev).add(instanceId))
    if (!apiKeys[instanceId]) {
      setLoadingKeys((prev) => new Set(prev).add(instanceId))
      try {
        const keys = await api.listAPIKeys(instanceId)
        setApiKeys((prev) => ({ ...prev, [instanceId]: keys }))
      } catch (err) {
        console.error('Error loading API keys', err)
      } finally {
        setLoadingKeys((prev) => {
          const n = new Set(prev)
          n.delete(instanceId)
          return n
        })
      }
    }
  }

  async function handleCreateKey(e: React.FormEvent) {
    e.preventDefault()
    if (!showKeyModal) return
    setKeyError('')
    setCreatingKey(true)
    try {
      // Parse IPs into JSON array
      let ipListJson = '[]'
      if (newKeyIps.trim()) {
        const ips = newKeyIps.split(',').map((s) => s.trim()).filter(Boolean)
        ipListJson = JSON.stringify(ips)
      }
      const res = await api.createAPIKey(showKeyModal, newKeyName, newKeyType, ipListJson)
      setCreatedKey(res.key)
      // Refresh the key list
      const keys = await api.listAPIKeys(showKeyModal)
      setApiKeys((prev) => ({ ...prev, [showKeyModal]: keys }))
    } catch (err) {
      if (err instanceof ApiException) {
        setKeyError(err.message)
      } else {
        setKeyError('Failed to create API key')
      }
    } finally {
      setCreatingKey(false)
    }
  }

  async function handleRevokeKey(keyId: string, instanceId: string) {
    if (!confirm('Revoke this API key? Any applications using it will stop working.')) return
    try {
      await api.revokeAPIKey(keyId)
      const keys = await api.listAPIKeys(instanceId)
      setApiKeys((prev) => ({ ...prev, [instanceId]: keys }))
    } catch (err) {
      alert('Failed to revoke key')
    }
  }

  function closeKeyModal() {
    setShowKeyModal(null)
    setNewKeyName('')
    setNewKeyType('fullaccess')
    setNewKeyIps('')
    setCreatedKey(null)
    setKeyError('')
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
                key={instance.instance_id}
                className="bg-white rounded-xl border border-slate-200 p-6 hover:border-primary-300 transition-colors"
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center space-x-3">
                    <div className="w-10 h-10 bg-primary-100 rounded-lg flex items-center justify-center">
                      <Database className="w-5 h-5 text-primary-600" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-slate-900">{instance.database_name}</h3>
                      <div className="flex items-center space-x-2 text-sm text-slate-500">
                        <span className="text-primary-600 font-medium">{instance.project_id}</span>
                        <span>•</span>
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
                      onClick={() => navigate(`/instances/${instance.instance_id}/sql`)}
                      className="p-2 text-slate-400 hover:text-primary-500 hover:bg-primary-50 rounded-lg transition-colors"
                      title="SQL Editor"
                    >
                      <SquareCode className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => navigate(`/instances/${instance.instance_id}/tables`)}
                      className="p-2 text-slate-400 hover:text-primary-500 hover:bg-primary-50 rounded-lg transition-colors"
                      title="Table Editor"
                    >
                      <Table2 className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(instance.instance_id)}
                      disabled={deletingId === instance.instance_id}
                      className="p-2 text-slate-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors disabled:opacity-50"
                    >
                      {deletingId === instance.instance_id ? (
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
                        onClick={() => copyToClipboard(instance.host, `${instance.instance_id}-host`)}
                        className="p-1 text-slate-400 hover:text-slate-600"
                      >
                        {copiedFields.has(`${instance.instance_id}-host`) ? (
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
                        onClick={() => copyToClipboard(instance.port.toString(), `${instance.instance_id}-port`)}
                        className="p-1 text-slate-400 hover:text-slate-600"
                      >
                        {copiedFields.has(`${instance.instance_id}-port`) ? (
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
                        onClick={() => copyToClipboard(instance.username, `${instance.instance_id}-user`)}
                        className="p-1 text-slate-400 hover:text-slate-600"
                      >
                        {copiedFields.has(`${instance.instance_id}-user`) ? (
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
                      {visiblePasswords.has(instance.instance_id) && instance.password ? (
                        <code className="text-sm text-slate-900">{instance.password}</code>
                      ) : (
                        <span className="text-sm text-slate-400">••••••••</span>
                      )}
                      <button
                        onClick={() => handleRevealPassword(instance.instance_id)}
                        className="p-1 text-slate-400 hover:text-slate-600"
                      >
                        {visiblePasswords.has(instance.instance_id) ? (
                          <EyeOff className="w-3 h-3" />
                        ) : (
                          <Eye className="w-3 h-3" />
                        )}
                      </button>
                      {visiblePasswords.has(instance.instance_id) && instance.password && (
                        <button
                          onClick={() => copyToClipboard(instance.password!, `${instance.instance_id}-pass`)}
                          className="p-1 text-slate-400 hover:text-slate-600"
                        >
                          {copiedFields.has(`${instance.instance_id}-pass`) ? (
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
                      postgresql://{instance.username}:{visiblePasswords.has(instance.instance_id) && instance.password ? instance.password : '******'}@{instance.host}:{instance.port}/{instance.database_name}
                    </code>
                    {visiblePasswords.has(instance.instance_id) && instance.password && (
                      <button
                        onClick={() =>
                          copyToClipboard(
                            `postgresql://${instance.username}:${instance.password}@${instance.host}:${instance.port}/${instance.database_name}`,
                            `${instance.instance_id}-conn`
                          )
                        }
                        className="p-1 text-slate-400 hover:text-white flex-shrink-0"
                      >
                        {copiedFields.has(`${instance.instance_id}-conn`) ? (
                          <Check className="w-4 h-4 text-green-500" />
                        ) : (
                          <Copy className="w-4 h-4" />
                        )}
                      </button>
                    )}
                  </div>
                </div>

                {/* API Keys Section */}
                <div className="mt-4 border-t border-slate-100 pt-4">
                  <button
                    onClick={() => toggleExpandKeys(instance.instance_id)}
                    className="flex items-center space-x-2 text-sm font-medium text-slate-700 hover:text-primary-600"
                  >
                    <Key className="w-4 h-4" />
                    <span>API Keys</span>
                    {expandedKeys.has(instance.instance_id) ? (
                      <ChevronUp className="w-4 h-4" />
                    ) : (
                      <ChevronDown className="w-4 h-4" />
                    )}
                  </button>
                  {expandedKeys.has(instance.instance_id) && (
                    <div className="mt-3 space-y-2">
                      {loadingKeys.has(instance.instance_id) ? (
                        <div className="flex items-center space-x-2 text-slate-500 text-sm">
                          <Loader2 className="w-4 h-4 animate-spin" />
                          <span>Loading keys...</span>
                        </div>
                      ) : (
                        <>
                          {(apiKeys[instance.instance_id] || []).length === 0 && (
                            <p className="text-sm text-slate-500">No API keys yet</p>
                          )}
                          {(apiKeys[instance.instance_id] || []).map((k) => (
                            <div key={k.key_id} className="flex items-center justify-between p-2 bg-slate-50 rounded text-sm">
                              <div>
                                <span className="font-medium text-slate-800">{k.name}</span>
                                <span className="text-slate-400 ml-2">({k.key_preview})</span>
                                <span className={`ml-2 px-1.5 py-0.5 rounded text-xs ${k.key_type === 'readonly' ? 'bg-blue-100 text-blue-700' : 'bg-purple-100 text-purple-700'}`}>
                                  {k.key_type}
                                </span>
                              </div>
                              <button
                                onClick={() => handleRevokeKey(k.key_id, instance.instance_id)}
                                className="text-red-500 hover:text-red-700"
                              >
                                <Trash2 className="w-4 h-4" />
                              </button>
                            </div>
                          ))}
                          <button
                            onClick={() => setShowKeyModal(instance.instance_id)}
                            className="w-full flex items-center justify-center space-x-1 p-2 border border-dashed border-slate-300 rounded text-sm text-slate-600 hover:border-primary-400 hover:text-primary-600"
                          >
                            <Plus className="w-4 h-4" />
                            <span>Create API Key</span>
                          </button>
                        </>
                      )}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </main>

      {/* Create Instance Modal */}
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
                <label htmlFor="projectId" className="block text-sm font-medium text-slate-700 mb-2">
                  Project ID
                </label>
                <input
                  id="projectId"
                  type="text"
                  value={newProjectId}
                  onChange={(e) => setNewProjectId(e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, ''))}
                  className="w-full px-4 py-3 border border-slate-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none"
                  placeholder="my-project"
                  required
                  autoFocus
                />
                <p className="mt-1 text-xs text-slate-500">
                  A unique identifier for your project (e.g., my-app, i360)
                </p>
              </div>
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
                    setNewProjectId('')
                    setError('')
                  }}
                  className="flex-1 py-2 border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-50 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isCreating || !newDbName || !newProjectId}
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

      {/* Create API Key Modal */}
      {showKeyModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl max-w-md w-full p-6">
            {createdKey ? (
              // Show the one-time key
              <div>
                <h2 className="text-xl font-bold text-slate-900 mb-4">API Key Created</h2>
                <div className="p-4 bg-yellow-50 border border-yellow-200 rounded-lg mb-4">
                  <p className="text-yellow-800 text-sm mb-2 font-medium">
                    ⚠️ Copy this key now. You won't be able to see it again!
                  </p>
                  <div className="flex items-center space-x-2 bg-white border border-yellow-300 p-2 rounded">
                    <code className="text-sm text-slate-900 flex-1 break-all">{createdKey}</code>
                    <button
                      onClick={() => copyToClipboard(createdKey, 'new-api-key')}
                      className="p-1 text-slate-400 hover:text-slate-600"
                    >
                      {copiedFields.has('new-api-key') ? (
                        <Check className="w-4 h-4 text-green-500" />
                      ) : (
                        <Copy className="w-4 h-4" />
                      )}
                    </button>
                  </div>
                </div>
                <button
                  onClick={closeKeyModal}
                  className="w-full py-2 bg-slate-900 text-white rounded-lg hover:bg-slate-800"
                >
                  Done
                </button>
              </div>
            ) : (
              <form onSubmit={handleCreateKey}>
                <h2 className="text-xl font-bold text-slate-900 mb-4">Create API Key</h2>
                {keyError && (
                  <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
                    {keyError}
                  </div>
                )}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-slate-700 mb-2">Key Name</label>
                  <input
                    type="text"
                    value={newKeyName}
                    onChange={(e) => setNewKeyName(e.target.value)}
                    className="w-full px-4 py-3 border border-slate-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none"
                    placeholder="My API Key"
                    required
                    autoFocus
                  />
                </div>
                <div className="mb-4">
                  <label className="block text-sm font-medium text-slate-700 mb-2">Access Type</label>
                  <div className="flex space-x-4">
                    <label className="flex items-center space-x-2 cursor-pointer">
                      <input
                        type="radio"
                        name="keyType"
                        value="fullaccess"
                        checked={newKeyType === 'fullaccess'}
                        onChange={() => setNewKeyType('fullaccess')}
                        className="text-primary-600"
                      />
                      <span className="text-sm text-slate-700">Full Access</span>
                    </label>
                    <label className="flex items-center space-x-2 cursor-pointer">
                      <input
                        type="radio"
                        name="keyType"
                        value="readonly"
                        checked={newKeyType === 'readonly'}
                        onChange={() => setNewKeyType('readonly')}
                        className="text-primary-600"
                      />
                      <span className="text-sm text-slate-700">Read Only</span>
                    </label>
                  </div>
                </div>
                <div className="mb-4">
                  <label className="block text-sm font-medium text-slate-700 mb-2">IP Allowlist (optional)</label>
                  <input
                    type="text"
                    value={newKeyIps}
                    onChange={(e) => setNewKeyIps(e.target.value)}
                    className="w-full px-4 py-3 border border-slate-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none"
                    placeholder="1.2.3.4, 10.0.0.0/8"
                  />
                  <p className="mt-1 text-xs text-slate-500">
                    Comma-separated IPs or CIDR ranges. Leave empty to allow all.
                  </p>
                </div>
                <div className="flex space-x-3">
                  <button
                    type="button"
                    onClick={closeKeyModal}
                    className="flex-1 py-2 border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-50"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={creatingKey || !newKeyName}
                    className="flex-1 py-2 bg-gradient-to-r from-primary-500 to-accent-500 text-white rounded-lg font-medium hover:opacity-90 disabled:opacity-50"
                  >
                    {creatingKey ? <Loader2 className="w-5 h-5 animate-spin mx-auto" /> : 'Create Key'}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
