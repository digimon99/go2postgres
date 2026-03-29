import { useState, useRef, useEffect } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { Mail, ArrowLeft, Loader2 } from 'lucide-react'
import { api, ApiException } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

export default function SignUp() {
  const location = useLocation()
  const initialEmail = (location.state as { email?: string })?.email || ''
  
  const [step, setStep] = useState<'email' | 'otp'>(initialEmail ? 'otp' : 'email')
  const [email, setEmail] = useState(initialEmail)
  const [otp, setOtp] = useState(['', '', '', '', '', ''])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [resendTimer, setResendTimer] = useState(initialEmail ? 60 : 0)
  const otpRefs = useRef<(HTMLInputElement | null)[]>([])
  const navigate = useNavigate()
  const { login } = useAuth()

  useEffect(() => {
    if (resendTimer > 0) {
      const timer = setTimeout(() => setResendTimer(resendTimer - 1), 1000)
      return () => clearTimeout(timer)
    }
  }, [resendTimer])

  useEffect(() => {
    if (step === 'otp') {
      otpRefs.current[0]?.focus()
    }
  }, [step])

  async function handleSendOTP(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      const response = await api.sendOTP(email)
      if (!response.is_new_user) {
        // Redirect to signin if existing user
        navigate('/signin', { state: { email } })
        return
      }
      setStep('otp')
      setResendTimer(60)
    } catch (err) {
      if (err instanceof ApiException) {
        setError(err.message)
      } else {
        setError('Failed to send verification code. Please try again.')
      }
    } finally {
      setIsLoading(false)
    }
  }

  async function handleVerifyOTP() {
    const code = otp.join('')
    if (code.length !== 6) return

    setError('')
    setIsLoading(true)

    try {
      const response = await api.verifyOTP(email, code)
      login(response.access_token, response.refresh_token, response.user)
      navigate('/dashboard')
    } catch (err) {
      if (err instanceof ApiException) {
        setError(err.message)
      } else {
        setError('Verification failed. Please try again.')
      }
      setOtp(['', '', '', '', '', ''])
      otpRefs.current[0]?.focus()
    } finally {
      setIsLoading(false)
    }
  }

  function handleOtpChange(index: number, value: string) {
    if (!/^\d*$/.test(value)) return

    const newOtp = [...otp]
    newOtp[index] = value.slice(-1)
    setOtp(newOtp)

    if (value && index < 5) {
      otpRefs.current[index + 1]?.focus()
    }

    if (index === 5 && value) {
      const code = newOtp.join('')
      if (code.length === 6) {
        setTimeout(() => handleVerifyOTP(), 100)
      }
    }
  }

  function handleOtpKeyDown(index: number, e: React.KeyboardEvent) {
    if (e.key === 'Backspace' && !otp[index] && index > 0) {
      otpRefs.current[index - 1]?.focus()
    }
  }

  function handleOtpPaste(e: React.ClipboardEvent) {
    e.preventDefault()
    const pasted = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, 6)
    if (pasted.length === 6) {
      setOtp(pasted.split(''))
      otpRefs.current[5]?.focus()
      setTimeout(() => {
        api.verifyOTP(email, pasted).then((response) => {
          login(response.access_token, response.refresh_token, response.user)
          navigate('/dashboard')
        }).catch((err) => {
          if (err instanceof ApiException) {
            setError(err.message)
          } else {
            setError('Verification failed. Please try again.')
          }
          setOtp(['', '', '', '', '', ''])
          otpRefs.current[0]?.focus()
        })
      }, 100)
    }
  }

  async function handleResendOTP() {
    if (resendTimer > 0) return
    setError('')
    setIsLoading(true)

    try {
      await api.sendOTP(email)
      setResendTimer(60)
    } catch (err) {
      if (err instanceof ApiException) {
        setError(err.message)
      }
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-50 to-white flex flex-col">
      {/* Header */}
      <header className="p-6">
        <Link to="/" className="inline-flex items-center text-slate-600 hover:text-slate-900">
          <ArrowLeft className="w-4 h-4 mr-2" />
          Back to home
        </Link>
      </header>

      {/* Main */}
      <main className="flex-1 flex items-center justify-center px-4">
        <div className="w-full max-w-md">
          {/* Logo */}
          <div className="text-center mb-8">
            <Link to="/" className="inline-flex items-center space-x-2">
              <div className="w-12 h-12 bg-gradient-to-br from-primary-500 to-accent-500 rounded-xl flex items-center justify-center">
                <span className="text-white font-bold text-xl">g2</span>
              </div>
            </Link>
            <h1 className="mt-6 text-2xl font-bold text-slate-900">Create your account</h1>
            <p className="mt-2 text-slate-600">
              {step === 'email'
                ? 'Enter your email to get started'
                : `Enter the code sent to ${email}`}
            </p>
          </div>

          {/* Error */}
          {error && (
            <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
              {error}
            </div>
          )}

          {/* Email Step */}
          {step === 'email' && (
            <form onSubmit={handleSendOTP}>
              <div className="mb-6">
                <label htmlFor="email" className="block text-sm font-medium text-slate-700 mb-2">
                  Email address
                </label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
                  <input
                    id="email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="w-full pl-11 pr-4 py-3 border border-slate-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none transition-all"
                    placeholder="you@example.com"
                    required
                    autoFocus
                  />
                </div>
              </div>
              <button
                type="submit"
                disabled={isLoading || !email}
                className="w-full py-3 bg-gradient-to-r from-primary-500 to-accent-500 text-white rounded-lg font-semibold hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center"
              >
                {isLoading ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  'Continue with Email'
                )}
              </button>
              <p className="mt-4 text-center text-sm text-slate-500">
                By signing up, you agree to our Terms of Service and Privacy Policy.
              </p>
            </form>
          )}

          {/* OTP Step */}
          {step === 'otp' && (
            <div>
              <div className="flex justify-center gap-3 mb-6" onPaste={handleOtpPaste}>
                {otp.map((digit, index) => (
                  <input
                    key={index}
                    ref={(el) => (otpRefs.current[index] = el)}
                    type="text"
                    inputMode="numeric"
                    maxLength={1}
                    value={digit}
                    onChange={(e) => handleOtpChange(index, e.target.value)}
                    onKeyDown={(e) => handleOtpKeyDown(index, e)}
                    className="otp-input"
                    disabled={isLoading}
                  />
                ))}
              </div>
              <button
                onClick={handleVerifyOTP}
                disabled={isLoading || otp.join('').length !== 6}
                className="w-full py-3 bg-gradient-to-r from-primary-500 to-accent-500 text-white rounded-lg font-semibold hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center"
              >
                {isLoading ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  'Create Account'
                )}
              </button>
              <div className="mt-6 text-center">
                <p className="text-slate-600 mb-2">Didn't receive the code?</p>
                <button
                  onClick={handleResendOTP}
                  disabled={resendTimer > 0 || isLoading}
                  className="text-primary-600 hover:text-primary-700 font-medium disabled:text-slate-400"
                >
                  {resendTimer > 0 ? `Resend in ${resendTimer}s` : 'Resend code'}
                </button>
              </div>
              <button
                onClick={() => {
                  setStep('email')
                  setOtp(['', '', '', '', '', ''])
                  setError('')
                }}
                className="mt-4 w-full py-2 text-slate-600 hover:text-slate-900"
              >
                Use a different email
              </button>
            </div>
          )}

          {/* Footer */}
          <p className="mt-8 text-center text-slate-600">
            Already have an account?{' '}
            <Link to="/signin" className="text-primary-600 hover:text-primary-700 font-medium">
              Sign in
            </Link>
          </p>
        </div>
      </main>
    </div>
  )
}
