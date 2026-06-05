import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Play, Clock, Eye, Film } from 'lucide-react';
import { api } from '../api/client';
import './HomePage.css';

export default function HomePage() {
  const [videos, setVideos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [greeting, setGreeting] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    const h = new Date().getHours();
    if (h < 6) setGreeting('夜深了');
    else if (h < 9) setGreeting('早上好');
    else if (h < 12) setGreeting('上午好');
    else if (h < 14) setGreeting('中午好');
    else if (h < 18) setGreeting('下午好');
    else setGreeting('晚上好');

    api.getRandomVideos()
      .then(data => setVideos(data.videos || []))
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const formatDuration = (size) => {
    if (!size) return '';
    const mb = size / (1024 * 1024);
    if (mb < 50) return '00:' + String(Math.floor(mb * 1.2)).padStart(2, '0');
    const mins = Math.floor(mb * 1.2 / 60);
    const secs = Math.floor(mb * 1.2 % 60);
    return String(mins).padStart(2, '0') + ':' + String(secs).padStart(2, '0');
  };

  const formatSize = (bytes) => {
    if (!bytes) return '';
    const units = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i];
  };

  return (
    <div className="home-page">
      {/* Hero区域 */}
      <section className="home-hero">
        <div className="hero-content">
          <h1>{greeting}，欢迎来到 LMS</h1>
          <p>发现和分享你喜爱的视频</p>
        </div>
      </section>

      {/* 视频推荐网格 */}
      <section className="home-section">
        <div className="section-header">
          <Film size={20} />
          <h2>热门推荐</h2>
        </div>

        {loading ? (
          <div className="home-loading">
            <div className="loader">加载中...</div>
          </div>
        ) : videos.length === 0 ? (
          <div className="home-empty">
            <Play size={48} />
            <p>还没有视频，快去上传吧</p>
            <button className="btn-primary" onClick={() => navigate('/files')}>
              前往网盘上传
            </button>
          </div>
        ) : (
          <div className="video-grid">
            {videos.map(v => (
              <div key={v.id} className="video-card" onClick={() => navigate(/video/)}>
                <div className="video-thumb">
                  <div className="thumb-placeholder">
                    <Play size={32} />
                  </div>
                  <span className="video-duration">{formatDuration(v.size)}</span>
                </div>
                <div className="video-info">
                  <h3 className="video-title" title={v.name}>{v.name}</h3>
                  <div className="video-meta">
                    <span className="video-meta-item">
                      <Eye size={12} />
                      {formatSize(v.size)}
                    </span>
                    <span className="video-meta-item">
                      <Clock size={12} />
                      {new Date(v.created_at).toLocaleDateString()}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
