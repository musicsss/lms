import { useState } from 'react';
import { Link } from 'react-router-dom';
import { LogIn } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import './AuthPage.css';

export default function LoginPage() {
  const { login, loading } = useAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    const result = await login(username, password);
    if (!result.success) setError(result.error);
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
