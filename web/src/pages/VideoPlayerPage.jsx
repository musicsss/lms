import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { User, Calendar, Eye, HardDrive, ThumbsUp, Star, Share2, MessageCircle, Clock, Play, ChevronDown, ChevronUp, MessageSquare, Send, Users } from 'lucide-react';
import { api } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import DanmakuOverlay from '../components/DanmakuOverlay';
import './VideoPlayerPage.css';

const DANMAKU_COLORS = ['#ffffff', '#e54256', '#ffe133', '#64dd17', '#18ffff', '#ff9100'];

export default function VideoPlayerPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const videoRef = useRef(null);
  const [info, setInfo] = useState(null);
  const [recommended, setRecommended] = useState([]);
  const [comments, setComments] = useState([]);
  const [commentText, setCommentText] = useState('');
  const [replyTo, setReplyTo] = useState(null);
  const [replyText, setReplyText] = useState('');
  const [liked, setLiked] = useState(false);
  const [likeCount, setLikeCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [commentLoading, setCommentLoading] = useState(false);
  const [submittingComment, setSubmittingComment] = useState(false);
  const [submittingReply, setSubmittingReply] = useState(false);
  const [videoError, setVideoError] = useState(false);
  const [danmakuOpen, setDanmakuOpen] = useState(false);
  const [danmakuEnabled, setDanmakuEnabled] = useState(true);
  const [danmakuText, setDanmakuText] = useState('');
  const [danmakuColor, setDanmakuColor] = useState('#ffffff');
  const [danmakuList, setDanmakuList] = useState([]);
  const [watcherCount, setWatcherCount] = useState(0);

  const loadComments = useCallback(() => {
    if (!id) return;
    setCommentLoading(true);
    api.getComments(id)
      .then(data => setComments(data.comments || []))
      .catch(err => console.error('Failed to load comments:', err))
      .finally(() => setCommentLoading(false));
  }, [id]);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    setVideoError(false);

    Promise.all([
      api.getVideoInfo(id),
      api.getRandomVideos(),
      api.getVideoLikeStatus(id).catch(() => ({ liked: false, count: 0 })),
      api.getDanmaku(id).then(d => d.danmaku || []).catch(() => []),
    ])
      .then(([infoData, videoData, likeData, danmakuData]) => {
        setInfo(infoData);
        setRecommended((videoData.videos || []).filter(v => v.id !== parseInt(id)).slice(0, 8));
        setLiked(likeData.liked);
        setLikeCount(likeData.count);
        setDanmakuList(danmakuData);
      })
      .catch(err => {
        console.error('Failed to load video:', err);
      })
      .finally(() => setLoading(false));

    loadComments();

    // Heartbeat + watchers polling
    let heartbeatTimer;
    if (id) {
      const poll = () => {
        api.videoHeartbeat(id).catch(() => {});
        api.videoWatchers(id)
          .then(d => setWatcherCount(d.count || 0))
          .catch(() => {});
        heartbeatTimer = setTimeout(poll, 8000);
      };
      poll();
    }
    return () => clearTimeout(heartbeatTimer);
  }, [id, loadComments]);

  const formatDate = (d) => {
    if (!d) return '';
    const date = new Date(d);
    const now = new Date();
    const diff = now - date;
    const mins = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);
    if (mins < 60) return mins + '分钟前';
    if (hours < 24) return hours + '小时前';
    if (days < 30) return days + '天前';
    return date.toLocaleDateString('zh-CN');
  };

  const formatSize = (bytes) => {
    if (!bytes) return '';
    const units = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i];
  };

  const formatNumber = (n) => {
    if (!n) return '0';
    if (n >= 10000) return (n / 10000).toFixed(1) + '万';
    return n.toString();
  };

  const formatDanmakuTime = (sec) => {
    if (sec == null) return '00:00';
    const m = Math.floor(sec / 60);
    const s = Math.floor(sec % 60);
    return String(m).padStart(2, '0') + ':' + String(s).padStart(2, '0');
  };

  const handleLike = async () => {
    try {
      const data = await api.toggleVideoLike(id);
      setLiked(data.liked);
      setLikeCount(prev => data.liked ? prev + 1 : Math.max(0, prev - 1));
    } catch (err) {
      console.error('Like failed:', err);
    }
  };

  const handleSubmitComment = async () => {
    if (!commentText.trim()) return;
    setSubmittingComment(true);
    try {
      await api.createComment(id, commentText.trim());
      setCommentText('');
      loadComments();
    } catch (err) {
      console.error('Comment failed:', err);
    } finally {
      setSubmittingComment(false);
    }
  };

  const handleSubmitReply = async (parentId) => {
    if (!replyText.trim()) return;
    setSubmittingReply(true);
    try {
      await api.createComment(id, replyText.trim(), parentId);
      setReplyText('');
      setReplyTo(null);
      loadComments();
    } catch (err) {
      console.error('Reply failed:', err);
    } finally {
      setSubmittingReply(false);
    }
  };

  const handleVideoError = () => {
    console.error('Video playback error for id:', id);
    setVideoError(true);
  };

  const handleDanmakuSend = () => {
    if (!danmakuText.trim() || !videoRef.current) return;
    const timeSec = videoRef.current.currentTime;
    api.sendDanmaku(id, danmakuText.trim(), timeSec, danmakuColor, 25, 1)
      .then(() => setDanmakuText(''))
      .catch(console.error);
  };

  if (loading) return <div className="video-page-loading"><div className="loader">加载中...</div></div>;
  if (!info) return <div className="video-page-loading">视频不存在或已被删除</div>;

  const tags = info.tags ? info.tags.split(/[,，\s]+/).filter(Boolean) : [];

  return (
    <div className="bili-video-page">
      <div className="bili-video-main">
        {/* ===== 标题 & 播放量/日期 ===== */}
        <div className="bili-video-info-top">
          <h1 className="bili-video-title">{info.name}</h1>
          <div className="bili-video-meta-row">
            <div className="bili-video-stats">
              <span className="bili-stat-item">
                <Eye size={16} />
                {formatNumber(info.view_count || info.like_count || 0)} 播放
              </span>
              {watcherCount > 0 && (
                <span className="bili-stat-item">
                  <Users size={16} />
                  {formatNumber(watcherCount)} 正在观看
                </span>
              )}
              <span className="bili-stat-item">
                <Calendar size={16} />
                {new Date(info.created_at).toLocaleDateString('zh-CN')}
              </span>
              <span className="bili-stat-item">
                <HardDrive size={16} />
                {formatSize(info.size)}
              </span>
            </div>
          </div>
        </div>

        {/* ===== 播放器 ===== */}
        <div className="bili-player-wrap">
          {videoError ? (
            <div className="bili-player-error">
              <p>视频加载失败</p>
              <p className="bili-player-error-sub">请尝试刷新页面或检查视频文件是否完整</p>
            </div>
          ) : (
            <>
              <video
                ref={videoRef}
                controls
                autoPlay
                className="bili-player"
                key={id}
                onError={handleVideoError}
              >
                <source src={api.playVideoUrl(id)} />
                您的浏览器不支持视频播放
              </video>
              {danmakuEnabled && (
                <DanmakuOverlay
                  videoRef={videoRef}
                  danmakuEnabled={danmakuEnabled}
                  danmakuList={danmakuList}
                />
              )}
            </>
          )}
        </div>

        {/* ===== 播放器底部栏：观看人数 / 弹幕开关 / 弹幕输入 ===== */}
        <div className="bili-player-bar">
          <div className="bili-player-bar-left">
            <div className="bili-watching-count">
              <Users size={16} />
              
            </div>
          </div>
          <div className="bili-player-bar-right">
            <div className="bili-danmaku-colors">
              {DANMAKU_COLORS.map(c => (
                <button
                  key={c}
                  className={'bili-dm-color-btn' + (danmakuColor === c ? ' active' : '')}
                  style={{ backgroundColor: c }}
                  onClick={() => setDanmakuColor(c)}
                  title={`颜色 ${c}`}
                />
              ))}
            </div>
            <button
              className={'bili-danmaku-toggle' + (danmakuEnabled ? '' : ' disabled')}
              onClick={() => setDanmakuEnabled(!danmakuEnabled)}
            >
              <MessageSquare size={16} />
              <span>弹幕</span>
            </button>
            <div className="bili-danmaku-input-wrap">
              <input
                className="bili-danmaku-input"
                type="text"
                placeholder="发个弹幕呗~"
                value={danmakuText}
                onChange={e => setDanmakuText(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && handleDanmakuSend()}
              />
              <button
                className="bili-danmaku-send"
                disabled={!danmakuText.trim()}
                onClick={handleDanmakuSend}
              >
                <Send size={14} />
              </button>
            </div>
          </div>
        </div>

        {/* ===== 操作按钮 ===== */}
        <div className="bili-video-actions">
          <button className={'bili-action-btn' + (liked ? ' liked' : '')} onClick={handleLike}>
            <ThumbsUp size={20} fill={liked ? '#fb7299' : 'none'} />
            <span>{formatNumber(likeCount)}</span>
          </button>
          <button className="bili-action-btn">
            <Star size={20} />
            <span>收藏</span>
          </button>
          <button className="bili-action-btn" onClick={() => {
            navigator.clipboard?.writeText(window.location.href);
            alert('链接已复制到剪贴板');
          }}>
            <Share2 size={20} />
            <span>分享</span>
          </button>
        </div>

        {/* ===== 简介 ===== */}
        {info.description && (
          <div className="bili-video-desc">
            <p>{info.description}</p>
          </div>
        )}

        {/* ===== 标签 ===== */}
        {tags.length > 0 && (
          <div className="bili-video-tags">
            {tags.map((tag, i) => (
              <span key={i} className="bili-tag">{tag}</span>
            ))}
          </div>
        )}

        {/* ===== 评论区 ===== */}
        <div className="bili-comment-section">
          <h3 className="bili-comment-title">
            <MessageCircle size={20} />
            评论 ({formatNumber(info.comment_count || comments.length)})
          </h3>

          <div className="bili-comment-input-wrap">
            <div className="bili-comment-avatar">
              {user?.avatar_url ? (
                <img src={user.avatar_url} alt="" className="bili-avatar-img" />
              ) : (
                user?.username ? user.username.charAt(0).toUpperCase() : 'U'
              )}
            </div>
            <div className="bili-comment-input-area">
              <textarea
                className="bili-comment-textarea"
                placeholder="发一条友善的评论吧"
                value={commentText}
                onChange={e => setCommentText(e.target.value)}
                rows={3}
              />
              <div className="bili-comment-input-actions">
                <button
                  className="bili-comment-submit"
                  disabled={!commentText.trim() || submittingComment}
                  onClick={handleSubmitComment}
                >
                  {submittingComment ? '发送中...' : '发表评论'}
                </button>
              </div>
            </div>
          </div>

          <div className="bili-comments-list">
            {commentLoading ? (
              <div className="bili-loading-text">加载评论中...</div>
            ) : comments.length === 0 ? (
              <div className="bili-no-comments">暂无评论，快来抢沙发吧！</div>
            ) : (
              comments.map(comment => (
                <div key={comment.id} className="bili-comment-item">
                  <div className="bili-comment-avatar-sm">
                    {comment.user?.avatar_url ? (
                      <img src={comment.user.avatar_url} alt="" className="bili-avatar-img" />
                    ) : (
                      comment.user?.username ? comment.user.username.charAt(0).toUpperCase() : 'U'
                    )}
                  </div>
                  <div className="bili-comment-body">
                    <div className="bili-comment-header">
                      <span className="bili-comment-user">{comment.user?.nickname || comment.user?.username || '用户'}</span>
                      <span className="bili-comment-time">{formatDate(comment.created_at)}</span>
                    </div>
                    <div className="bili-comment-content">{comment.content}</div>
                    <div className="bili-comment-footer">
                      <button
                        className="bili-comment-reply-btn"
                        onClick={() => setReplyTo(replyTo === comment.id ? null : comment.id)}
                      >
                        {replyTo === comment.id ? '取消回复' : '回复'}
                      </button>
                    </div>

                    {replyTo === comment.id && (
                      <div className="bili-reply-input-wrap">
                        <textarea
                          className="bili-comment-textarea bili-reply-textarea"
                          placeholder={'回复 @' + (comment.user?.username || '用户')}
                          value={replyText}
                          onChange={e => setReplyText(e.target.value)}
                          rows={2}
                        />
                        <div className="bili-comment-input-actions">
                          <button
                            className="bili-comment-submit bili-reply-submit"
                            disabled={!replyText.trim() || submittingReply}
                            onClick={() => handleSubmitReply(comment.id)}
                          >
                            {submittingReply ? '发送中...' : '回复'}
                          </button>
                        </div>
                      </div>
                    )}

                    {comment.replies && comment.replies.length > 0 && (
                      <div className="bili-replies-list">
                        {comment.replies.map(reply => (
                          <div key={reply.id} className="bili-reply-item">
                            <div className="bili-comment-avatar-xs">
                              {reply.user?.avatar_url ? (
                                <img src={reply.user.avatar_url} alt="" className="bili-avatar-img" />
                              ) : (
                                reply.user?.username ? reply.user.username.charAt(0).toUpperCase() : 'U'
                              )}
                            </div>                            <div className="bili-comment-body">
                              <div className="bili-comment-header">
                                <span className="bili-comment-user">{reply.user?.nickname || reply.user?.username || '用户'}</span>
                                <span className="bili-comment-time">{formatDate(reply.created_at)}</span>
                              </div>
                              <div className="bili-comment-content">
                                {reply.parent_id && <span className="bili-reply-at">@</span>}
                                {reply.content}
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* ===== 右侧边栏 ===== */}
      <aside className="bili-video-sidebar">
        <div className="sidebar-uploader-card">
          <div className="sidebar-uploader-top">
            <div className="bili-avatar sidebar-uploader-avatar">
              {info.user?.avatar_url ? (
                <img src={info.user.avatar_url} alt="" className="bili-avatar-img" />
              ) : (
                info.user?.username ? info.user.username.charAt(0).toUpperCase() : <User size={18} />
              )}
            </div>
            <div className="sidebar-uploader-info">
              <span className="sidebar-uploader-name">{info.user?.nickname || info.user?.username || '未知用户'}</span>
              <span className="sidebar-uploader-stats">
                {formatNumber(info.like_count || 0)} 获赞
              </span>
            </div>
          </div>
          <div className="sidebar-uploader-desc">
            <span>UP主</span>
          </div>
          <button className="sidebar-subscribe-btn">关注</button>
        </div>

        <div className={'sidebar-danmaku-section' + (danmakuOpen ? ' open' : '')}>
          <div className="sidebar-danmaku-header" onClick={() => setDanmakuOpen(!danmakuOpen)}>
            <div className="sidebar-danmaku-header-left">
              <MessageSquare size={16} />
              <span>弹幕列表</span>
            </div>
            <div className="sidebar-danmaku-header-right">
              <span className="sidebar-danmaku-count">{formatNumber(danmakuList.length)}</span>
              {danmakuOpen ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
            </div>
          </div>
          <div className="sidebar-danmaku-body">
            {danmakuList.length === 0 ? (
              <div className="sidebar-danmaku-empty">暂无弹幕</div>
            ) : (
              <div className="sidebar-danmaku-list">
                {danmakuList.map((dm, i) => (
                  <div key={dm.id || i} className="sidebar-danmaku-item">
                    <span className="sidebar-danmaku-time">{formatDanmakuTime(dm.time_sec)}</span>
                    <span className="sidebar-danmaku-content" style={{ color: '#18191c' }}>{dm.content}</span>
                    <span className="sidebar-danmaku-user">{dm.user?.nickname || dm.user?.username || '用户'}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="sidebar-recommend-section">
          <h3 className="sidebar-title">推荐视频</h3>
          <div className="sidebar-list">
            {recommended.map(v => (
              <div key={v.id} className="sidebar-card" onClick={() => navigate('/video/' + v.id)}>
                <div className="sidebar-thumb">
                  <div className="sidebar-thumb-placeholder">
                    <Play size={16} />
                  </div>
                </div>
                <div className="sidebar-card-info">
                  <div className="sidebar-card-title" title={v.name}>{v.name}</div>
                  <div className="sidebar-card-meta">
                    <span>{v.user?.nickname || v.user?.username || '未知'}</span>
                    <span>{' '}{formatSize(v.size)}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </aside>
    </div>
  );
}



