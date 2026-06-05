import { NavLink, Outlet } from 'react-router-dom';
import { LogOut } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import './AppLayout.css';

export default function AppLayout() {
  const { user, logout } = useAuth();

  return (
    <div className="app-layout">
      <header className="bili-header">
        <NavLink to="/home" className="bili-logo" style={{ textDecoration: 'none' }}>LMS</NavLink>
        <nav className="bili-nav">
          <NavLink to="/home" className={({ isActive }) => isActive ? 'bili-nav-link active' : 'bili-nav-link'}>
            推荐
          </NavLink>
          <NavLink to="/files" className={({ isActive }) => isActive ? 'bili-nav-link active' : 'bili-nav-link'}>
            网盘
          </NavLink>
          <NavLink to="/forum" className={({ isActive }) => isActive ? 'bili-nav-link active' : 'bili-nav-link'}>
            论坛
          </NavLink>
        </nav>
        <div className="bili-header-right">
          <NavLink to="/user/profile" className="bili-user-avatar">
            {user?.avatar_url ? (
              <img src={user.avatar_url} alt={user.username} className="bili-user-avatar-img" />
            ) : (
              <div className="bili-user-avatar-placeholder">
                {user?.username?.charAt(0).toUpperCase() || 'U'}
              </div>
            )}
          </NavLink>
          <button className="bili-logout-btn" onClick={logout}>
            <LogOut size={16} />
            退出
          </button>
        </div>
      </header>
      <div className="bili-content">
        <Outlet />
      </div>
    </div>
  );
}