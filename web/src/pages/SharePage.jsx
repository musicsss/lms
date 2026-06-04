import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Download, ArrowLeft } from 'lucide-react';
import { api } from '../api/client';
import './SharePage.css';

export default function SharePage() {
  const { token } = useParams();
  const [share, setShare] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    api.getShare(token)
      .then((data) => setShare(data))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  if (loading) return <div className="share-page"><div className="share-card">加载中...</div></div>;
  if (error) return <div className="share-page"><div className="share-card share-error">{error}</div></div>;
  if (!share) return null;

  const file = share.file;
  const downloadUrl = api.downloadUrl(file.id);

  return (
    <div className="share-page">
      <div className="share-card">
        <h2>共享文件</h2>
        <div className="share-file-info">
          <div className="share-file-name">{file.name}</div>
          <div className="share-file-meta">
            {formatSize(file.size)} · {file.mime_type || '未知类型'}
          </div>
        </div>
        <a href={downloadUrl} className="btn-primary share-download-btn">
          <Download size={16} />
          下载文件
        </a>
        <Link to="/login" className="share-login-link">登录 LMS</Link>
      </div>
    </div>
  );
}

function formatSize(bytes) {
  if (!bytes) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + units[i];
}
