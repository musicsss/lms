import { useParams, Link } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { api } from '../api/client';
import './VideoPlayerPage.css';

export default function VideoPlayerPage() {
  const { id } = useParams();

  return (
    <div className="video-page">
      <div className="video-toolbar">
        <Link to="/files" className="btn-sm">
          <ArrowLeft size={16} />
          返回网盘
        </Link>
      </div>
      <div className="video-container">
        <video controls autoPlay className="video-player">
          <source src={api.downloadUrl(id)} />
          您的浏览器不支持视频播放
        </video>
      </div>
    </div>
  );
}
