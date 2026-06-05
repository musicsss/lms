import { useState, useEffect, useCallback } from 'react'
import { api } from '../api/client'
import {
  Shield, AlertTriangle, Info, Bug, RefreshCw,
  ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight,
  ArrowUp, ArrowDown, Filter, X,
} from 'lucide-react'

const severityConfig = {
  critical: { label: '严重', color: '#dc2626', bg: '#fef2f2', icon: Shield },
  warning:  { label: '警告', color: '#d97706', bg: '#fffbeb', icon: AlertTriangle },
  info:     { label: '普通', color: '#2563eb', bg: '#eff6ff', icon: Info },
  debug:    { label: '调试', color: '#6b7280', bg: '#f9fafb', icon: Bug },
}

const allSeverities = ['critical', 'warning', 'info', 'debug']

const actionLabels = {
  login_success: '登录成功', login_failed: '登录失败', login_blocked: '登录被封禁',
  register: '用户注册', logout: '退出登录',
  file_upload: '文件上传', file_delete: '文件删除', file_download: '文件下载',
  video_comment: '视频评论', video_like: '视频点赞', video_collect: '视频收藏', video_play: '视频播放',
  danmaku_send: '发送弹幕', danmaku_delete: '删除弹幕',
  forum_post: '论坛发帖', forum_reply: '论坛回复', forum_like: '论坛点赞',
  profile_update: '资料更新', password_change: '密码修改', avatar_upload: '头像上传',
  admin_access: '管理操作', admin_delete_user: '删除用户', admin_delete_file: '删除文件', admin_config: '配置修改',
}

const columns = [
  { key: 'created_at', label: '时间', sortable: true, width: 160 },
  { key: 'username', label: '用户', sortable: false, width: 100 },
  { key: 'action', label: '操作', sortable: true, width: 110, filterable: true },
  { key: 'severity', label: '等级', sortable: true, width: 80, filterable: true },
  { key: 'resource', label: '资源', sortable: false, width: 140 },
  { key: 'detail', label: '详情', sortable: false },
  { key: 'ip', label: 'IP', sortable: true, width: 130 },
  { key: 'success', label: '结果', sortable: true, width: 70 },
]

const PAGE_SIZE = 20

