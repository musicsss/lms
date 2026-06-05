import React, { useState, useEffect, useCallback } from 'react'
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

  const [showForm, setShowForm] = useState(false)
  const [formMode, setFormMode] = useState('create')
  const [formValues, setFormValues] = useState({})
  const [editingId, setEditingId] = useState(null)
  const [confirmDelete, setConfirmDelete] = useState(null)
  const [message, setMessage] = useState(null)

  const fetchTables = useCallback(async () => {
    try {
      const data = await api.get('/admin/db/tables')
      setTables(data.tables || [])
    } catch (e) {
      setError(e.message)
    }
  }, [])

  useEffect(() => { fetchTables() }, [fetchTables])

  const loadTable = async (name) => {
    setActiveTable(name)
    setLoading(true)
    setError(null)
    setMessage(null)
    setSchema(null)
    setRows(null)
    try {
      const [schemaData, rowsData] = await Promise.all([
        api.get('/admin/db/tables/' + encodeURIComponent(name)),
        api.get('/admin/db/tables/' + encodeURIComponent(name) + '/rows?page=1&page_size=20'),
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

  const loadPage = async (p) => {
    if (!activeTable) return
    setLoading(true)
    try {
      const data = await api.get('/admin/db/tables/' + encodeURIComponent(activeTable) + '/rows?page=' + p + '&page_size=20')
      setRows(data)
      setPage(p)
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  const openCreate = () => {
    const defaults = {}
    if (schema && schema.columns) {
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

  const openEdit = (row) => {
    setFormValues({ ...row })
    setFormMode('edit')
    setEditingId(row.id)
    setShowForm(true)
    setMessage(null)
  }

  const handleSubmit = async () => {
    setLoading(true)
    setMessage(null)
    try {
      const cleaned = {}
      for (const [k, v] of Object.entries(formValues)) {
        if (k === 'id') continue
        if (v === '') { cleaned[k] = null }
        else if (v === 'true') { cleaned[k] = true }
        else if (v === 'false') { cleaned[k] = false }
        else if (!isNaN(v) && v !== '' && schema?.columns?.find(c => c.name === k)?.type !== 'text') { cleaned[k] = Number(v) }
        else { cleaned[k] = v }
      }

      if (formMode === 'create') {
        await api.post('/admin/db/tables/' + encodeURIComponent(activeTable), cleaned)
      } else {
        await api.put('/admin/db/tables/' + encodeURIComponent(activeTable) + '/' + editingId, cleaned)
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

  const handleDelete = async () => {
    if (!confirmDelete) return
    setLoading(true)
    try {
      await api.del('/admin/db/tables/' + encodeURIComponent(activeTable) + '/' + confirmDelete.id)
      setMessage({ ok: true, text: 'Row ID=' + confirmDelete.id + ' deleted' })
      setConfirmDelete(null)
      loadTable(activeTable)
    } catch (e) {
      setMessage({ ok: false, text: e.message })
      setConfirmDelete(null)
    } finally {
      setLoading(false)
    }
  }

  const formatCell = (val) => {
    if (val === null || val === undefined) return React.createElement('span', { style: { color: '#94a3b8', fontStyle: 'italic' } }, 'NULL')
    if (typeof val === 'boolean') return val ? 'true' : 'false'
    if (typeof val === 'object') return JSON.stringify(val).slice(0, 80)
    const s = String(val)
    if (s.length > 200) return s.slice(0, 200) + '...'
    return s
  }

  const totalPages = rows && rows.page_size > 0 ? Math.ceil((rows.total || 0) / rows.page_size) : 0
  const colList = (rows && rows.columns) || []

  return React.createElement('div', null,
    React.createElement('div', { style: { marginBottom: 20 } },
      React.createElement('h1', { style: { fontSize: 22, fontWeight: 700, margin: 0, display: 'flex', alignItems: 'center', gap: 10 } },
        React.createElement(Database, { size: 22 }),
        ' Database Browser'
      )
    ),

    confirmDelete && React.createElement('div', {
      style: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 },
      onClick: () => setConfirmDelete(null)
    },
      React.createElement('div', {
        style: { background: '#fff', borderRadius: 12, padding: 24, minWidth: 400, maxWidth: 520, boxShadow: '0 20px 60px rgba(0,0,0,0.2)' },
        onClick: e => e.stopPropagation()
      },
        React.createElement('div', { style: { display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 } },
          React.createElement(AlertTriangle, { size: 20, color: '#d97706' }),
          React.createElement('h2', { style: { margin: 0, fontSize: 16 } }, 'Confirm Delete')
        ),
        React.createElement('p', { style: { fontSize: 14, color: '#6b7280', marginBottom: 16 } },
          'Delete row ID=', React.createElement('strong', null, String(confirmDelete.id)),
          ' from ', React.createElement('code', null, activeTable), '?'
        ),
        React.createElement('div', { style: { display: 'flex', gap: 8, justifyContent: 'flex-end' } },
          React.createElement('button', { className: 'btn btn-ghost', onClick: () => setConfirmDelete(null) }, 'Cancel'),
          React.createElement('button', { className: 'btn btn-danger', onClick: handleDelete, disabled: loading },
            loading ? '...' : 'Delete'
          )
        )
      )
    ),

    React.createElement('div', { style: { display: 'flex', gap: 24, alignItems: 'flex-start' } },
      React.createElement('div', { style: { width: 240, flexShrink: 0, display: 'flex', flexDirection: 'column', gap: 4 } },
        React.createElement('div', { style: { fontSize: 13, fontWeight: 600, color: '#6b7280', padding: '4px 12px', marginBottom: 4 } },
          'Tables (' + tables.length + ')'
        ),
        tables.map(t =>
          React.createElement('button', {
            key: t.name,
            onClick: () => loadTable(t.name),
            style: {
              display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px',
              borderRadius: 6, border: 'none', cursor: 'pointer', width: '100%',
              textAlign: 'left', fontSize: 13, background: activeTable === t.name ? '#f3f4f6' : 'transparent',
              color: '#18191c'
            }
          },
            React.createElement(Table, { size: 14 }),
            React.createElement('span', { style: { flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } }, t.name),
            React.createElement('span', { style: { fontSize: 11, color: '#9ca3af', flexShrink: 0 } }, String(t.row_count))
          )
        )
      ),

      React.createElement('div', { style: { flex: 1, minWidth: 0 } },
        !activeTable && !loading && React.createElement('div', { style: { textAlign: 'center', padding: 60, color: '#9ca3af', fontSize: 14 } },
          'Please select a table from the left'
        ),

        activeTable && loading && React.createElement('div', { style: { textAlign: 'center', padding: 40, color: '#9ca3af', fontSize: 14 } },
          'Loading...'
        ),

        activeTable && !loading && error && React.createElement('div', {
          style: { background: '#fef2f2', border: '1px solid #fecaca', borderRadius: 8, padding: '12px 16px', color: '#991b1b', fontSize: 13, marginBottom: 16 }
        }, error),

        activeTable && !loading && !error && rows && React.createElement(React.Fragment, null,
          React.createElement('div', { style: { display: 'flex', gap: 8, marginBottom: 16, alignItems: 'center', flexWrap: 'wrap' } },
            React.createElement('span', { style: { fontSize: 14, fontWeight: 600, flex: 1 } },
              activeTable,
              React.createElement('span', { style: { fontSize: 12, fontWeight: 400, color: '#9ca3af', marginLeft: 8 } },
                String(rows.total) + ' rows'
              )
            ),
            React.createElement('button', { className: 'btn btn-sm btn-primary', onClick: openCreate },
              React.createElement(Plus, { size: 14 }),
              ' Insert'
            )
          ),

          message && React.createElement('div', {
            style: {
              padding: '8px 12px', borderRadius: 6, marginBottom: 12, fontSize: 13,
              background: message.ok ? '#ecfdf5' : '#fef2f2',
              color: message.ok ? '#065f46' : '#991b1b'
            }
          }, message.text),

          colList.length === 0
            ? React.createElement('div', { style: { textAlign: 'center', padding: 40, color: '#9ca3af', fontSize: 14 } },
                'This table has no data'
              )
            : React.createElement(React.Fragment, null,
                React.createElement('div', { style: { overflowX: 'auto', border: '1px solid #e5e7eb', borderRadius: 8 } },
                  React.createElement('table', { style: { width: '100%', borderCollapse: 'collapse', fontSize: 13 } },
                    React.createElement('thead', null,
                      React.createElement('tr', null,
                        colList.map(col =>
                          React.createElement('th', {
                            key: col,
                            style: { textAlign: 'left', padding: '8px 10px', background: '#f9fafb', borderBottom: '2px solid #e5e7eb', fontWeight: 600, color: '#374151', whiteSpace: 'nowrap', fontFamily: 'monospace' }
                          }, col)
                        ),
                        React.createElement('th', { style: { textAlign: 'left', padding: '8px 10px', background: '#f9fafb', borderBottom: '2px solid #e5e7eb', width: 80 } }, 'Actions')
                      )
                    ),
                    React.createElement('tbody', null,
                      (!rows.rows || rows.rows.length === 0)
                        ? React.createElement('tr', null,
                            React.createElement('td', { colSpan: colList.length + 1, style: { textAlign: 'center', padding: 30, color: '#9ca3af' } },
                              'This table has no data'
                            )
                          )
                        : rows.rows.map((row, idx) =>
                            React.createElement('tr', { key: row.id ?? idx },
                              colList.map(col =>
                                React.createElement('td', {
                                  key: col,
                                  style: { padding: '6px 10px', borderBottom: '1px solid #f3f4f6', maxWidth: 250, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' },
                                  title: String(row[col] ?? '')
                                }, formatCell(row[col]))
                              ),
                              React.createElement('td', { style: { padding: '6px 10px', borderBottom: '1px solid #f3f4f6' } },
                                React.createElement('div', { style: { display: 'flex', gap: 2 } },
                                  React.createElement('button', {
                                    style: { background: 'none', border: 'none', cursor: 'pointer', padding: '4px 6px', borderRadius: 4, color: '#6b7280' },
                                    title: 'Edit', onClick: () => openEdit(row)
                                  }, React.createElement(Edit3, { size: 14 })),
                                  React.createElement('button', {
                                    style: { background: 'none', border: 'none', cursor: 'pointer', padding: '4px 6px', borderRadius: 4, color: '#ef4444' },
                                    title: 'Delete', onClick: () => setConfirmDelete({ id: row.id })
                                  }, React.createElement(Trash2, { size: 14 }))
                                )
                              )
                            )
                          )
                    )
                  )
                ),

                totalPages > 1 && React.createElement('div', {
                  style: { display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, padding: '10px 12px', borderTop: '1px solid #e5e7eb' }
                },
                  React.createElement('button', {
                    style: { padding: '4px 12px', border: '1px solid #d1d5db', borderRadius: 4, background: page <= 1 ? '#f3f4f6' : '#fff', color: page <= 1 ? '#9ca3af' : '#374151', cursor: page <= 1 ? 'not-allowed' : 'pointer', fontSize: 12 },
                    disabled: page <= 1, onClick: () => loadPage(page - 1)
                  }, 'Prev'),
                  React.createElement('span', { style: { fontSize: 12, color: '#6b7280' } }, String(page) + ' / ' + String(totalPages)),
                  React.createElement('button', {
                    style: { padding: '4px 12px', border: '1px solid #d1d5db', borderRadius: 4, background: page >= totalPages ? '#f3f4f6' : '#fff', color: page >= totalPages ? '#9ca3af' : '#374151', cursor: page >= totalPages ? 'not-allowed' : 'pointer', fontSize: 12 },
                    disabled: page >= totalPages, onClick: () => loadPage(page + 1)
                  }, 'Next')
                )
              )
        ),

        activeTable && !loading && !error && !rows && React.createElement('div', { style: { textAlign: 'center', padding: 40, color: '#9ca3af', fontSize: 14 } },
          'Loading...'
        )
      )
    ),

    showForm && React.createElement('div', {
      style: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 },
      onClick: () => setShowForm(false)
    },
      React.createElement('div', {
        style: { background: '#fff', borderRadius: 12, padding: 24, minWidth: 400, maxWidth: 520, maxHeight: '80vh', overflow: 'auto', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' },
        onClick: e => e.stopPropagation()
      },
        React.createElement('div', { style: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 } },
          React.createElement('h2', { style: { margin: 0, fontSize: 16 } },
            formMode === 'create' ? 'Insert Row' : 'Edit Row',
            React.createElement('span', { style: { fontFamily: 'monospace', fontSize: 13, color: '#9ca3af', marginLeft: 8 } }, activeTable)
          ),
          React.createElement('button', { className: 'btn btn-sm btn-ghost', onClick: () => setShowForm(false) },
            React.createElement(X, { size: 16 })
          )
        ),

        React.createElement('div', { style: { display: 'flex', flexDirection: 'column', gap: 10 } },
          schema?.columns?.map(col => {
            if (col.is_pk && formMode === 'create') return null
            return React.createElement('div', { key: col.name, style: { display: 'flex', alignItems: 'center', gap: 10 } },
              React.createElement('label', {
                style: { width: 130, fontSize: 12, fontFamily: 'monospace', color: col.is_pk ? '#92400e' : '#6b7280', fontWeight: col.nullable ? 400 : 600 }
              },
                col.name,
                col.is_pk && React.createElement('span', { style: { fontSize: 10, marginLeft: 4 }, color: '#92400e' }, 'PK'),
                !col.nullable && React.createElement('span', { style: { color: '#ef4444', marginLeft: 2 } }, '*')
              ),
              col.type === 'boolean'
                ? React.createElement('select', {
                    value: formValues[col.name] ?? '',
                    onChange: e => setFormValues({ ...formValues, [col.name]: e.target.value }),
                    disabled: col.is_pk,
                    style: { flex: 1, padding: '6px 8px', border: '1px solid #d1d5db', borderRadius: 6, fontSize: 13 }
                  },
                    React.createElement('option', { value: '' }, 'NULL'),
                    React.createElement('option', { value: 'true' }, 'true'),
                    React.createElement('option', { value: 'false' }, 'false')
                  )
                : React.createElement('input', {
                    type: 'text',
                    value: formValues[col.name] ?? '',
                    onChange: e => setFormValues({ ...formValues, [col.name]: e.target.value }),
                    disabled: col.is_pk,
                    placeholder: col.nullable ? 'NULL' : 'required',
                    style: { flex: 1, padding: '6px 8px', border: '1px solid #d1d5db', borderRadius: 6, fontSize: 13, fontFamily: col.name === 'password_hash' ? 'monospace' : 'inherit' }
                  })
            )
          })
        ),

        React.createElement('div', { style: { display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 16 } },
          React.createElement('button', { className: 'btn btn-ghost', onClick: () => setShowForm(false) }, 'Cancel'),
          React.createElement('button', { className: 'btn btn-primary', onClick: handleSubmit, disabled: loading },
            loading ? '...' : (formMode === 'create' ? 'Insert' : 'Update')
          )
        )
      )
    )
  )
}