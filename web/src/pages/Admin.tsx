import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { 
  Database, 
  Users, 
  Server,
  Activity,
  LogOut, 
  User,
  Shield,
  Search,
  ChevronDown,
  ChevronUp,
  RefreshCw,
  Loader2,
  UserCheck,
  UserX,
  Clock,
  Mail
} from 'lucide-react'
import { api, ApiException, AdminStats, AdminUser, AdminInstance } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

type Tab = 'overview' | 'users' | 'instances'

export default function Admin() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<Tab>('overview')
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [users, setUsers] = useState<AdminUser[]>([])
  const [instances, setInstances] = useState<AdminInstance[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [sortField, setSortField] = useState<string>('created_at')
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')

  useEffect(() => {
    if (user?.role !== 'admin') {
      navigate('/dashboard')
      return
    }
    fetchData()
  }, [user])

  async function fetchData() {
    setIsLoading(true)
    try {
      const [statsData, usersData, instancesData] = await Promise.all([
        api.adminStats(),
        api.adminListUsers(),
        api.adminListInstances(),
      ])
      setStats(statsData)
      setUsers(usersData)
      setInstances(instancesData)
    } catch (err) {
      if (err instanceof ApiException && err.status === 401) {
        logout()
        navigate('/signin')
      }
    } finally {
      setIsLoading(false)
    }
  }

  async function handleSuspendUser(userId: string, suspend: boolean) {
    try {
      await api.adminUpdateUser(userId, { status: suspend ? 'suspended' : 'active' })
      setUsers((prev) =>
        prev.map((u) =>
          u.id === userId ? { ...u, status: suspend ? 'suspended' : 'active' } : u
        )
      )
    } catch (err) {
      if (err instanceof ApiException) {
        alert(err.message)
      }
    }
  }

  async function handleSuspendInstance(instanceId: string, suspend: boolean) {
    try {
      await api.adminUpdateInstance(instanceId, { status: suspend ? 'suspended' : 'active' })
      setInstances((prev) =>
        prev.map((i) =>
          i.id === instanceId ? { ...i, status: suspend ? 'suspended' : 'active' } : i
        )
      )
    } catch (err) {
      if (err instanceof ApiException) {
        alert(err.message)
      }
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

  const filteredUsers = users
    .filter((u) => u.email.toLowerCase().includes(searchQuery.toLowerCase()))
    .sort((a, b) => {
      const aVal = a[sortField as keyof AdminUser] as string
      const bVal = b[sortField as keyof AdminUser] as string
      return sortDir === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal)
    })

  const filteredInstances = instances
    .filter(
      (i) =>
        i.db_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        i.user_email.toLowerCase().includes(searchQuery.toLowerCase())
    )
    .sort((a, b) => {
      const aVal = a[sortField as keyof AdminInstance] as string
      const bVal = b[sortField as keyof AdminInstance] as string
      return sortDir === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal)
    })

  function toggleSort(field: string) {
    if (sortField === field) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDir('desc')
    }
  }

  const SortIcon = ({ field }: { field: string }) => {
    if (sortField !== field) return null
    return sortDir === 'asc' ? (
      <ChevronUp className="w-4 h-4" />
    ) : (
      <ChevronDown className="w-4 h-4" />
    )
  }

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <header className="bg-white border-b border-slate-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-4">
              <Link to="/" className="flex items-center space-x-2">
                <div className="w-8 h-8 bg-gradient-to-br from-primary-500 to-accent-500 rounded-lg flex items-center justify-center">
                  <span className="text-white font-bold text-sm">g2</span>
                </div>
                <span className="font-bold text-slate-900">go2postgres</span>
              </Link>
              <span className="px-2 py-1 bg-primary-100 text-primary-700 text-xs font-medium rounded">
                Admin
              </span>
            </div>
            <div className="flex items-center space-x-4">
              <Link
                to="/dashboard"
                className="text-sm text-slate-600 hover:text-slate-900"
              >
                My Databases
              </Link>
              <div className="flex items-center space-x-2 text-slate-600">
                <Shield className="w-4 h-4" />
                <span className="text-sm">{user?.email}</span>
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
            <h1 className="text-2xl font-bold text-slate-900">Admin Dashboard</h1>
            <p className="text-slate-600">Manage users and database instances</p>
          </div>
          <button
            onClick={fetchData}
            disabled={isLoading}
            className="flex items-center space-x-2 px-4 py-2 border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors"
          >
            <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>
        </div>

        {/* Tabs */}
        <div className="flex space-x-1 mb-8 bg-slate-100 p-1 rounded-lg w-fit">
          <button
            onClick={() => setActiveTab('overview')}
            className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
              activeTab === 'overview'
                ? 'bg-white text-slate-900 shadow-sm'
                : 'text-slate-600 hover:text-slate-900'
            }`}
          >
            Overview
          </button>
          <button
            onClick={() => setActiveTab('users')}
            className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
              activeTab === 'users'
                ? 'bg-white text-slate-900 shadow-sm'
                : 'text-slate-600 hover:text-slate-900'
            }`}
          >
            Users
          </button>
          <button
            onClick={() => setActiveTab('instances')}
            className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
              activeTab === 'instances'
                ? 'bg-white text-slate-900 shadow-sm'
                : 'text-slate-600 hover:text-slate-900'
            }`}
          >
            Instances
          </button>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <Loader2 className="w-8 h-8 animate-spin text-primary-500" />
          </div>
        ) : (
          <>
            {/* Overview Tab */}
            {activeTab === 'overview' && stats && (
              <div className="space-y-8">
                {/* Stats Grid */}
                <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
                  <div className="bg-white rounded-xl border border-slate-200 p-6">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium text-slate-500">Total Users</p>
                        <p className="text-3xl font-bold text-slate-900 mt-1">{stats.total_users}</p>
                      </div>
                      <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center">
                        <Users className="w-6 h-6 text-blue-600" />
                      </div>
                    </div>
                  </div>
                  <div className="bg-white rounded-xl border border-slate-200 p-6">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium text-slate-500">Active Users</p>
                        <p className="text-3xl font-bold text-slate-900 mt-1">{stats.active_users}</p>
                      </div>
                      <div className="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center">
                        <UserCheck className="w-6 h-6 text-green-600" />
                      </div>
                    </div>
                  </div>
                  <div className="bg-white rounded-xl border border-slate-200 p-6">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium text-slate-500">Total Instances</p>
                        <p className="text-3xl font-bold text-slate-900 mt-1">{stats.total_instances}</p>
                      </div>
                      <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center">
                        <Database className="w-6 h-6 text-primary-600" />
                      </div>
                    </div>
                  </div>
                  <div className="bg-white rounded-xl border border-slate-200 p-6">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium text-slate-500">Active Instances</p>
                        <p className="text-3xl font-bold text-slate-900 mt-1">{stats.active_instances}</p>
                      </div>
                      <div className="w-12 h-12 bg-accent-100 rounded-lg flex items-center justify-center">
                        <Activity className="w-6 h-6 text-accent-600" />
                      </div>
                    </div>
                  </div>
                </div>

                {/* Recent Activity */}
                <div className="bg-white rounded-xl border border-slate-200 p-6">
                  <h3 className="text-lg font-semibold text-slate-900 mb-4">Recent Users</h3>
                  <div className="space-y-3">
                    {users.slice(0, 5).map((u) => (
                      <div key={u.id} className="flex items-center justify-between py-2 border-b border-slate-100 last:border-0">
                        <div className="flex items-center space-x-3">
                          <div className="w-8 h-8 bg-slate-100 rounded-full flex items-center justify-center">
                            <User className="w-4 h-4 text-slate-600" />
                          </div>
                          <div>
                            <p className="text-sm font-medium text-slate-900">{u.email}</p>
                            <p className="text-xs text-slate-500">Joined {formatDate(u.created_at)}</p>
                          </div>
                        </div>
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(u.status)}`}>
                          {u.status}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}

            {/* Users Tab */}
            {activeTab === 'users' && (
              <div className="space-y-4">
                {/* Search */}
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Search users by email..."
                    className="w-full pl-11 pr-4 py-3 border border-slate-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none"
                  />
                </div>

                {/* Users Table */}
                <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead className="bg-slate-50 border-b border-slate-200">
                        <tr>
                          <th
                            className="px-6 py-3 text-left text-xs font-semibold text-slate-600 uppercase cursor-pointer hover:bg-slate-100"
                            onClick={() => toggleSort('email')}
                          >
                            <div className="flex items-center space-x-1">
                              <span>Email</span>
                              <SortIcon field="email" />
                            </div>
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-semibold text-slate-600 uppercase">
                            Role
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-semibold text-slate-600 uppercase">
                            Status
                          </th>
                          <th
                            className="px-6 py-3 text-left text-xs font-semibold text-slate-600 uppercase cursor-pointer hover:bg-slate-100"
                            onClick={() => toggleSort('created_at')}
                          >
                            <div className="flex items-center space-x-1">
                              <span>Created</span>
                              <SortIcon field="created_at" />
                            </div>
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-semibold text-slate-600 uppercase">
                            Instances
                          </th>
                          <th className="px-6 py-3 text-right text-xs font-semibold text-slate-600 uppercase">
                            Actions
                          </th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-200">
                        {filteredUsers.map((u) => (
                          <tr key={u.id} className="hover:bg-slate-50">
                            <td className="px-6 py-4">
                              <div className="flex items-center space-x-3">
                                <div className="w-8 h-8 bg-slate-100 rounded-full flex items-center justify-center">
                                  <Mail className="w-4 h-4 text-slate-600" />
                                </div>
                                <span className="text-sm font-medium text-slate-900">{u.email}</span>
                              </div>
                            </td>
                            <td className="px-6 py-4">
                              <span className={`px-2 py-1 text-xs font-medium rounded ${
                                u.role === 'admin' ? 'bg-primary-100 text-primary-700' : 'bg-slate-100 text-slate-700'
                              }`}>
                                {u.role}
                              </span>
                            </td>
                            <td className="px-6 py-4">
                              <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(u.status)}`}>
                                {u.status}
                              </span>
                            </td>
                            <td className="px-6 py-4">
                              <div className="flex items-center space-x-1 text-sm text-slate-600">
                                <Clock className="w-3 h-3" />
                                <span>{formatDate(u.created_at)}</span>
                              </div>
                            </td>
                            <td className="px-6 py-4">
                              <span className="text-sm text-slate-900">{u.instance_count}</span>
                            </td>
                            <td className="px-6 py-4 text-right">
                              {u.role !== 'admin' && (
                                <button
                                  onClick={() => handleSuspendUser(u.id, u.status !== 'suspended')}
                                  className={`inline-flex items-center space-x-1 px-3 py-1 rounded text-sm font-medium ${
                                    u.status === 'suspended'
                                      ? 'bg-green-100 text-green-700 hover:bg-green-200'
                                      : 'bg-red-100 text-red-700 hover:bg-red-200'
                                  }`}
                                >
                                  {u.status === 'suspended' ? (
                                    <>
                                      <UserCheck className="w-3 h-3" />
                                      <span>Activate</span>
                                    </>
                                  ) : (
                                    <>
                                      <UserX className="w-3 h-3" />
                                      <span>Suspend</span>
                                    </>
                                  )}
                                </button>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  {filteredUsers.length === 0 && (
                    <div className="p-8 text-center text-slate-500">
                      No users found matching "{searchQuery}"
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Instances Tab */}
            {activeTab === 'instances' && (
              <div className="space-y-4">
                {/* Search */}
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Search instances by name or owner email..."
                    className="w-full pl-11 pr-4 py-3 border border-slate-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none"
                  />
                </div>

                {/* Instances Table */}
                <div className="bg-white rounded-xl border border-slate-200 overflow-hidden">
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead className="bg-slate-50 border-b border-slate-200">
                        <tr>
                          <th
                            className="px-6 py-3 text-left text-xs font-semibold text-slate-600 uppercase cursor-pointer hover:bg-slate-100"
                            onClick={() => toggleSort('db_name')}
                          >
                            <div className="flex items-center space-x-1">
                              <span>Database</span>
                              <SortIcon field="db_name" />
                            </div>
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-semibold text-slate-600 uppercase">
                            Owner
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-semibold text-slate-600 uppercase">
                            Status
                          </th>
                          <th
                            className="px-6 py-3 text-left text-xs font-semibold text-slate-600 uppercase cursor-pointer hover:bg-slate-100"
                            onClick={() => toggleSort('created_at')}
                          >
                            <div className="flex items-center space-x-1">
                              <span>Created</span>
                              <SortIcon field="created_at" />
                            </div>
                          </th>
                          <th className="px-6 py-3 text-right text-xs font-semibold text-slate-600 uppercase">
                            Actions
                          </th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-200">
                        {filteredInstances.map((i) => (
                          <tr key={i.id} className="hover:bg-slate-50">
                            <td className="px-6 py-4">
                              <div className="flex items-center space-x-3">
                                <div className="w-8 h-8 bg-primary-100 rounded-lg flex items-center justify-center">
                                  <Database className="w-4 h-4 text-primary-600" />
                                </div>
                                <div>
                                  <p className="text-sm font-medium text-slate-900">{i.db_name}</p>
                                  <p className="text-xs text-slate-500">{i.username}@{i.host}:{i.port}</p>
                                </div>
                              </div>
                            </td>
                            <td className="px-6 py-4">
                              <span className="text-sm text-slate-600">{i.user_email}</span>
                            </td>
                            <td className="px-6 py-4">
                              <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(i.status)}`}>
                                {i.status}
                              </span>
                            </td>
                            <td className="px-6 py-4">
                              <div className="flex items-center space-x-1 text-sm text-slate-600">
                                <Clock className="w-3 h-3" />
                                <span>{formatDate(i.created_at)}</span>
                              </div>
                            </td>
                            <td className="px-6 py-4 text-right">
                              <button
                                onClick={() => handleSuspendInstance(i.id, i.status !== 'suspended')}
                                className={`inline-flex items-center space-x-1 px-3 py-1 rounded text-sm font-medium ${
                                  i.status === 'suspended'
                                    ? 'bg-green-100 text-green-700 hover:bg-green-200'
                                    : 'bg-red-100 text-red-700 hover:bg-red-200'
                                }`}
                              >
                                {i.status === 'suspended' ? (
                                  <>
                                    <Activity className="w-3 h-3" />
                                    <span>Activate</span>
                                  </>
                                ) : (
                                  <>
                                    <Server className="w-3 h-3" />
                                    <span>Suspend</span>
                                  </>
                                )}
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  {filteredInstances.length === 0 && (
                    <div className="p-8 text-center text-slate-500">
                      No instances found matching "{searchQuery}"
                    </div>
                  )}
                </div>
              </div>
            )}
          </>
        )}
      </main>
    </div>
  )
}
