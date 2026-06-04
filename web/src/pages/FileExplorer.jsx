import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import {
  Folder, File, Upload, FolderPlus, Trash2, Download,
  Share2, ArrowLeft, FolderOpen, RefreshCw,
} from 'lucide-react';
import { api } from '../api/client';
import './FileExplorer.css';

export default function FileExplorer() {
  const { folderId } = useParams();
  const navigate = useNavigate();
  const [files, setFiles] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showMkdir, setShowMkdir] = useState(false);
  const [dirName, setDirName] = useState('');
  const [uploading, setUploading] = useState(false);
  const [shareToken, setShareToken] = useState(null);

  const parentId = folderId || null;

  const loadFiles = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.listFiles(parentId);
      setFiles(data.files || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [parentId]);

  useEffect(() => { loadFiles(); }, [loadFiles]);

  const handleUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    setUploading(true);
    try {
      await api.uploadFile(file, parentId ? parseInt(parentId) : null);
      e.target.value = '';
      loadFiles();
    } catch (err) {
      alert(err.message);
    } finally {
      setUploading(false);
    }
  };

  const handleMkdir = async () => {
    if (!dirName.trim()) return;
    try {
      await api.mkdir(dirName, parentId ? parseInt(parentId) : null);
      setDirName('');
      setShowMkdir(false);
      loadFiles();
    } catch (err) {
      alert(err.message);
    }
  };

  const handleDelete = async (id) => {
    if (!confirm('确认删除？')) return;
    try {
      await api.deleteFile(id);
      loadFiles();
    } catch (err) {
      alert(err.message);
    }
  };

  const handleShare = async (id) => {
    try {
      const data = await api.shareFile(id, '', 0);
      const url = `${window.location.origin}/share/${data.token}`;
      setShareToken(url);
    } catch (err) {
      alert(err.message);
    }
  };

  const openItem = (file) => {
    if (file.is_dir) {
      navigate(`/files/${file.id}`);
    } else if (file.is_video) {
      navigate(`/video/${file.id}`);
    }
  };

  return (
    <div className="file-explorer">
      <div className="explorer-toolbar">
        <div className="toolbar-left">
          {parentId && (
            <button className="btn-sm" onClick={() => navigate('/files')} title="返回根目录">
              <ArrowLeft size={16} />
            </button>
          )}
          <button className="btn-sm" onClick={loadFiles} title="刷新">
            <RefreshCw size={16} />
          </button>
          <label className="btn-sm" title="上传文件">
            <Upload size={16} />
            <input type="file" hidden onChange={handleUpload} disabled={uploading} />
          </label>
          <button className="btn-sm" onClick={() => setShowMkdir(!showMkdir)} title="新建文件夹">
            <FolderPlus size={16} />
          </button>
        </div>
        {uploading && <span className="upload-status">上传中...</span>}
      </div>

      {showMkdir && (
        <div className="mkdir-row">
          <input
            type="text"
            value={dirName}
            onChange={(e) => setDirName(e.target.value)}
            placeholder="文件夹名称"
            onKeyDown={(e) => e.key === 'Enter' && handleMkdir()}
            autoFocus
          />
          <button className="btn-sm" onClick={handleMkdir}>创建</button>
          <button className="btn-sm" onClick={() => setShowMkdir(false)}>取消</button>
        </div>
      )}

      {shareToken && (
        <div className="share-result">
          <span>分享链接：</span>
          <input readOnly value={shareToken} onFocus={(e) => e.target.select()} />
          <button className="btn-sm" onClick={() => setShareToken(null)}>关闭</button>
        </div>
      )}

      {loading ? (
        <div className="explorer-loading">加载中...</div>
      ) : files.length === 0 ? (
        <div className="explorer-empty">
          <FolderOpen size={48} />
          <p>此目录为空</p>
        </div>
      ) : (
        <div className="file-grid">
          {files.map((f) => (
            <div key={f.id} className="file-card" onDoubleClick={() => openItem(f)}>
              <div className="file-icon">
                {f.is_dir ? <Folder size={36} /> : <File size={36} />}
              </div>
              <div className="file-name" title={f.name}>{f.name}</div>
              <div className="file-meta">
                {f.is_dir ? '文件夹' : formatSize(f.size)}
              </div>
              <div className="file-actions">
                {!f.is_dir && (
                  <>
                    <a href={api.downloadUrl(f.id)} className="btn-icon-sm" title="下载">
                      <Download size={14} />
                    </a>
                    <button className="btn-icon-sm" onClick={() => handleShare(f.id)} title="分享">
                      <Share2 size={14} />
                    </button>
                  </>
                )}
                <button className="btn-icon-sm danger" onClick={() => handleDelete(f.id)} title="删除">
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function formatSize(bytes) {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + units[i];
}
