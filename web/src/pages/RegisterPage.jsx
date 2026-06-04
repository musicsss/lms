import { useState } from 'react';
import { Link } from 'react-router-dom';
import { UserPlus } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import './AuthPage.css';

export default function RegisterPage() {
  const { register, loading } = useAuth();
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    const result = await register(username, password, email);
    if (!result.success) setError(result.error);
  };

  return (
    <div className="auth-page">
      <form className="auth-form" onSubmit={handleSubmit}>
        <h1 className="auth-title">注册 LMS</h1>
        {error && <div className="auth-error">{error}</div>}
        <label>
          <span>用户名</span>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="2-64个字符"
            required
            minLength={2}
            maxLength={64}
          />
        </label>
        <label>
          <span>邮箱（选填）</span>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="your@email.com"
          />
        </label>
        <label>
          <span>密码</span>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="至少6个字符"
            required
            minLength={6}
          />
        </label>
        <button type="submit" disabled={loading} className="btn-primary">
          <UserPlus size={16} />
          {loading ? '注册中...' : '注册'}
        </button>
        <p className="auth-switch">
          已有账号？<Link to="/login">立即登录</Link>
        </p>
      </form>
    </div>
  );
}
