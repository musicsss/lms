import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import Dashboard from './pages/Dashboard'
import UsersPage from './pages/UsersPage'
import FilesPage from './pages/FilesPage'
import ForumPage from './pages/ForumPage'
import ConfigPage from './pages/ConfigPage'
import DBManagerPage from './pages/DBManager'
import DanmakuPage from './pages/DanmakuPage'
import AuditPage from './pages/AuditPage'
import AdminLayout from './components/AdminLayout'

function ProtectedRoute({ children }) {
  const token = localStorage.getItem('admin_token')
  if (!token) return <Navigate to="/login" replace />
  return children
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<ProtectedRoute><AdminLayout /></ProtectedRoute>}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="users" element={<UsersPage />} />
          <Route path="files" element={<FilesPage />} />
          <Route path="forum" element={<ForumPage />} />
          <Route path="danmaku" element={<DanmakuPage />} />
          <Route path="config" element={<ConfigPage />} />
          <Route path="db" element={<DBManagerPage />} />
          <Route path="audit" element={<AuditPage />} />
        </Route>
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </BrowserRouter>
  )
}