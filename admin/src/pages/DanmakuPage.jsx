import { useState, useEffect } from 'react';
import { MessageSquare, Check, X, Trash2, AlertTriangle } from 'lucide-react';
import { api } from '../api/client';

export default function DanmakuPage() {
  const [danmaku, setDanmaku] = useState([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const pageSize = 20;

  const loadDanmaku = async (p) => {
    setLoading(true);
    setError('');
    try {
      const data = await api.get(`/admin/danmaku?page=${p}&page_size=${pageSize}`);
      setDanmaku(data.danmaku || []);
      setTotal(data.total || 0);
    } catch (err) {
      setError(err.message || 'Failed to load danmaku');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadDanmaku(page); }, [page]);

  const totalPages = Math.ceil(total / pageSize);

  const handleApprove = async (id) => {
    try {
      await api.post(`/admin/danmaku/${id}/approve`);
      loadDanmaku(page);
    } catch (err) {
      setError(err.message || 'Failed to approve');
    }
  };

  const handleReject = async (id) => {
    try {
      await api.post(`/admin/danmaku/${id}/reject`);
      loadDanmaku(page);
    } catch (err) {
      setError(err.message || 'Failed to reject');
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('确定删除此弹幕？')) return;
    try {
      await api.del(`/admin/danmaku/${id}`);
      loadDanmaku(page);
    } catch (err) {
      setError(err.message || 'Failed to delete');
    }
  };

  const statusBadge = (status) => {
    const colors = {
      pending: { bg: '#fff8e1', color: '#f57c00' },
      approved: { bg: '#e8f5e9', color: '#2e7d32' },
      rejected: { bg: '#ffebee', color: '#c62828' },
    };
    const s = colors[status] || colors.pending;
    return { background: s.bg, color: s.color, padding: '2px 8px', borderRadius: 4, fontSize: 12, fontWeight: 600 };
  };

  const style = {
    container: { maxWidth: 1200, margin: '0 auto' },
    header: { fontSize: 22, fontWeight: 700, marginBottom: 24, display: 'flex', alignItems: 'center', gap: 10 },
    error: { background: '#ffebee', color: '#c62828', padding: '10px 16px', borderRadius: 6, marginBottom: 16, fontSize: 14 },
    table: { width: '100%', borderCollapse: 'collapse', background: '#fff', borderRadius: 8, overflow: 'hidden', boxShadow: '0 1px 3px rgba(0,0,0,0.08)' },
    th: { textAlign: 'left', padding: '12px 14px', background: '#f8f9fa', borderBottom: '2px solid #e9ecef', fontSize: 13, fontWeight: 700, color: '#495057' },
    td: { padding: '12px 14px', borderBottom: '1px solid #f0f0f0', fontSize: 13, verticalAlign: 'middle' },
    actionBtn: { background: 'none', border: 'none', cursor: 'pointer', padding: '6px 8px', borderRadius: 4, transition: 'background 0.1s' },
    pagination: { display: 'flex', justifyContent: 'center', gap: 8, marginTop: 24, alignItems: 'center' },
    pageBtn: (active) => ({
      padding: '6px 14px', border: '1px solid #dee2e6', borderRadius: 4,
      background: active ? '#fb7299' : '#fff', color: active ? '#fff' : '#495057',
      cursor: 'pointer', fontSize: 13,
    }),
    empty: { textAlign: 'center', padding: 60, color: '#9499a0', fontSize: 14 },
  };

  return (
    <div style={style.container}>
      <div style={style.header}>
        <MessageSquare size={22} />
        弹幕审核
      </div>

      {error && <div style={style.error}>{error}</div>}

      {loading ? (
        <div style={style.empty}>加载中...</div>
      ) : danmaku.length === 0 ? (
        <div style={style.empty}>暂无弹幕</div>
      ) : (
        <>
          <table style={style.table}>
            <thead>
              <tr>
                <th style={style.th}>内容</th>
                <th style={style.th}>用户</th>
                <th style={style.th}>视频ID</th>
                <th style={style.th}>时间点</th>
                <th style={style.th}>颜色</th>
                <th style={style.th}>状态</th>
                <th style={style.th}>发送时间</th>
                <th style={style.th}>操作</th>
              </tr>
            </thead>
            <tbody>
              {danmaku.map(d => (
                <tr key={d.id}>
                  <td style={style.td}><span style={{ color: d.color || '#fff', background: d.color && d.color !== '#ffffff' ? 'transparent' : '#333', padding: '2px 6px', borderRadius: 3, maxWidth: 250, display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{d.content}</span></td>
                  <td style={style.td}>{d.user?.username || '用户'}</td>
                  <td style={style.td}>{d.video_id}</td>
                  <td style={style.td}>{d.time_sec?.toFixed(1)}s</td>
                  <td style={style.td}><span style={{ display: 'inline-block', width: 20, height: 20, borderRadius: '50%', background: d.color || '#fff', border: '1px solid #ddd', verticalAlign: 'middle' }} /></td>
                  <td style={style.td}><span style={statusBadge(d.status)}>{d.status}</span></td>
                  <td style={style.td}>{d.created_at ? new Date(d.created_at).toLocaleString() : '-'}</td>
                  <td style={style.td}>
                    <div style={{ display: 'flex', gap: 4 }}>
                      {d.status !== 'approved' && (
                        <button style={{ ...style.actionBtn, color: '#2e7d32' }} title="通过" onClick={() => handleApprove(d.id)}>
                          <Check size={16} />
                        </button>
                      )}
                      {d.status !== 'rejected' && (
                        <button style={{ ...style.actionBtn, color: '#c62828' }} title="拒绝" onClick={() => handleReject(d.id)}>
                          <X size={16} />
                        </button>
                      )}
                      <button style={{ ...style.actionBtn, color: '#d32f2f' }} title="删除" onClick={() => handleDelete(d.id)}>
                        <Trash2 size={16} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          {totalPages > 1 && (
            <div style={style.pagination}>
              <button style={style.pageBtn(false)} disabled={page <= 1} onClick={() => setPage(p => Math.max(1, p - 1))}>上一页</button>
              <span style={{ fontSize: 13, color: '#666' }}>{page} / {totalPages}</span>
              <button style={style.pageBtn(false)} disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>下一页</button>
            </div>
          )}
        </>
      )}
    </div>
  );
}