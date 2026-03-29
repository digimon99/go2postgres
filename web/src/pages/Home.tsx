import { Link } from 'react-router-dom'
import { Database, Shield, Zap, Server, Code, Users, ArrowRight, Check } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'

const features = [
  {
    icon: Zap,
    title: 'Instant Provisioning',
    description: 'Spin up isolated PostgreSQL databases in under 5 seconds via API or dashboard.',
  },
  {
    icon: Shield,
    title: 'Secure by Default',
    description: 'AES-256 encryption, per-database credentials, and automatic role isolation.',
  },
  {
    icon: Server,
    title: 'Single Binary',
    description: 'No Kubernetes, no Terraform. Just one Go binary under 50MB with <100MB RAM.',
  },
  {
    icon: Database,
    title: 'Multi-Tenant',
    description: 'Support 100+ isolated databases on a single server with per-user access control.',
  },
  {
    icon: Code,
    title: 'Full REST API',
    description: 'Everything available via API. Build integrations, automate workflows, go headless.',
  },
  {
    icon: Users,
    title: 'Team Ready',
    description: 'User management, admin controls, audit logs, and role-based access out of the box.',
  },
]

const benefits = [
  'No vendor lock-in',
  'Self-hosted & private',
  'Open source',
  'PostgreSQL 16 support',
  'Extension support (pgcrypto, uuid-ossp, etc.)',
  'Connection string management',
]

