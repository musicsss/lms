import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { LogIn, Shield } from 'lucide-react'
import { api } from '../api/client'

export default function LoginPage() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [needCaptcha, setNeedCaptcha] = useState(false)
  const [captchaId, setCaptchaId] = useState('')
  const [captchaQuestion, setCaptchaQuestion] = useState('')
  const [captchaAnswer, setCaptchaAnswer] = useState('')
  const [blockedUntil, setBlockedUntil] = useState(null)

  const fetchCaptcha = async () => {
    try {
      const data = await api.getCaptcha()
      setCaptchaId(data.captcha_id)
      setCaptchaQuestion(data.question)
      setNeedCaptcha(true)
    } catch (e) {
      setError('Failed to load captcha')
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    if (blockedUntil && Date.now() < blockedUntil * 1000) {
      const remaining = Math.ceil((blockedUntil * 1000 - Date.now()) / 60000)
      setError(`Too many attempts. Please try again in ${remaining} minute(s).`)
      setLoading(false)
      return
    }

    try {
      const loginBody = { username, password }
      if (needCaptcha && captchaId) {
        loginBody.captcha_id = captchaId
        loginBody.captcha_answer = captchaAnswer
      }

      const data = await api.login(loginBody)
      if (data.user.role !== 'admin') {
        setError('Admin access required')
        setLoading(false)
        return
      }
      localStorage.setItem('admin_token', data.token)
      localStorage.setItem('admin_user', JSON.stringify(data.user))
      navigate('/dashboard')
    } catch (err) {
      setError(err.message)

      if (err.data?.need_captcha) {
        setNeedCaptcha(true)
        if (!captchaId) fetchCaptcha()
      }
      if (err.data?.blocked_until) {
        setBlockedUntil(err.data.blocked_until)
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: '#f0f2f5',
    }}>
      <form onSubmit={handleSubmit} style={{
        background: '#fff',
        padding: 40,
        borderRadius: 8,
        width: 380,
        boxShadow: '0 2px 12px rgba(0,0,0,0.08)',
      }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <div style={{
            width: 48,
            height: 48,
            borderRadius: 12,
            background: '#1e293b',
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginBottom: 12,
          }}>
            <Shield size={24} color="#fff" />
          </div>
          <h1 style={{ fontSize: 22, fontWeight: 700 }}>LMS Admin</h1>
          <p style={{ color: '#666', fontSize: 13, marginTop: 4 }}>Sign in to manage the system</p>
        </div>

        {error && (
          <div style={{
            background: '#fef2f2',
            color: '#dc2626',
            padding: '8px 12px',
            borderRadius: 6,
            fontSize: 13,
            marginBottom: 16,
          }}>{error}</div>
        )}

        <div className="form-group">
          <label>Username</label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
            autoFocus
          />
        </div>

        <div className="form-group">
          <label>Password</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>

        {needCaptcha && (
          <div className="form-group">
            <label>{captchaQuestion || 'Captcha'}</label>
            <div style={{ display: 'flex', gap: 8 }}>
              <input
                type="text"
                value={captchaAnswer}
                onChange={(e) => setCaptchaAnswer(e.target.value)}
                placeholder="Answer"
                required
                style={{ flex: 1 }}
              />
              <button type="button" className="btn btn-sm btn-ghost" onClick={fetchCaptcha}
                style={{ whiteSpace: 'nowrap' }}>
                Refresh
              </button>
            </div>
          </div>
        )}

        <button
          type="submit"
          disabled={loading}
          className="btn btn-primary"
          style={{ width: '100%', marginTop: 8, justifyContent: 'center' }}
        >
          <LogIn size={16} />
          {loading ? 'Signing in...' : 'Sign In'}
        </button>
      </form>
    </div>
  )
}
