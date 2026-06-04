import { useState, useEffect, useCallback } from 'react'
import { Trash2, File, Folder } from 'lucide-react'
import { api } from '../api/client'

function formatSize(bytes) {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(1)} ${units[i]}`
}

export default function FilesPage() {
  const [files, setFiles] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [toast, setToast] = useState(null)

  const totalPages = Math.max(1, Math.ceil(total / 20))

  const fetchFiles = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.listFiles(page)
      setFiles(data.files)
      setTotal(data.total)
    } catch (e) {
      showToast(e.message, 'error')
    } finally {
      setLoading(false)
    }
  }, [page])

  useEffect(() => { fetchFiles() }, [fetchFiles])

  const showToast = (msg, type) => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  const handleDelete = async (id) => {
    if (!confirm('Delete this file? This will also delete all children if it is a directory.')) return
    try {
      await api.deleteFile(id)
      showToast('File deleted', 'success')
      fetchFiles()
    } catch (e) {
      showToast(e.message, 'error')
    }
  }

  return (
    <div>
      {toast && <div className={`toast toast-${toast.type}`}>{toast.msg}</div>}

      <div className="page-header">
        <h1>Files</h1>
        <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{total} total</span>
      </div>

      <div className="card" style={{ padding: 0, overflow: 'auto' }}>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Name</th>
              <th>Type</th>
              <th>Size</th>
              <th>Owner</th>
              <th>Created</th>
              <th style={{ width: 80 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={7} style={{ textAlign: 'center', padding: 32 }}>Loading...</td></tr>
            ) : files.length === 0 ? (
              <tr><td colSpan={7} style={{ textAlign: 'center', padding: 32, color: '#999' }}>No files found</td></tr>
            ) : files.map((f) => (
              <tr key={f.id}>
                <td style={{ color: '#999', fontSize: 12 }}>{f.id}</td>
                <td style={{ fontWeight: 500, display: 'flex', alignItems: 'center', gap: 8 }}>
                  {f.is_dir ? <Folder size={16} color="#d97706" /> : <File size={16} color="#666" />}
                  {f.name}
                </td>
                <td style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                  {f.is_dir ? 'Directory' : (f.mime_type || 'File')}
                </td>
                <td style={{ fontSize: 13 }}>{f.is_dir ? '-' : formatSize(f.size)}</td>
                <td style={{ fontSize: 13 }}>{f.user?.username || `#${f.user_id}`}</td>
                <td style={{ color: 'var(--text-secondary)', fontSize: 12 }}>
                  {new Date(f.created_at).toLocaleDateString()}
                </td>
                <td>
                  <button className="btn btn-sm btn-danger" title="Delete"
                    onClick={() => handleDelete(f.id)}>
                    <Trash2 size={14} />
                  </button>
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
