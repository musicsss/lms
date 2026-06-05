import { useState, useEffect, useCallback } from 'react'
import { Database, Table, ChevronRight, Plus, Edit3, Trash2, X, AlertTriangle, Search } from 'lucide-react'
import { api } from '../api/client'

export default function DBManagerPage() {
  const [tables, setTables] = useState([])
  const [activeTable, setActiveTable] = useState(null)
  const [schema, setSchema] = useState(null)
  const [rows, setRows] = useState(null)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  // Form state
  const [showForm, setShowForm] = useState(false)
  const [formMode, setFormMode] = useState('create') // create | edit
  const [formValues, setFormValues] = useState({})
  const [editingId, setEditingId] = useState(null)

  // Confirm dialog
  const [confirmDelete, setConfirmDelete] = useState(null)

  // Result message
  const [message, setMessage] = useState(null)

  // Load table list
  const fetchTables = useCallback(async () => {
    try {
      const data = await api.get('/admin/db/tables')
      setTables(data.tables || [])
    } catch (e) {
      setError(e.message)
    }
  }, [])

  useEffect(() => { fetchTables() }, [fetchTables])

  // Load table schema and rows
  const loadTable = async (name) => {
    setActiveTable(name)
    setLoading(true)
    setError(null)
    setMessage(null)
    try {
      const [schemaData, rowsData] = await Promise.all([
        api.get(`/admin/db/tables/${name}`),
        api.get(`/admin/db/tables/${name}/rows?page=1&page_size=20`),
      ])
      setSchema(schemaData)
      setRows(rowsData)
      setPage(1)
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  // Load rows for a page
  const loadPage = async (p) => {
    if (!activeTable) return
    setLoading(true)
    try {
      const data = await api.get(`/admin/db/tables/${activeTable}/rows?page=${p}&page_size=20`)
      setRows(data)
      setPage(p)
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  // Open create form
  const openCreate = () => {
    const defaults = {}
    if (schema) {
      for (const col of schema.columns) {
        if (col.is_pk) continue
        defaults[col.name] = ''
      }
    }
    setFormValues(defaults)
    setFormMode('create')
    setEditingId(null)
    setShowForm(true)
    setMessage(null)
  }

  // Open edit form
  const openEdit = (row) => {
    setFormValues({ ...row })
    setFormMode('edit')
    setEditingId(row.id)
    setShowForm(true)
    setMessage(null)
  }

  // Submit form
  const handleSubmit = async () => {
    setLoading(true)
    setMessage(null)
    try {
      const cleaned = {}
      for (const [k, v] of Object.entries(formValues)) {
        if (k === 'id') continue
        if (v === '') {
          cleaned[k] = null
        } else if (v === 'true') {
          cleaned[k] = true
        } else if (v === 'false') {
          cleaned[k] = false
        } else if (!isNaN(v) && v !== '' && schema?.columns.find(c => c.name === k)?.type !== 'text') {
          cleaned[k] = Number(v)
        } else {
          cleaned[k] = v
        }
      }

      if (formMode === 'create') {
        await api.post(`/admin/db/tables/${activeTable}`, cleaned)
      } else {
        await api.put(`/admin/db/tables/${activeTable}/${editingId}`, cleaned)
      }
      setMessage({ ok: true, text: formMode === 'create' ? 'Row inserted' : 'Row updated' })
      setShowForm(false)
      loadTable(activeTable)
    } catch (e) {
      setMessage({ ok: false, text: e.message })
    } finally {
      setLoading(false)
    }
  }

  // Delete row
  const handleDelete = async () => {
    if (!confirmDelete) return
    setLoading(true)
    try {
      await api.del(`/admin/db/tables/${activeTable}/${confirmDelete.id}`)
      setMessage({ ok: true, text: `Row ID=${confirmDelete.id} deleted` })
      setConfirmDelete(null)
      loadTable(activeTable)
    } catch (e) {
      setMessage({ ok: false, text: e.message })
      setConfirmDelete(null)
    } finally {
      setLoading(false)
    }
  }

  // Format cell value for display
  const formatCell = (val) => {
    if (val === null || val === undefined) return <span style={{ color: '#94a3b8', fontStyle: 'italic' }}>NULL</span>
    if (typeof val === 'boolean') return val ? 'true' : 'false'
    if (typeof val === 'object') return JSON.stringify(val).slice(0, 80)
    const s = String(val)
    if (s.length > 200) return s.slice(0, 200) + '…'
    return s
  }

  const totalPages = rows ? Math.ceil(rows.total / rows.page_size) : 0

  return (
    <div>
      <div className="page-header">
        <h1>Database Browser</h1>
      </div>

      {/* Confirm delete modal */}
      {confirmDelete && (
        <div className="modal-overlay" onClick={() => setConfirmDelete(null)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 440 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
              <AlertTriangle size={24} color="#ef4444" />
              <h2 style={{ margin: 0 }}>危险操作</h2>
            </div>
            <p style={{ color: 'var(--text-secondary)', fontSize: 13, marginBottom: 8 }}>
              你正在删除 <strong>{activeTable}</strong> 表中 ID = <strong>{confirmDelete.id}</strong> 的记录。
            </p>
            <p style={{ color: '#ef4444', fontSize: 13, marginBottom: 16 }}>
              此操作不可撤销，可能影响关联数据和外键约束。
            </p>
            <div className="modal-actions">
              <button className="btn btn-ghost" onClick={() => setConfirmDelete(null)}>取消</button>
              <button className="btn btn-danger" onClick={handleDelete} disabled={loading}>
                {loading ? '删除中...' : '确认删除'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit/Modify warning */}
      {formMode === 'edit' && showForm && (
        <div style={{
          background: '#fef3c7', border: '1px solid #fcd34d', borderRadius: 6,
          padding: '10px 14px', marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8,
          fontSize: 13,
        }}>
          <AlertTriangle size={16} color="#d97706" />
          <span style={{ color: '#92400e' }}>
            正在修改 <strong>{activeTable}</strong> 表 ID=<strong>{editingId}</strong> 的记录，请谨慎操作。
          </span>
        </div>
      )}

      <div style={{ display: 'flex', gap: 20, minHeight: 'calc(100vh - 160px)' }}>
        {/* Left: table list */}
        <div style={{ width: 240, flexShrink: 0 }}>
          <div className="card" style={{ padding: 12 }}>
            <div style={{
              display: 'flex', alignItems: 'center', gap: 6, padding: '8px 4px',
              fontWeight: 600, fontSize: 13, borderBottom: '1px solid var(--border)',
              color: 'var(--text-secondary)', marginBottom: 4,
            }}>
              <Database size={14} />
              Tables
            </div>
            {tables.map(t => (
              <div
                key={t.name}
                onClick={() => loadTable(t.name)}
                style={{
                  display: 'flex', alignItems: 'center', gap: 8, padding: '8px 4px 8px 12px',
                  cursor: 'pointer', fontSize: 13, borderRadius: 4,
                  color: activeTable === t.name ? 'var(--primary)' : 'var(--text)',
                  background: activeTable === t.name ? '#e8f0fe' : 'transparent',
                }}
              >
                <Table size={13} />
                <span style={{ flex: 1, fontFamily: 'monospace', fontSize: 12 }}>{t.name}</span>
                <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{t.row_count}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Right: data view */}
        <div style={{ flex: 1, minWidth: 0 }}>
          {!activeTable ? (
            <div style={{
              flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
              color: 'var(--text-muted)', fontSize: 14, flexDirection: 'column', gap: 8,
              height: 300,
            }}>
              <Database size={40} style={{ opacity: 0.25 }} />
              <div>Select a table from the left panel</div>
            </div>
          ) : loading ? (
            <div className="card">Loading…</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              {/* Message */}
              {message && (
                <div className="card" style={{
                  background: message.ok ? '#f0fdf4' : '#fef2f2',
                  borderColor: message.ok ? '#bbf7d0' : '#fecaca',
                  padding: '10px 14px', fontSize: 13,
                  color: message.ok ? '#166534' : '#991b1b',
                }}>
                  {message.text}
                  <button className="btn btn-sm btn-ghost" style={{ marginLeft: 12, padding: '1px 8px' }}
                    onClick={() => setMessage(null)}>
                    <X size={12} />
                  </button>
                </div>
              )}

              {/* Schema header */}
              {schema && (
                <div className="card" style={{ padding: 16 }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Table size={16} />
                      <span style={{ fontFamily: 'monospace', fontWeight: 600, fontSize: 14 }}>{schema.name}</span>
                      <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>
                        {schema.columns.length} columns
                      </span>
                    </div>
                    <button className="btn btn-primary btn-sm" onClick={openCreate}>
                      <Plus size={14} /> Insert Row
                    </button>
                  </div>

                  {/* Column list */}
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                    {schema.columns.map(col => (
                      <span key={col.name} style={{
                        display: 'inline-flex', alignItems: 'center', gap: 4,
                        padding: '3px 10px', borderRadius: 4, fontSize: 12,
                        background: col.is_pk ? '#fef3c7' : '#f1f5f9',
                        color: col.is_pk ? '#92400e' : '#475569',
                        fontFamily: 'monospace',
                      }}>
                        {col.is_pk && <span style={{ fontWeight: 700 }}>PK</span>}
                        {col.name}
                        <span style={{ color: '#94a3b8', fontSize: 10 }}>{col.type}</span>
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* Data table */}
              {rows && (
                <div className="card" style={{ padding: 0, overflow: 'auto' }}>
                  <table style={{
                    width: '100%', borderCollapse: 'collapse', fontSize: 12,
                  }}>
                    <thead>
                      <tr style={{ background: '#f8fafc', borderBottom: '2px solid var(--border)' }}>
                        {rows.columns.map(col => (
                          <th key={col} style={{
                            padding: '8px 10px', textAlign: 'left', fontWeight: 600,
                            whiteSpace: 'nowrap', fontFamily: 'monospace', fontSize: 11,
                            color: 'var(--text-secondary)',
                          }}>
                            {col}
                          </th>
                        ))}
                        <th style={{ padding: '8px 10px', width: 80 }}>Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {rows.rows.map((row, i) => (
                        <tr key={row.id || i} style={{
                          borderBottom: '1px solid var(--border)',
                          background: i % 2 === 0 ? '#fff' : '#fafafa',
                        }}>
                          {rows.columns.map(col => (
                            <td key={col} style={{
                              padding: '6px 10px', maxWidth: 250,
                              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                            }}>
                              {formatCell(row[col])}
                            </td>
                          ))}
                          <td style={{ padding: '6px 10px', whiteSpace: 'nowrap' }}>
                            <button className="btn btn-sm btn-ghost" style={{ padding: '2px 6px' }}
                              onClick={() => openEdit(row)}
                              title="Edit">
                              <Edit3 size={13} />
                            </button>
                            <button className="btn btn-sm btn-ghost" style={{ padding: '2px 6px', color: 'var(--danger)' }}
                              onClick={() => setConfirmDelete({ id: row.id })}
                              title="Delete">
                              <Trash2 size={13} />
                            </button>
                          </td>
                        </tr>
                      ))}
                      {rows.rows.length === 0 && (
                        <tr>
                          <td colSpan={rows.columns.length + 1} style={{
                            padding: 24, textAlign: 'center', color: 'var(--text-muted)',
                          }}>
                            No rows
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>

                  {/* Pagination */}
                  {totalPages > 1 && (
                    <div style={{
                      display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8,
                      padding: '10px 12px', borderTop: '1px solid var(--border)',
                    }}>
                      <button className="btn btn-sm btn-ghost" disabled={page <= 1}
                        onClick={() => loadPage(page - 1)}>
                        Prev
                      </button>
                      <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                        Page {page} / {totalPages} ({rows.total} rows)
                      </span>
                      <button className="btn btn-sm btn-ghost" disabled={page >= totalPages}
                        onClick={() => loadPage(page + 1)}>
                        Next
                      </button>
                    </div>
                  )}
                </div>
              )}

              {/* Error */}
              {error && (
                <div className="card" style={{ background: '#fef2f2', borderColor: '#fecaca', color: '#991b1b', fontSize: 13 }}>
                  {error}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* CRUD Form Modal */}
      {showForm && (
        <div className="modal-overlay" onClick={() => setShowForm(false)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 520, maxHeight: '80vh', overflow: 'auto' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
              <h2 style={{ margin: 0 }}>
                {formMode === 'create' ? 'Insert Row' : 'Edit Row'}
                <span style={{ fontFamily: 'monospace', fontSize: 13, color: 'var(--text-muted)', marginLeft: 8 }}>
                  {activeTable}
                </span>
              </h2>
              <button className="btn btn-sm btn-ghost" onClick={() => setShowForm(false)}>
                <X size={16} />
              </button>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {schema?.columns.map(col => {
                if (col.is_pk && formMode === 'create') return null
                return (
                  <div key={col.name} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <label style={{
                      width: 130, fontSize: 12, fontFamily: 'monospace',
                      color: col.is_pk ? '#92400e' : 'var(--text-secondary)',
                      fontWeight: col.nullable ? 400 : 600,
                    }}>
                      {col.name}
                      {col.is_pk && <span style={{ fontSize: 10, marginLeft: 4 }}>PK</span>}
                      {!col.nullable && <span style={{ color: '#ef4444', marginLeft: 2 }}>*</span>}
                    </label>
                    {col.type === 'boolean' ? (
                      <select
                        value={formValues[col.name] ?? ''}
                        onChange={e => setFormValues({ ...formValues, [col.name]: e.target.value })}
                        disabled={col.is_pk}
                        style={{
                          flex: 1, padding: '6px 8px', border: '1px solid var(--border)',
                          borderRadius: 'var(--radius)', fontSize: 13,
                        }}
                      >
                        <option value="">NULL</option>
                        <option value="true">true</option>
                        <option value="false">false</option>
                      </select>
                    ) : (
                      <input
                        type="text"
                        value={formValues[col.name] ?? ''}
                        onChange={e => setFormValues({ ...formValues, [col.name]: e.target.value })}
                        disabled={col.is_pk}
                        placeholder={col.nullable ? 'NULL' : 'required'}
                        style={{
                          flex: 1, padding: '6px 8px', border: '1px solid var(--border)',
                          borderRadius: 'var(--radius)', fontSize: 13,
                          fontFamily: col.name === 'password_hash' ? 'monospace' : 'inherit',
                        }}
                      />
                    )}
                  </div>
                )
              })}
            </div>

            <div className="modal-actions" style={{ marginTop: 16 }}>
              <button className="btn btn-ghost" onClick={() => setShowForm(false)}>Cancel</button>
              <button className="btn btn-primary" onClick={handleSubmit} disabled={loading}>
                {loading ? '...' : formMode === 'create' ? 'Insert' : 'Update'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