export default function Home() {
  const { user } = useAuth()

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-50 to-white">
      {/* Header */}
      <header className="fixed top-0 left-0 right-0 bg-white/80 backdrop-blur-md border-b border-slate-200 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <Link to="/" className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-gradient-to-br from-primary-500 to-accent-500 rounded-lg flex items-center justify-center">
                <span className="text-white font-bold text-sm">g2</span>
              </div>
              <span className="font-semibold text-slate-900">go2postgres</span>
            </Link>
            <nav className="flex items-center space-x-4">
              {user ? (
                <Link
                  to="/dashboard"
                  className="px-4 py-2 bg-primary-500 text-white rounded-lg font-medium hover:bg-primary-600 transition-colors"
                >
                  Dashboard
                </Link>
              ) : (
                <>
                  <Link to="/signin" className="text-slate-600 hover:text-slate-900 font-medium">
                    Sign In
                  </Link>
                  <Link
                    to="/signup"
                    className="px-4 py-2 bg-primary-500 text-white rounded-lg font-medium hover:bg-primary-600 transition-colors"
                  >
                    Get Started
                  </Link>
                </>
              )}
            </nav>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="pt-32 pb-20 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto text-center">
          <div className="inline-flex items-center px-3 py-1 rounded-full bg-primary-50 text-primary-700 text-sm font-medium mb-6">
            <Zap className="w-4 h-4 mr-1" />
            Self-hosted PostgreSQL provisioning
          </div>
          <h1 className="text-5xl sm:text-6xl lg:text-7xl font-bold text-slate-900 mb-6 tracking-tight">
            PostgreSQL databases
            <br />
            <span className="gradient-text">without the complexity</span>
          </h1>
          <p className="text-xl text-slate-600 max-w-3xl mx-auto mb-10">
            Provision isolated PostgreSQL databases in seconds. Simple API, modern dashboard, 
            zero vendor lock-in. Perfect for SaaS platforms, agencies, and development teams.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              to="/signup"
              className="w-full sm:w-auto px-8 py-4 bg-gradient-to-r from-primary-500 to-accent-500 text-white rounded-xl font-semibold text-lg hover:opacity-90 transition-opacity flex items-center justify-center"
            >
              Start Free <ArrowRight className="ml-2 w-5 h-5" />
            </Link>
            <a
              href="https://github.com/digimon99/go2postgres"
              target="_blank"
              rel="noopener noreferrer"
              className="w-full sm:w-auto px-8 py-4 bg-slate-900 text-white rounded-xl font-semibold text-lg hover:bg-slate-800 transition-colors flex items-center justify-center"
            >
              <svg className="w-5 h-5 mr-2" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
              </svg>
              View on GitHub
            </a>
          </div>
        </div>
      </section>

      {/* Feature Preview */}
      <section className="py-20 px-4 sm:px-6 lg:px-8 bg-slate-900">
        <div className="max-w-7xl mx-auto">
          <div className="bg-slate-800 rounded-2xl border border-slate-700 overflow-hidden shadow-2xl">
            <div className="flex items-center gap-2 px-4 py-3 bg-slate-900 border-b border-slate-700">
              <div className="w-3 h-3 rounded-full bg-red-500"></div>
              <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
              <div className="w-3 h-3 rounded-full bg-green-500"></div>
              <span className="ml-2 text-slate-400 text-sm font-mono">Terminal</span>
            </div>
            <div className="p-6 font-mono text-sm">
              <p className="text-slate-400"># Create a new database in seconds</p>
              <p className="text-green-400 mt-2">
                $ curl -X POST https://your-server:8443/api/v1/instances \
              </p>
              <p className="text-green-400">
                &nbsp;&nbsp;-H "Authorization: Bearer $TOKEN" \
              </p>
              <p className="text-green-400">
                &nbsp;&nbsp;-d '{`{"project_id": "my-app"}`}'
              </p>
              <p className="text-slate-300 mt-4">
                {`{
  "instance_id": "inst_abc123",
  "database_name": "db_my_app",
  "host": "localhost",
  "port": 5438,
  "username": "u_my_app",
  "password": "SecurePassword123!",
  "connection_string": "postgres://u_my_app:...@localhost:5438/db_my_app"
}`}
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="py-20 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold text-slate-900 mb-4">
              Everything you need for database provisioning
            </h2>
            <p className="text-lg text-slate-600 max-w-2xl mx-auto">
              Built for developers who want PostgreSQL without the operational overhead.
            </p>
          </div>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
            {features.map((feature) => (
              <div
                key={feature.title}
                className="bg-white rounded-2xl p-8 border border-slate-200 card-hover"
              >
                <div className="w-12 h-12 bg-gradient-to-br from-primary-100 to-accent-100 rounded-xl flex items-center justify-center mb-6">
                  <feature.icon className="w-6 h-6 text-primary-600" />
                </div>
                <h3 className="text-xl font-semibold text-slate-900 mb-3">{feature.title}</h3>
                <p className="text-slate-600">{feature.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Benefits */}
      <section className="py-20 px-4 sm:px-6 lg:px-8 bg-gradient-to-br from-primary-500 to-accent-600">
        <div className="max-w-7xl mx-auto">
          <div className="grid lg:grid-cols-2 gap-12 items-center">
            <div>
              <h2 className="text-3xl sm:text-4xl font-bold text-white mb-6">
                Why choose go2postgres?
              </h2>
              <p className="text-xl text-primary-100 mb-8">
                A modern, lightweight alternative to managed database services and complex IaC tools.
              </p>
              <ul className="space-y-4">
                {benefits.map((benefit) => (
                  <li key={benefit} className="flex items-center text-white">
                    <Check className="w-5 h-5 mr-3 text-primary-200" />
                    {benefit}
                  </li>
                ))}
              </ul>
            </div>
            <div className="bg-white/10 backdrop-blur-sm rounded-2xl p-8 border border-white/20">
              <h3 className="text-2xl font-bold text-white mb-6">Quick comparison</h3>
              <div className="space-y-4">
                <div className="flex justify-between text-white border-b border-white/20 pb-4">
                  <span>Supabase</span>
                  <span className="text-primary-200">Vendor lock-in, $$$ at scale</span>
                </div>
                <div className="flex justify-between text-white border-b border-white/20 pb-4">
                  <span>Terraform/Pulumi</span>
                  <span className="text-primary-200">Complex, state files</span>
                </div>
                <div className="flex justify-between text-white border-b border-white/20 pb-4">
                  <span>Crossplane</span>
                  <span className="text-primary-200">Requires Kubernetes</span>
                </div>
                <div className="flex justify-between text-white">
                  <span className="font-semibold">go2postgres</span>
                  <span className="text-green-300 font-semibold">Simple, fast, free</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-20 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-3xl sm:text-4xl font-bold text-slate-900 mb-6">
            Ready to simplify your database provisioning?
          </h2>
          <p className="text-xl text-slate-600 mb-10">
            Get started in minutes. No credit card required.
          </p>
          <Link
            to="/signup"
            className="inline-flex items-center px-8 py-4 bg-gradient-to-r from-primary-500 to-accent-500 text-white rounded-xl font-semibold text-lg hover:opacity-90 transition-opacity"
          >
            Create your first database <ArrowRight className="ml-2 w-5 h-5" />
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 px-4 sm:px-6 lg:px-8 border-t border-slate-200">
        <div className="max-w-7xl mx-auto">
          <div className="flex flex-col md:flex-row items-center justify-between gap-4">
            <div className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-gradient-to-br from-primary-500 to-accent-500 rounded-lg flex items-center justify-center">
                <span className="text-white font-bold text-sm">g2</span>
              </div>
              <span className="text-slate-600">© 2025 go2postgres. Open source under MIT.</span>
            </div>
            <div className="flex items-center space-x-6 text-slate-600">
              <a href="https://github.com/digimon99/go2postgres" target="_blank" rel="noopener noreferrer" className="hover:text-slate-900">
                GitHub
              </a>
              <a href="https://github.com/digimon99/go2postgres#readme" target="_blank" rel="noopener noreferrer" className="hover:text-slate-900">
                Documentation
              </a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  )
}
