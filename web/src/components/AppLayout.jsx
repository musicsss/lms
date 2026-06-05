import { NavLink, Outlet } from 'react-router-dom';
import { Home, FolderOpen, MessageSquare, LogOut, Cloud } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import './AppLayout.css';

export default function AppLayout() {
  const { user, logout } = useAuth();

  return (
    <div className="app-layout">
      <aside className="sidebar">
        <div className="sidebar-brand">
          <Cloud size={22} />
          <span>LMS</span>
        </div>
        <nav className="sidebar-nav">
          <NavLink to="/home" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
            <Home size={18} />
            <span>首页</span>
          </NavLink>
          <NavLink to="/files" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
            <FolderOpen size={18} />
            <span>网盘</span>
          </NavLink>
          <NavLink to="/forum" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
            <MessageSquare size={18} />
            <span>论坛</span>
          </NavLink>
        </nav>
      </aside>
      <div className="main-area">
        <header className="topbar">
          <div className="topbar-title">LMS</div>
          <div className="topbar-right">
            <span className="topbar-user">{user?.username}</span>
            <button className="btn-icon" onClick={logout} title="退出登录">
              <LogOut size={18} />
            </button>
          </div>
        </header>
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
