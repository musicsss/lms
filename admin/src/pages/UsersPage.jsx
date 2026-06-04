import { useState, useEffect, useCallback } from 'react'
import { Trash2, Search, Shield, UserX } from 'lucide-react'
import { api } from '../api/client'

export default function UsersPage() {
  const [users, setUsers] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const [toast, setToast] = useState(null)

  const totalPages = Math.max(1, Math.ceil(total / 20))

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.listUsers(page, search)
      setUsers(data.users)
      setTotal(data.total)
    } catch (e) {
      showToast(e.message, 'error')
    } finally {
      setLoading(false)
    }
  }, [page, search])

  useEffect(() => { fetchUsers() }, [fetchUsers])

  const showToast = (msg, type) => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  const handleUpdateRole = async (id, role) => {
    try {
      await api.updateUserRole(id, role)
      showToast('Role updated', 'success')
      fetchUsers()
    } catch (e) {
      showToast(e.message, 'error')
    }
  }

  const handleDelete = async (id) => {
    if (!confirm('Delete this user?')) return
    try {
      await api.deleteUser(id)
      showToast('User deleted', 'success')
      fetchUsers()
    } catch (e) {
      showToast(e.message, 'error')
    }
  }

  return (
    <div>
      {toast && <div className={`toast toast-${toast.type}`}>{toast.msg}</div>}

      <div className="page-header">
        <h1>Users</h1>
        <div style={{ position: 'relative' }}>
          <Search size={16} style={{ position: 'absolute', left: 10, top: 10, color: '#999' }} />
          <input
            placeholder="Search..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1) }}
            style={{ padding: '8px 10px 8px 32px', border: '1px solid var(--border)', borderRadius: 'var(--radius)', width: 220 }}
          />
        </div>
      </div>

      <div className="card" style={{ padding: 0, overflow: 'auto' }}>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Username</th>
              <th>Email</th>
              <th>Role</th>
              <th>Created</th>
              <th style={{ width: 140 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={6} style={{ textAlign: 'center', padding: 32 }}>Loading...</td></tr>
            ) : users.length === 0 ? (
              <tr><td colSpan={6} style={{ textAlign: 'center', padding: 32, color: '#999' }}>No users found</td></tr>
            ) : users.map((u) => (
              <tr key={u.id}>
                <td style={{ color: '#999', fontSize: 12 }}>{u.id}</td>
                <td style={{ fontWeight: 500 }}>{u.username}</td>
                <td style={{ color: 'var(--text-secondary)' }}>{u.email || '-'}</td>
                <td>
                  <span className={`badge ${u.role === 'admin' ? 'badge-admin' : 'badge-user'}`}>
                    {u.role}
                  </span>
                </td>
                <td style={{ color: 'var(--text-secondary)', fontSize: 12 }}>
                  {new Date(u.created_at).toLocaleDateString()}
                </td>
                <td>
                  <div style={{ display: 'flex', gap: 4 }}>
                    {u.role === 'user' ? (
                      <button className="btn btn-sm btn-ghost" title="Promote to admin"
                        onClick={() => handleUpdateRole(u.id, 'admin')}>
                        <Shield size={14} />
                      </button>
                    ) : (
                      <button className="btn btn-sm btn-ghost" title="Demote to user"
                        onClick={() => handleUpdateRole(u.id, 'user')}>
                        <UserX size={14} />
                      </button>
                    )}
                    <button className="btn btn-sm btn-danger" title="Delete"
                      onClick={() => handleDelete(u.id)}>
                      <Trash2 size={14} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {totalPages > 1 && (
          <div className="pagination" style={{ padding: '12px 16px', justifyContent: 'center' }}>
            <button disabled={page <= 1} onClick={() => setPage(page - 1)}>Prev</button>
            <span style={{ fontSize: 13, color: 'var(--text-secondary)', padding: '0 12px' }}>
              {page} / {totalPages}
            </span>
            <button disabled={page >= totalPages} onClick={() => setPage(page + 1)}>Next</button>
          </div>
        )}
      </div>
    </div>
  )
}
