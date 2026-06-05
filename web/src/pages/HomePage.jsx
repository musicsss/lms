import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Play, Clock, Film } from 'lucide-react';
import { api } from '../api/client';
import './HomePage.css';

export default function HomePage() {
  const [videos, setVideos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('推荐');
  const navigate = useNavigate();

  useEffect(() => {
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

  const tabs = ['推荐', '热门', '最新'];

  return (
    <div className="home-page">
      {/* Category tabs */}
      <div className="bili-tabs">
        {tabs.map(tab => (
          <button
            key={tab}
            className={'bili-tab' + (activeTab === tab ? ' active' : '')}
            onClick={() => setActiveTab(tab)}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Video grid */}
      <section className="bili-section">
        {loading ? (
          <div className="bili-loading">加载中...</div>
        ) : videos.length === 0 ? (
          <div className="bili-empty">
            <Film size={48} />
            <p>还没有视频，快去上传吧</p>
            <button className="bili-btn" onClick={() => navigate('/files')}>
              前往网盘上传
            </button>
          </div>
        ) : (
          <div className="video-grid">
            {videos.map(v => (
              <div key={v.id} className="video-card" onClick={() => navigate('/video/' + v.id)} onMouseEnter={(e) => { const vid = e.currentTarget.querySelector('.hover-video'); if (vid) { vid.currentTime = 0; vid.play().catch(() => {}); } }} onMouseLeave={(e) => { const vid = e.currentTarget.querySelector('.hover-video'); if (vid) { vid.pause(); vid.currentTime = 0; vid.style.opacity = 0; } }}>
                <div className="video-thumb">
                  <video className="hover-video" muted preload="none" style={{ opacity: 0, position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', objectFit: 'cover' }} onLoadedData={(e) => { if (e.target.parentElement && e.target.parentElement.matches(':hover')) { e.target.style.opacity = 1; e.target.play().catch(() => {}); } else { e.target.style.opacity = 0; } }}>
                    <source src={api.playVideoUrl(v.id)} />
                  </video>
                  <img src={api.thumbnailUrl(v.id)} alt="" className="video-thumb-img" onError={(e) => { e.target.style.display = 'none'; e.target.nextSibling.style.display = 'flex'; }} />
                  <div className="thumb-placeholder" style={{ display: 'none' }}>
                    <Play size={32} />
                  </div>
                  <span className="video-duration">{formatDuration(v.size)}</span>
                </div>
                <div className="video-info">
                  <h3 className="video-title" title={v.name}>{v.name}</h3>
                  <div className="video-meta">
                    <span><Clock size={12} /> {formatSize(v.size)}</span>
                    <span>{new Date(v.created_at).toLocaleDateString()}</span>
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
