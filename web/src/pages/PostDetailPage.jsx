import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ArrowLeft, Heart, MessageSquare } from 'lucide-react';
import { api } from '../api/client';
import './PostDetailPage.css';

export default function PostDetailPage() {
  const { boardId, postId } = useParams();
  const [post, setPost] = useState(null);
  const [loading, setLoading] = useState(true);
  const [replyContent, setReplyContent] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const loadPost = async () => {
    try {
      const data = await api.getPost(postId);
      setPost(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadPost(); }, [postId]);

  const handleReply = async (e) => {
    e.preventDefault();
    if (!replyContent.trim()) return;
    setSubmitting(true);
    try {
      await api.replyPost(postId, replyContent);
      setReplyContent('');
      loadPost();
    } catch (err) {
      alert(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleLike = async (id) => {
    try {
      await api.toggleLike(id);
      loadPost();
    } catch (err) {
      alert(err.message);
    }
  };

  if (loading) return <div className="post-detail-loading">加载中...</div>;
  if (!post) return <div className="post-detail-loading">帖子不存在</div>;

  return (
    <div className="post-detail-page">
      <Link to={`/forum/${boardId}`} className="btn-sm post-back">
        <ArrowLeft size={16} />
        返回列表
      </Link>

      <article className="post-main">
        <h1 className="post-title">{post.title}</h1>
        <div className="post-meta">
          <span>{post.user?.username || '匿名'}</span>
          <span className="meta-sep">·</span>
          <span>{formatDate(post.created_at)}</span>
        </div>
        <div className="post-body">{post.content}</div>
        <div className="post-footer">
          <button className="btn-sm" onClick={() => handleLike(post.id)}>
            <Heart size={14} />
            {post.like_count || 0}
          </button>
        </div>
      </article>

      {post.replies && post.replies.length > 0 && (
        <section className="replies-section">
          <h3 className="replies-title">
            <MessageSquare size={16} />
            回复 ({post.replies.length})
          </h3>
          {post.replies.map((reply) => (
            <div key={reply.id} className="reply-item">
              <div className="reply-meta">
                <span className="reply-author">{reply.user?.username || '匿名'}</span>
                <span>{formatDate(reply.created_at)}</span>
              </div>
              <div className="reply-content">{reply.content}</div>
              <button className="btn-sm-subtle" onClick={() => handleLike(reply.id)}>
                <Heart size={12} />
                {reply.like_count || 0}
              </button>
            </div>
          ))}
        </section>
      )}

      <form className="reply-form" onSubmit={handleReply}>
        <textarea
          value={replyContent}
          onChange={(e) => setReplyContent(e.target.value)}
          placeholder="写下你的回复..."
          rows={3}
          required
        />
        <button type="submit" disabled={submitting} className="btn-sm">
          {submitting ? '提交中...' : '回复'}
        </button>
      </form>
    </div>
  );
}

function formatDate(d) {
  if (!d) return '';
  return new Date(d).toLocaleDateString('zh-CN');
}
