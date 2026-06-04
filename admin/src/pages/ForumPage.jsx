import { useState, useEffect, useCallback } from 'react'
import { Plus, Trash2, Edit3, ChevronDown, ChevronRight, MessageSquare } from 'lucide-react'
import { api } from '../api/client'

export default function ForumPage() {
  const [boards, setBoards] = useState([])
  const [toast, setToast] = useState(null)
  const [showBoardModal, setShowBoardModal] = useState(false)
  const [editingBoard, setEditingBoard] = useState(null)
  const [expandedBoard, setExpandedBoard] = useState(null)
  const [posts, setPosts] = useState([])
  const [postsPage, setPostsPage] = useState(1)
  const [postsTotal, setPostsTotal] = useState(0)
  const [loadingPosts, setLoadingPosts] = useState(false)

  const fetchBoards = useCallback(async () => {
    try {
      const data = await api.listBoards()
      setBoards(data.boards)
    } catch (e) {
      showToast(e.message, 'error')
    }
  }, [])

  useEffect(() => { fetchBoards() }, [fetchBoards])

  const showToast = (msg, type) => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  const fetchPosts = async (boardId, page = 1) => {
    setLoadingPosts(true)
    try {
      const data = await api.listPosts(boardId, page)
      setPosts(data.posts)
      setPostsTotal(data.total)
      setPostsPage(page)
    } catch (e) {
      showToast(e.message, 'error')
    } finally {
      setLoadingPosts(false)
    }
  }

  const toggleBoard = (boardId) => {
    if (expandedBoard === boardId) {
      setExpandedBoard(null)
      setPosts([])
    } else {
      setExpandedBoard(boardId)
      fetchPosts(boardId)
    }
  }

  const handleSaveBoard = async (e) => {
    e.preventDefault()
    const form = e.target
    const data = {
      name: form.name.value,
      slug: form.slug.value,
      description: form.description.value,
      sort_order: parseInt(form.sort_order.value) || 0,
    }
    try {
      if (editingBoard) {
        await api.updateBoard(editingBoard.id, data)
        showToast('Board updated', 'success')
      } else {
        await api.createBoard(data)
        showToast('Board created', 'success')
      }
      setShowBoardModal(false)
      setEditingBoard(null)
      fetchBoards()
    } catch (e) {
      showToast(e.message, 'error')
    }
  }

  const handleDeleteBoard = async (id) => {
    if (!confirm('Delete this board and all its posts?')) return
    try {
      await api.deleteBoard(id)
      showToast('Board deleted', 'success')
      if (expandedBoard === id) setExpandedBoard(null)
      fetchBoards()
    } catch (e) {
      showToast(e.message, 'error')
    }
  }

  const handleDeletePost = async (id) => {
    if (!confirm('Delete this post and its replies?')) return
    try {
      await api.deletePost(id)
      showToast('Post deleted', 'success')
      if (expandedBoard) fetchPosts(expandedBoard, postsPage)
    } catch (e) {
      showToast(e.message, 'error')
    }
  }

  const totalPostsPages = Math.max(1, Math.ceil(postsTotal / 20))

  return (
    <div>
      {toast && <div className={`toast toast-${toast.type}`}>{toast.msg}</div>}

      <div className="page-header">
        <h1>Forum</h1>
        <button className="btn btn-primary" onClick={() => { setEditingBoard(null); setShowBoardModal(true) }}>
          <Plus size={16} />
          New Board
        </button>
      </div>

      <div className="card" style={{ padding: 0, overflow: 'auto' }}>
        <table>
          <thead>
            <tr>
              <th style={{ width: 40 }}></th>
              <th>Name</th>
              <th>Slug</th>
              <th>Sort</th>
              <th style={{ width: 120 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {boards.length === 0 ? (
              <tr><td colSpan={5} style={{ textAlign: 'center', padding: 32, color: '#999' }}>No boards</td></tr>
            ) : boards.map((b) => (
              <>
                <tr key={b.id}>
                  <td>
                    <button
                      className="btn btn-sm btn-ghost"
                      onClick={() => toggleBoard(b.id)}
                      style={{ padding: 2 }}
                    >
                      {expandedBoard === b.id ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                    </button>
                  </td>
                  <td style={{ fontWeight: 500 }}>{b.name}</td>
                  <td style={{ color: 'var(--text-secondary)', fontSize: 13 }}>{b.slug}</td>
                  <td style={{ fontSize: 13 }}>{b.sort_order}</td>
                  <td>
                    <div style={{ display: 'flex', gap: 4 }}>
                      <button className="btn btn-sm btn-ghost" title="Edit"
                        onClick={() => { setEditingBoard(b); setShowBoardModal(true) }}>
                        <Edit3 size={14} />
                      </button>
                      <button className="btn btn-sm btn-danger" title="Delete"
                        onClick={() => handleDeleteBoard(b.id)}>
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </td>
                </tr>
                {expandedBoard === b.id && (
                  <tr key={`posts-${b.id}`}>
                    <td colSpan={5} style={{ padding: '0 0 16px 40px' }}>
                      <div style={{ padding: '8px 0' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                          <MessageSquare size={14} />
                          <span style={{ fontSize: 13, fontWeight: 600 }}>Posts ({postsTotal})</span>
                        </div>

                        {loadingPosts ? (
                          <div style={{ fontSize: 13, color: '#999' }}>Loading...</div>
                        ) : posts.length === 0 ? (
                          <div style={{ fontSize: 13, color: '#999' }}>No posts</div>
                        ) : (
                          <>
                            <table style={{ fontSize: 12 }}>
                              <thead>
                                <tr>
                                  <th>ID</th>
                                  <th>Title</th>
                                  <th>Author</th>
                                  <th>Created</th>
                                  <th style={{ width: 60 }}></th>
                                </tr>
                              </thead>
                              <tbody>
                                {posts.map((p) => (
                                  <tr key={p.id}>
                                    <td style={{ color: '#999' }}>{p.id}</td>
                                    <td>{p.title || p.content?.substring(0, 60)}</td>
                                    <td>{p.user?.username || `#${p.user_id}`}</td>
                                    <td style={{ color: '#999' }}>{new Date(p.created_at).toLocaleDateString()}</td>
                                    <td>
                                      <button className="btn btn-sm btn-danger" title="Delete"
                                        onClick={() => handleDeletePost(p.id)}>
                                        <Trash2 size={12} />
                                      </button>
                                    </td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                            {totalPostsPages > 1 && (
                              <div className="pagination" style={{ marginTop: 8 }}>
                                <button disabled={postsPage <= 1} onClick={() => fetchPosts(b.id, postsPage - 1)}>Prev</button>
                                <span style={{ fontSize: 12, color: '#999', padding: '0 8px' }}>
                                  {postsPage} / {totalPostsPages}
                                </span>
                                <button disabled={postsPage >= totalPostsPages} onClick={() => fetchPosts(b.id, postsPage + 1)}>Next</button>
                              </div>
                            )}
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                )}
              </>
            ))}
          </tbody>
        </table>
      </div>

      {showBoardModal && (
        <div className="modal-overlay" onClick={() => { setShowBoardModal(false); setEditingBoard(null) }}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2>{editingBoard ? 'Edit Board' : 'New Board'}</h2>
            <form onSubmit={handleSaveBoard}>
              <div className="form-group">
                <label>Name</label>
                <input name="name" defaultValue={editingBoard?.name || ''} required />
              </div>
              <div className="form-group">
                <label>Slug</label>
                <input name="slug" defaultValue={editingBoard?.slug || ''} required />
              </div>
              <div className="form-group">
                <label>Description</label>
                <input name="description" defaultValue={editingBoard?.description || ''} />
              </div>
              <div className="form-group">
                <label>Sort Order</label>
                <input name="sort_order" type="number" defaultValue={editingBoard?.sort_order ?? 0} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-ghost"
                  onClick={() => { setShowBoardModal(false); setEditingBoard(null) }}>Cancel</button>
                <button type="submit" className="btn btn-primary">
                  {editingBoard ? 'Save' : 'Create'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
