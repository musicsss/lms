import { useState } from 'react';
import { Link } from 'react-router-dom';
import { LogIn } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import { api } from '../api/client';
import './AuthPage.css';

export default function LoginPage() {
  const { login, loading } = useAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [needCaptcha, setNeedCaptcha] = useState(false);
  const [captchaId, setCaptchaId] = useState('');
  const [captchaQuestion, setCaptchaQuestion] = useState('');
  const [captchaAnswer, setCaptchaAnswer] = useState('');
  const [blockedUntil, setBlockedUntil] = useState(null);

  const fetchCaptcha = async () => {
    try {
      const data = await api.getCaptcha();
      setCaptchaId(data.captcha_id);
      setCaptchaQuestion(data.question);
      setNeedCaptcha(true);
    } catch (e) {
      setError('Failed to load captcha');
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');

    if (blockedUntil && Date.now() < blockedUntil * 1000) {
      const remaining = Math.ceil((blockedUntil * 1000 - Date.now()) / 60000);
      setError(`Too many attempts. Please try again in ${remaining} minute(s).`);
      return;
    }

    const loginBody = { username, password };
    if (needCaptcha && captchaId) {
      loginBody.captcha_id = captchaId;
      loginBody.captcha_answer = captchaAnswer;
    }

    const result = await login(loginBody);
    if (result.success) return;

    setError(result.error);

    if (result.needCaptcha) {
      setNeedCaptcha(true);
      if (!captchaId) fetchCaptcha();
    }
    if (result.blockedUntil) {
      setBlockedUntil(result.blockedUntil);
    }
    if (result.needCaptcha === false) {
      setCaptchaAnswer('');
    }
  };

  return (
    <div className="auth-page">
      <form className="auth-form" onSubmit={handleSubmit}>
        <h1 className="auth-title">登录 LMS</h1>
        {error && <div className="auth-error">{error}</div>}
        <label>
          <span>用户名</span>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="请输入用户名"
            required
          />
        </label>
        <label>
          <span>密码</span>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="请输入密码"
            required
          />
        </label>
        {needCaptcha && (
          <>
            <label>
              <span>{captchaQuestion || '验证码'}</span>
              <div style={{ display: 'flex', gap: 8 }}>
                <input
                  type="text"
                  value={captchaAnswer}
                  onChange={(e) => setCaptchaAnswer(e.target.value)}
                  placeholder="输入答案"
                  required
                  style={{ flex: 1 }}
                />
                <button type="button" className="btn-ghost" onClick={fetchCaptcha}
                  style={{ whiteSpace: 'nowrap', padding: '8px 12px', border: '1px solid var(--border)', borderRadius: 6, cursor: 'pointer' }}>
                  刷新
                </button>
              </div>
            </label>
          </>
        )}
        <button type="submit" disabled={loading} className="btn-primary">
          <LogIn size={16} />
          {loading ? '登录中...' : '登录'}
        </button>
        <p className="auth-switch">
          还没有账号？<Link to="/register">立即注册</Link>
        </p>
      </form>
    </div>
  );
}