export default function AuditPage() {
  const [logs, setLogs] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [sort, setSort] = useState('created_at')
  const [order, setOrder] = useState('desc')
  const [filters, setFilters] = useState({})
  const [filterOpen, setFilterOpen] = useState(null)

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const fetchLogs = useCallback(async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams({ page, page_size: PAGE_SIZE, sort, order })
      if (filters.severity) params.set('severity', filters.severity)
      if (filters.action) params.set('action', filters.action)
      const data = await api.get('/admin/audit?' + params)
      setLogs(data.logs || [])
      setTotal(data.total || 0)
    } catch (e) {
      console.error('Failed to load audit logs:', e)
    } finally {
      setLoading(false)
    }
  }, [page, sort, order, filters])

  useEffect(() => { fetchLogs() }, [fetchLogs])

  const handleSort = (colKey) => {
    if (sort === colKey) {
      setOrder(o => o === 'asc' ? 'desc' : 'asc')
    } else {
      setSort(colKey)
      setOrder('desc')
    }
    setPage(1)
  }

  const toggleFilter = (colKey) => {
    setFilterOpen(f => f === colKey ? null : colKey)
  }

  const setFilter = (key, val) => {
    setFilters(f => {
      const next = { ...f }
      if (val) next[key] = val; else delete next[key]
      return next
    })
    setPage(1)
  }

  const clearFilters = () => {
    setFilters({})
    setFilterOpen(null)
    setPage(1)
  }

  const formatTime = (t) => {
    if (!t) return '-'
    const d = new Date(t)
    return d.toLocaleString('zh-CN', { month:'2-digit', day:'2-digit', hour:'2-digit', minute:'2-digit', second:'2-digit' })
  }

  const hasFilters = Object.keys(filters).length > 0

  const renderFilterDropdown = (col) => {
    if (filterOpen !== col.key) return null

    if (col.key === 'severity') {
      return (
        <div style={filterDropdownStyle}>
          {allSeverities.map(s => {
            const sc = severityConfig[s] || severityConfig.info
            return (
              <button key={s} onClick={() => {setFilter('severity', s); setFilterOpen(null)}}
                style={{...filterItemStyle, background: filters.severity === s ? sc.bg : 'transparent'}}>
                <sc.icon size={12} color={sc.color} />
                <span>{sc.label}</span>
                {filters.severity === s && <X size={12} onClick={e => {e.stopPropagation(); setFilter('severity', '')}} />}
              </button>
            )
          })}
          {filters.severity && (
            <button onClick={() => {setFilter('severity', ''); setFilterOpen(null)}} style={clearFilterItemStyle}>
              <X size={12} /> 清除筛选
            </button>
          )}
        </div>
      )
    }

    if (col.key === 'action') {
      const actions = Object.entries(actionLabels)
      return (
        <div style={{...filterDropdownStyle, maxHeight: 300, overflowY: 'auto'}}>
          {actions.map(([value, label]) => (
            <button key={value} onClick={() => {setFilter('action', value); setFilterOpen(null)}}
              style={{...filterItemStyle, background: filters.action === value ? '#f1f5f9' : 'transparent'}}>
              <span>{label}</span>
              {filters.action === value && <X size={12} onClick={e => {e.stopPropagation(); setFilter('action', '')}} />}
            </button>
          ))}
          {filters.action && (
            <button onClick={() => {setFilter('action', ''); setFilterOpen(null)}} style={clearFilterItemStyle}>
              <X size={12} /> 清除筛选
            </button>
          )}
        </div>
      )
    }

    return null
  }

  // ----- pagination range -----
  const getPageRange = () => {
    const range = []
    const start = Math.max(1, page - 3)
    const end = Math.min(totalPages, page + 3)
    for (let i = start; i <= end; i++) range.push(i)
    return range
  }

  const SortIcon = ({ colKey }) => {
    if (sort !== colKey) return null
    return order === 'asc'
      ? <ArrowUp size={11} style={{ marginLeft: 4 }} />
      : <ArrowDown size={11} style={{ marginLeft: 4 }} />
  }

  return (
    <div>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 12 }}>
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700, color: '#1e293b' }}>{'审计日志'}</h1>
          <span style={{ fontSize: 13, color: '#94a3b8' }}>{'共'} {total} {'条记录'}</span>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          {hasFilters && (
            <button onClick={clearFilters} style={toolBtnStyle}>
              <X size={14} /> {'清除筛选'}
            </button>
          )}
          <button onClick={fetchLogs} style={toolBtnStyle}>
            <RefreshCw size={14} /> {'刷新'}
          </button>
        </div>
      </div>

      {/* Table */}
      <div style={{ background: '#fff', border: '1px solid #e2e8f0', borderRadius: 8, overflow: 'hidden' }}>
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13, tableLayout: 'fixed' }}>
            <colgroup>
              {columns.map(col => (
                <col key={col.key} style={col.width ? { width: col.width } : {}} />
              ))}
            </colgroup>
            <thead>
              <tr style={{ background: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                {columns.map(col => (
                  <th key={col.key} style={{
                    padding: '10px 12px', textAlign: 'left', fontWeight: 600,
                    color: '#475569', fontSize: 12, whiteSpace: 'nowrap',
                    cursor: col.sortable ? 'pointer' : 'default',
                    userSelect: 'none', position: 'relative',
                  }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}
                      onClick={() => col.sortable && handleSort(col.key)}>
                      <span>{col.label}</span>
                      <SortIcon colKey={col.key} />
                    </div>
                    {col.filterable && (
                      <button
                        onClick={(e) => { e.stopPropagation(); toggleFilter(col.key) }}
                        style={{
                          position: 'absolute', right: 4, top: '50%', transform: 'translateY(-50%)',
                          background: 'none', border: 'none', cursor: 'pointer',
                          padding: 2, borderRadius: 3, color: filters[col.key] ? '#2563eb' : '#94a3b8',
                        }}
                        title={'筛选'}
                      >
                        <Filter size={12} />
                      </button>
                    )}
                    {renderFilterDropdown(col)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={columns.length} style={{ textAlign: 'center', padding: 60, color: '#94a3b8' }}>
                    {'加载中...'}
                  </td>
                </tr>
              ) : logs.length === 0 ? (
                <tr>
                  <td colSpan={columns.length} style={{ textAlign: 'center', padding: 60, color: '#94a3b8' }}>
                    {'暂无审计记录'}
                  </td>
                </tr>
              ) : (
                logs.map(log => {
                  const sc = severityConfig[log.severity] || severityConfig.info
                  const Icon = sc.icon
                  return (
                    <tr key={log.id}
                      style={{ borderBottom: '1px solid #f1f5f9' }}
                      onMouseEnter={e => e.currentTarget.style.background = '#fafbfc'}
                      onMouseLeave={e => e.currentTarget.style.background = 'transparent'}>
                      <td style={tdStyle}>{formatTime(log.created_at)}</td>
                      <td style={tdStyle}>
                        <span style={{ fontWeight: 500 }}>
                          {log.username || (log.user_id ? '#' + log.user_id : '匿名')}
                        </span>
                      </td>
                      <td style={tdStyle}>{actionLabels[log.action] || log.action}</td>
                      <td style={tdStyle}>
                        <span style={{
                          display: 'inline-flex', alignItems: 'center', gap: 4,
                          padding: '2px 8px', borderRadius: 4,
                          background: sc.bg, color: sc.color, fontWeight: 600, fontSize: 11,
                        }}>
                          <Icon size={11} /> {sc.label}
                        </span>
                      </td>
                      <td style={tdStyle}>
                        {log.resource
                          ? <span style={{ fontSize: 12 }}>{log.resource}{log.resource_id ? ' #' + log.resource_id : ''}</span>
                          : <span style={{ color: '#cbd5e1' }}>-</span>}
                      </td>
                      <td style={{ ...tdStyle, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {log.detail || <span style={{ color: '#cbd5e1' }}>-</span>}
                      </td>
                      <td style={tdStyle}>
                        <code style={{ fontSize: 11, color: '#64748b' }}>{log.ip || '-'}</code>
                      </td>
                      <td style={tdStyle}>
                        <span style={{
                          display: 'inline-flex', alignItems: 'center', gap: 3,
                          color: log.success ? '#16a34a' : '#dc2626',
                          fontWeight: 600, fontSize: 12,
                        }}>
                          <span style={{
                            width: 6, height: 6, borderRadius: '50%',
                            background: log.success ? '#16a34a' : '#dc2626',
                            display: 'inline-block',
                          }} />
                          {log.success ? '成功' : '失败'}
                        </span>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination bar */}
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '12px 16px', borderTop: '1px solid #e2e8f0', background: '#fafbfc',
        }}>
          <span style={{ fontSize: 12, color: '#94a3b8' }}>
            {'第'} {(page-1)*PAGE_SIZE + 1}-{Math.min(page*PAGE_SIZE, total)} {'条，共'} {total} {'条'}
          </span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <button disabled={page <= 1} onClick={() => setPage(1)} style={pageBtnStyle(page <= 1)}>
              <ChevronsLeft size={14} />
            </button>
            <button disabled={page <= 1} onClick={() => setPage(p => p - 1)} style={pageBtnStyle(page <= 1)}>
              <ChevronLeft size={14} />
            </button>

            {getPageRange().map(i => (
              <button key={i} onClick={() => setPage(i)}
                style={{
                  minWidth: 32, height: 32, padding: '0 6px', borderRadius: 6,
                  border: i === page ? '1px solid #2563eb' : '1px solid transparent',
                  background: i === page ? '#eff6ff' : 'transparent',
                  color: i === page ? '#2563eb' : '#475569',
                  fontWeight: i === page ? 600 : 400,
                  cursor: 'pointer', fontSize: 13,
                }}>
                {i}
              </button>
            ))}

            <button disabled={page >= totalPages} onClick={() => setPage(p => p + 1)} style={pageBtnStyle(page >= totalPages)}>
              <ChevronRight size={14} />
            </button>
            <button disabled={page >= totalPages} onClick={() => setPage(totalPages)} style={pageBtnStyle(page >= totalPages)}>
              <ChevronsRight size={14} />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

const thStyle = {
  padding: '12px 12px', textAlign: 'left', fontWeight: 600,
  color: '#475569', fontSize: 12, textTransform: 'uppercase', whiteSpace: 'nowrap',
}
const tdStyle = { padding: '10px 12px', color: '#334155' }

const toolBtnStyle = {
  display: 'inline-flex', alignItems: 'center', gap: 6,
  padding: '7px 14px', background: '#fff', border: '1px solid #e2e8f0',
  borderRadius: 6, cursor: 'pointer', fontSize: 13, color: '#475569',
}

const pageBtnStyle = (disabled) => ({
  display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
  width: 32, height: 32, borderRadius: 6,
  border: '1px solid #e2e8f0', background: disabled ? '#f8fafc' : '#fff',
  color: disabled ? '#cbd5e1' : '#475569',
  cursor: disabled ? 'default' : 'pointer',
})

const filterDropdownStyle = {
  position: 'absolute', top: '100%', left: 0, zIndex: 50,
  background: '#fff', border: '1px solid #e2e8f0', borderRadius: 8,
  boxShadow: '0 4px 12px rgba(0,0,0,0.1)', minWidth: 160,
  padding: '4px 0', marginTop: 4,
}

const filterItemStyle = {
  display: 'flex', alignItems: 'center', gap: 8,
  width: '100%', padding: '7px 14px', border: 'none',
  background: 'transparent', cursor: 'pointer',
  fontSize: 13, textAlign: 'left',
}

const clearFilterItemStyle = {
  display: 'flex', alignItems: 'center', gap: 8,
  width: '100%', padding: '7px 14px', border: 'none',
  borderTop: '1px solid #f1f5f9',
  background: 'transparent', cursor: 'pointer',
  fontSize: 12, color: '#94a3b8', textAlign: 'left',
}
