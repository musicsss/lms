import { useState, useEffect } from 'react'
import { Outlet, Link, useNavigate, useLocation } from 'react-router-dom'
import { LayoutDashboard, Users, FolderOpen, MessageSquare, Settings, LogOut, Shield, Database, MessageCircle } from 'lucide-react'

const navItems = [
  { path: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { path: '/users', label: 'Users', icon: Users },
  { path: '/files', label: 'Files', icon: FolderOpen },
  { path: '/forum', label: 'Forum', icon: MessageSquare },
  { path: '/danmaku', label: 'Danmaku', icon: MessageCircle },
  { path: '/config', label: 'Config', icon: Settings },
  { path: '/db', label: 'Database', icon: Database },
]

export default function AdminLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const [user, setUser] = useState(null)

  useEffect(() => {
    const stored = localStorage.getItem('admin_user')
    if (stored) setUser(JSON.parse(stored))
  }, [])

  const handleLogout = () => {
    localStorage.removeItem('admin_token')
    localStorage.removeItem('admin_user')
    navigate('/login')
  }

  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      <aside style={{
        width: 220,
        background: '#1e293b',
        color: '#e2e8f0',
        display: 'flex',
        flexDirection: 'column',
        padding: '20px 0',
      }}>
        <div style={{ padding: '0 16px 24px', borderBottom: '1px solid #334155', marginBottom: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <Shield size={22} />
            <span style={{ fontWeight: 700, fontSize: 16 }}>LMS Admin</span>
          </div>
          {user && (
            <div style={{ marginTop: 12, fontSize: 12, color: '#94a3b8' }}>
              {user.username}
            </div>
          )}
        </div>

        <nav style={{ flex: 1 }}>
          {navItems.map(({ path, label, icon: Icon }) => (
            <Link
              key={path}
              to={path}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                padding: '10px 20px',
                color: location.pathname.startsWith(path) ? '#fff' : '#94a3b8',
                background: location.pathname.startsWith(path) ? '#334155' : 'transparent',
                textDecoration: 'none',
                fontSize: 14,
                transition: 'background 0.1s',
              }}
            >
              <Icon size={18} />
              {label}
            </Link>
          ))}
        </nav>

        <div style={{ padding: '0 16px' }}>
          <button
            onClick={handleLogout}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              background: 'none',
              border: 'none',
              color: '#94a3b8',
              cursor: 'pointer',
              fontSize: 14,
              padding: '8px 0',
            }}
          >
            <LogOut size={16} />
            Logout
          </button>
        </div>
      </aside>

      <main style={{ flex: 1, padding: 32, overflow: 'auto' }}>
        <Outlet />
      </main>
    </div>
  )
}