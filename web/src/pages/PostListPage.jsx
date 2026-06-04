import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ArrowLeft, Plus, MessageSquare, Eye } from 'lucide-react';
import { api } from '../api/client';
import './PostListPage.css';

export default function PostListPage() {
  const { boardId } = useParams();
  const [posts, setPosts] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [showNewPost, setShowNewPost] = useState(false);
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');

  const loadPosts = async (p) => {
    setLoading(true);
    try {
      const data = await api.listPosts(boardId, p);
      setPosts(data.posts || []);
      setTotal(data.total || 0);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadPosts(page); }, [boardId, page]);

  const handleCreate = async (e) => {
    e.preventDefault();
    if (!title.trim() || !content.trim()) return;
    try {
      await api.createPost(boardId, title, content);
      setTitle('');
      setContent('');
      setShowNewPost(false);
      loadPosts(1);
      setPage(1);
    } catch (err) {
      alert(err.message);
    }
  };

  const totalPages = Math.ceil(total / 20);

  return (
    <div className="post-list-page">
      <div className="post-list-toolbar">
        <Link to="/forum" className="btn-sm">
          <ArrowLeft size={16} />
          返回板块
        </Link>
        <button className="btn-sm" onClick={() => setShowNewPost(!showNewPost)}>
          <Plus size={16} />
          发帖
        </button>
      </div>

      {showNewPost && (
        <form className="new-post-form" onSubmit={handleCreate}>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="帖子标题"
            required
          />
          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            placeholder="帖子内容..."
            rows={5}
            required
          />
          <div className="form-actions">
            <button type="submit" className="btn-sm">发布</button>
            <button type="button" className="btn-sm" onClick={() => setShowNewPost(false)}>取消</button>
          </div>
        </form>
      )}

      {loading ? (
        <div className="post-list-loading">加载中...</div>
      ) : posts.length === 0 ? (
        <div className="post-list-empty">暂无帖子</div>
      ) : (
        <>
          <div className="post-items">
            {posts.map((post) => (
              <Link to={`/forum/${boardId}/${post.id}`} key={post.id} className="post-item">
                <div className="post-item-title">{post.title}</div>
                <div className="post-item-meta">
                  <span>{post.user?.username || '匿名'}</span>
                  <span className="meta-sep">·</span>
                  <span><MessageSquare size={12} /> {post.like_count || 0}</span>
                  <span className="meta-sep">·</span>
                  <span>{formatDate(post.created_at)}</span>
                </div>
              </Link>
            ))}
          </div>
          {totalPages > 1 && (
            <div className="pagination">
              {Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
                <button
                  key={p}
                  className={`page-btn ${p === page ? 'active' : ''}`}
                  onClick={() => setPage(p)}
                >
                  {p}
                </button>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

function formatDate(d) {
  if (!d) return '';
  return new Date(d).toLocaleDateString('zh-CN');
}
