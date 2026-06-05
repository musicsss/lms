import { useState, useEffect, useRef, useCallback } from 'react';
import Cropper from 'react-easy-crop';
import { useParams, useNavigate } from 'react-router-dom';
import { User, Film, FileText, MessageSquare, Heart, Settings, Grid, List, Edit3, Save,
  Camera, Calendar, ChevronLeft, ChevronRight, Lock, Check, X, Clock, Play, Crop, RotateCcw, ZoomIn, ZoomOut } from 'lucide-react';
import { api } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import './UserProfilePage.css';

const TABS = ['主页', '动态', '投稿', '收藏', '设置'];
const SUBMIT_TABS = ['视频', '文件', '论坛发帖'];

export default function UserProfilePage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user: authUser, updateUser } = useAuth();
  const isSelf = !id || String(id) === String(authUser?.id);

  const [profile, setProfile] = useState(null);
  const [activeTab, setActiveTab] = useState('主页');
  const [submitTab, setSubmitTab] = useState('视频');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [userFiles, setUserFiles] = useState({ files: [], total: 0, page: 1 });
  const [userPosts, setUserPosts] = useState({ posts: [], total: 0, page: 1 });
  const [likedVideos, setLikedVideos] = useState({ videos: [], total: 0, page: 1 });
  const [tabLoading, setTabLoading] = useState(false);

  const [nickname, setNickname] = useState('');
  const [bio, setBio] = useState('');
  const [email, setEmail] = useState('');
  const [avatarUrl, setAvatarUrl] = useState('');
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [profileMsg, setProfileMsg] = useState(null);
  const [pwdMsg, setPwdMsg] = useState(null);
  const [savingProfile, setSavingProfile] = useState(false);
  const [savingPwd, setSavingPwd] = useState(false);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const avatarInputRef = useRef(null);

  // Crop state
  const [cropImage, setCropImage] = useState(null);
  const [crop, setCrop] = useState({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [rotation, setRotation] = useState(0);
  const [croppedAreaPixels, setCroppedAreaPixels] = useState(null);
  const [showCropper, setShowCropper] = useState(false);
  const [cropLoading, setCropLoading] = useState(false);

  const formatDuration = (bytes) => {
    if (!bytes) return '';
    const mb = bytes / (1024 * 1024);
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

  useEffect(() => {
    if (!authUser) return;
    setLoading(true);
    setError('');
    const fetchProfile = isSelf ? api.getUserProfile() : api.getPublicUserProfile(id);
    fetchProfile
      .then(data => {
        setProfile(data);
        setNickname(data.nickname || '');
        setBio(data.bio || '');
        setEmail(data.email || '');
        setAvatarUrl(data.avatar_url || '');
      })
      .catch(err => setError(err.message))
      .finally(() => setLoading(false));
  }, [id, authUser, isSelf]);

  const loadTabData = async (tab) => {
    setTabLoading(true);
    try {
      if (tab === '投稿') {
        const type = submitTab === '视频' ? 'video' : 'all';
        const data = await api.getUserFiles(type, 1);
        setUserFiles({ files: data.files || [], total: data.total || 0, page: 1 });
      } else if (tab === '动态') {
        const data = await api.getUserPosts(1);
        setUserPosts({ posts: data.posts || [], total: data.total || 0, page: 1 });
      } else if (tab === '收藏') {
        const data = await api.getUserLikedVideos(1);
        setLikedVideos({ videos: data.videos || [], total: data.total || 0, page: 1 });
      }
    } catch (err) {
      console.error('Tab load failed:', err);
    } finally {
      setTabLoading(false);
    }
  };

  useEffect(() => {
    if (!profile) return;
    if (['投稿', '动态', '收藏'].includes(activeTab)) {
      loadTabData(activeTab);
    }
  }, [activeTab, submitTab, profile]);

  const loadSubmitPage = async (p) => {
    setTabLoading(true);
    try {
      const type = submitTab === '视频' ? 'video' : 'all';
      const data = await api.getUserFiles(type, p);
      setUserFiles({ files: data.files || [], total: data.total || 0, page: p });
    } catch (err) {
      console.error(err);
    } finally {
      setTabLoading(false);
    }
  };

  const loadPostsPage = async (p) => {
    setTabLoading(true);
    try {
      const data = await api.getUserPosts(p);
      setUserPosts({ posts: data.posts || [], total: data.total || 0, page: p });
    } catch (err) {
      console.error(err);
    } finally {
      setTabLoading(false);
    }
  };

  const loadLikedPage = async (p) => {
    setTabLoading(true);
    try {
      const data = await api.getUserLikedVideos(p);
      setLikedVideos({ videos: data.videos || [], total: data.total || 0, page: p });
    } catch (err) {
      console.error(err);
    } finally {
      setTabLoading(false);
    }
  };

  const handleAvatarUpload = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    // Validate file type
    const allowedTypes = ['image/jpeg', 'image/png', 'image/webp', 'image/gif'];
    if (!allowedTypes.includes(file.type)) {
      setProfileMsg({ ok: false, text: '只支持 jpg, png, webp, gif 格式' });
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      setProfileMsg({ ok: false, text: '文件太大，最大 5MB' });
      return;
    }
    setUploadingAvatar(true);
    setProfileMsg(null);
    try {
      const data = await api.uploadAvatar(file);
      setAvatarUrl(data.avatar_url);
      updateUser({ avatar_url: data.avatar_url });
      setProfileMsg({ ok: true, text: '头像已更新' });
    } catch (err) {
      console.error('Avatar upload failed:', err);
      setProfileMsg({ ok: false, text: err.message || '上传失败' });
    } finally {
      setUploadingAvatar(false);
      // Reset file input
      if (avatarInputRef.current) avatarInputRef.current.value = '';
    }
  };

  const handleSaveProfile = async () => {
    setSavingProfile(true);
    setProfileMsg(null);
    try {
      await api.updateUserProfile({ nickname, bio, email, avatar_url: avatarUrl });
      setProfileMsg({ ok: true, text: '个人信息更新成功' });
      updateUser({ nickname, bio, email, avatar_url: avatarUrl });
    } catch (err) {
      setProfileMsg({ ok: false, text: err.message || '更新失败' });
    } finally {
      setSavingProfile(false);
    }
  };

  const handleSavePassword = async () => {
    if (newPassword !== confirmPassword) {
      setPwdMsg({ ok: false, text: '两次密码输入不一致' });
      return;
    }
    if (newPassword.length < 6) {
      setPwdMsg({ ok: false, text: '密码长度至少6位' });
      return;
    }
    setSavingPwd(true);
    setPwdMsg(null);
    try {
      await api.updateUserPassword({ old_password: oldPassword, new_password: newPassword });
      setPwdMsg({ ok: true, text: '密码修改成功' });
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err) {
      console.error('Password update failed:', err);
      setPwdMsg({ ok: false, text: err.message || '修改失败' });
    } finally {
      setSavingPwd(false);
    }
  };

  const onCropComplete = useCallback((croppedArea, croppedAreaPixels) => {
    setCroppedAreaPixels(croppedAreaPixels);
  }, []);

  const createImage = (url) =>
    new Promise((resolve, reject) => {
      const image = new Image();
      image.addEventListener('load', () => resolve(image));
      image.addEventListener('error', (error) => reject(error));
      image.setAttribute('crossOrigin', 'anonymous');
      image.src = url;
    });

  function getRadianAngle(degreeValue) {
    return (degreeValue * Math.PI) / 180;
  }

  async function cropImageBlob(imageSrc, pixelCrop, rotation) {
    const image = await createImage(imageSrc);
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    const maxSize = Math.max(image.width, image.height);
    const safeArea = 2 * Math.sqrt(maxSize * maxSize * 2);
    canvas.width = safeArea;
    canvas.height = safeArea;
    ctx.translate(safeArea / 2, safeArea / 2);
    ctx.rotate(getRadianAngle(rotation));
    ctx.translate(-safeArea / 2, -safeArea / 2);
    ctx.drawImage(image, safeArea / 2 - image.width * 0.5, safeArea / 2 - image.height * 0.5);
    const data = ctx.getImageData(0, 0, safeArea, safeArea);
    canvas.width = pixelCrop.width;
    canvas.height = pixelCrop.height;
    ctx.putImageData(data, Math.round(0 - safeArea / 2 + image.width * 0.5 - pixelCrop.x), Math.round(0 - safeArea / 2 + image.height * 0.5 - pixelCrop.y));
    return new Promise((resolve) => {
      canvas.toBlob((blob) => { resolve(blob); }, 'image/jpeg', 0.95);
    });
  }

  const uploadCroppedImage = async () => {
    if (!croppedAreaPixels) return;
    setCropLoading(true);
    try {
      const croppedBlob = await cropImageBlob(cropImage, croppedAreaPixels, rotation);
      const file = new File([croppedBlob], 'avatar.jpeg', { type: 'image/jpeg' });
      const data = await api.uploadAvatar(file);
      setAvatarUrl(data.avatar_url);
      setShowCropper(false);
      setCropImage(null);
      setProfileMsg({ ok: true, text: '头像已更新' });
      updateUser({ avatar_url: data.avatar_url });
    } catch (err) {
      console.error('Avatar upload failed:', err);
      setProfileMsg({ ok: false, text: err.message || '上传失败' });
    } finally {
      setCropLoading(false);
    }
  };

  const closeCropper = () => {
    setShowCropper(false);
    setCropImage(null);
  };

  if (loading) {
    return <div className='profile-page'><div className='profile-loading'>加载中...</div></div>;
  }

  if (error) {
    return (
      <div className='profile-page'>
        <div className='profile-loading' style={{ color: '#f73131' }}>{error}</div>
      </div>
    );
  }

  if (!profile) {
    return (
      <div className='profile-page'>
        <div className='profile-loading'>用户不存在</div>
      </div>
    );
  }

  const Pagination = ({ page, total, pageSize, onPage }) => {
    const totalPages = Math.ceil(total / pageSize);
    if (totalPages <= 1) return null;
    return (
      <div className='bili-pagination'>
        <button disabled={page <= 1} onClick={() => onPage(page - 1)}>
          <ChevronLeft size={16} />
        </button>
        <span>{page} / {totalPages}</span>
        <button disabled={page >= totalPages} onClick={() => onPage(page + 1)}>
          <ChevronRight size={16} />
        </button>
      </div>
    );
  };

  // Cropper modal
  {showCropper && (
    <div className='cropper-overlay' onClick={closeCropper}>
      <div className='cropper-modal' onClick={(e) => e.stopPropagation()}>
        <div className='cropper-header'>
          <h3>裁剪头像</h3>
          <button className='cropper-close' onClick={closeCropper}><X size={20} /></button>
        </div>
        <div className='cropper-container'>
          <Cropper
            image={cropImage}
            crop={crop}
            zoom={zoom}
            rotation={rotation}
            aspect={1 / 1}
            cropShape='round'
            showGrid={false}
            onCropChange={setCrop}
            onZoomChange={setZoom}
            onRotationChange={setRotation}
            onCropComplete={onCropComplete}
          />
        </div>
        <div className='cropper-controls'>
          <div className='cropper-zoom'>
            <button onClick={() => setZoom(Math.max(1, zoom - 0.1))}><ZoomOut size={18} /></button>
            <input type='range' min={1} max={3} step={0.01} value={zoom} onChange={(e) => setZoom(Number(e.target.value))} className='cropper-slider' />
            <button onClick={() => setZoom(Math.min(3, zoom + 0.1))}><ZoomIn size={18} /></button>
          </div>
          <div className='cropper-rotate'>
            <button onClick={() => setRotation((rotation - 90) % 360)}><RotateCcw size={18} /></button>
            <span>{rotation}°</span>
          </div>
        </div>
        <div className='cropper-actions'>
          <button className='cropper-btn cancel' onClick={closeCropper}>取消</button>
          <button className='cropper-btn confirm' onClick={uploadCroppedImage} disabled={cropLoading}>
            {cropLoading ? '上传中...' : '确定裁剪'}
          </button>
        </div>
      </div>
    </div>
  )}
  return (
    <div className='profile-page'>
      <div className='profile-header-card'>
        <div className='profile-cover' />
        <div className='profile-header-content'>
          <div className='profile-avatar-section'>
            <div className='profile-avatar-large'>
              {profile.avatar_url ? (
                <img src={profile.avatar_url} alt='avatar' className='profile-avatar-img' key={profile.avatar_url} />
              ) : (
                <div className='profile-avatar-text'>
                  {(profile.nickname || profile.username).charAt(0).toUpperCase()}
                </div>
              )}
            </div>
          </div>
          <div className='profile-info-section'>
            <h1 className='profile-username'>{profile.nickname || profile.username}</h1>
            <div className='profile-bio'>{profile.bio || '这个人很懒，什么都没留下~'}</div>
            <div className='profile-joined'>
              <Calendar size={14} />
              加入时间: {new Date(profile.created_at).toLocaleDateString()}
            </div>
          </div>
        </div>
      </div>

      <div className='profile-tabs'>
        {TABS.map(tab => {
          if (!isSelf && tab === '设置') return null;
          return (
            <button key={tab}
              className={'profile-tab' + (activeTab === tab ? ' active' : '')}
              onClick={() => setActiveTab(tab)}>
              {tab === '主页' && <User size={16} />}
              {tab === '动态' && <MessageSquare size={16} />}
              {tab === '投稿' && <Film size={16} />}
              {tab === '收藏' && <Heart size={16} />}
              {tab === '设置' && <Settings size={16} />}
              {tab}
            </button>
          );
        })}
      </div>

      <div className='profile-tab-content'>
        {tabLoading && <div className='profile-loading'>加载中...</div>}

        {activeTab === '主页' && !tabLoading && (
          <div className='profile-home'>
            <div className='profile-stats-row'>
              <div className='profile-stat-card'>
                <Film size={24} />
                <span className='profile-stat-num'>{userFiles.total || 0}</span>
                <span className='profile-stat-label'>投稿</span>
              </div>
              <div className='profile-stat-card'>
                <MessageSquare size={24} />
                <span className='profile-stat-num'>{userPosts.total || 0}</span>
                <span className='profile-stat-label'>帖子</span>
              </div>
              <div className='profile-stat-card'>
                <Heart size={24} />
                <span className='profile-stat-num'>{likedVideos.total || 0}</span>
                <span className='profile-stat-label'>获赞</span>
              </div>
            </div>
          </div>
        )}

        {activeTab === '动态' && !tabLoading && (
          <div className='profile-posts'>
            {userPosts.posts.length === 0 ? (
              <div className='profile-empty'>暂无动态</div>
            ) : (
              <>
                {userPosts.posts.map(post => (
                  <div key={post.id} className='profile-post-card' onClick={() => navigate('/forum/' + post.board_id + '/' + post.id)}>
                    <h4>{post.title}</h4>
                    <p>{post.content?.substring(0, 200)}</p>
                    <span className='profile-post-date'>{new Date(post.created_at).toLocaleDateString()}</span>
                  </div>
                ))}
                <Pagination page={userPosts.page} total={userPosts.total} pageSize={10} onPage={loadPostsPage} />
              </>
            )}
          </div>
        )}

        {activeTab === '投稿' && !tabLoading && (
          <div className='profile-submissions'>
            <div className='profile-submit-tabs'>
              {SUBMIT_TABS.map(st => (
                <button key={st} className={'profile-tab' + (submitTab === st ? ' active' : '')} onClick={() => setSubmitTab(st)}>{st}</button>
              ))}
            </div>
            <div className='profile-submit-content'>
              {submitTab === '视频' && (
                <>
                  {userFiles.files.filter(f => f.mime_type?.startsWith('video/')).length === 0 ? (
                    <div className='profile-empty'>暂无投稿视频</div>
                  ) : (
                    <>
                      <div className='profile-video-grid'>
                        {userFiles.files.filter(f => f.mime_type?.startsWith('video/')).map(f => (
                          <div key={f.id} className='profile-video-card' onClick={() => navigate('/video/' + f.id)}>
                            <div className='profile-video-thumb'>
                              <div className='profile-video-thumb-placeholder'>
                                <Play size={28} />
                              </div>
                              <span className='profile-video-duration'>{formatDuration(f.size)}</span>
                            </div>
                            <div className='profile-video-info'>
                              <h3 className='profile-video-title' title={f.name}>{f.name}</h3>
                              <div className='profile-video-meta'>
                                <span><Clock size={12} /> {formatSize(f.size)}</span>
                                <span>{new Date(f.created_at).toLocaleDateString()}</span>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                      <Pagination page={userFiles.page} total={userFiles.total} pageSize={10} onPage={loadSubmitPage} />
                    </>
                  )}
                </>
              )}
              {submitTab === '文件' && (
                <>
                  {userFiles.files.filter(f => !f.mime_type?.startsWith('video/')).length === 0 ? (
                    <div className='profile-empty'>暂无上传文件</div>
                  ) : (
                    <>
                      <div className='profile-file-grid'>
                        {userFiles.files.filter(f => !f.mime_type?.startsWith('video/')).map(f => (
                          <div key={f.id} className='profile-file-card'>
                            <div className='profile-file-thumb'>
                              <div className='profile-file-thumb-icon'>
                                <FileText size={28} />
                              </div>
                              <span className='profile-file-type'>{f.mime_type ? f.mime_type.split('/')[1].toUpperCase() : (f.name ? f.name.split('.').pop().toUpperCase() : 'FILE')}</span>
                            </div>
                            <div className='profile-video-info'>
                              <h3 className='profile-video-title' title={f.name}>{f.name}</h3>
                              <div className='profile-video-meta'>
                                <span>{f.size ? formatSize(f.size) : ''}</span>
                                <span>{new Date(f.created_at).toLocaleDateString()}</span>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                      <Pagination page={userFiles.page} total={userFiles.total} pageSize={10} onPage={loadSubmitPage} />
                    </>
                  )}
                </>
              )}
              {submitTab === '论坛发帖' && (
                <>
                  {userPosts.posts.length === 0 ? (
                    <div className='profile-empty'>暂无论坛发帖</div>
                  ) : (
                    <>
                      {userPosts.posts.map(post => (
                        <div key={post.id} className='profile-post-card' onClick={() => navigate('/forum/' + post.board_id + '/' + post.id)}>
                          <h4>{post.title}</h4>
                          <span className='profile-post-date'>{new Date(post.created_at).toLocaleDateString()}</span>
                        </div>
                      ))}
                      <Pagination page={userPosts.page} total={userPosts.total} pageSize={10} onPage={loadPostsPage} />
                    </>
                  )}
                </>
              )}
            </div>
          </div>
        )}

        {activeTab === '收藏' && !tabLoading && (
          <div className='profile-liked'>
            {likedVideos.videos.length === 0 ? (
              <div className='profile-empty'>暂无收藏</div>
            ) : (
              <>
                <div className='profile-video-grid'>
                  {likedVideos.videos.map(v => (
                    <div key={v.id} className='profile-video-card' onClick={() => navigate('/video/' + v.id)}>
                      <div className='profile-video-thumb'>
                        <div className='profile-video-thumb-placeholder'>
                          <Play size={28} />
                        </div>
                        <span className='profile-video-duration'>{formatDuration(v.size)}</span>
                      </div>
                      <div className='profile-video-info'>
                        <h3 className='profile-video-title' title={v.name}>{v.name}</h3>
                        <div className='profile-video-meta'>
                          <span><Clock size={12} /> {formatSize(v.size)}</span>
                          <span>{new Date(v.created_at).toLocaleDateString()}</span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
                <Pagination page={likedVideos.page} total={likedVideos.total} pageSize={10} onPage={loadLikedPage} />
              </>
            )}
          </div>
        )}

        {activeTab === '设置' && isSelf && (
          <div className='profile-settings'>
            <div className='profile-settings-section'>
              <h3 className='settings-section-title'><Edit3 size={18} /> 个人信息</h3>
              {profileMsg && (
                <div className={'settings-msg' + (profileMsg.ok ? ' success' : ' error')}>
                  {profileMsg.ok ? <Check size={14} /> : <X size={14} />}
                  {profileMsg.text}
                </div>
              )}
              <div className='settings-field'>
                <label>头像</label>
                <div className='settings-avatar-row'>
                  <div className='settings-avatar-preview' onClick={() => avatarInputRef.current?.click()} style={{ cursor: 'pointer' }} title='点击更换头像'>
                    {avatarUrl ? (
                      <img src={avatarUrl} alt='' className='settings-avatar-img' />
                    ) : (
                      <Camera size={20} />
                    )}
                  </div>
                  <div className='settings-avatar-upload'>
                    <input ref={avatarInputRef} type='file' accept='image/jpeg,image/png,image/webp,image/gif' style={{ display: 'none' }} onChange={handleAvatarUpload} />
                    <span className='settings-upload-hint'>点击头像上传图片，支持 jpg, png, webp, gif，最大 5MB</span>
                  </div>
                </div>
              </div>
              <div className='settings-field'>
                <label>昵称</label>
                <input type='text' value={nickname} onChange={e => setNickname(e.target.value)} placeholder='输入昵称' className='settings-input' />
              </div>
              <div className='settings-field'>
                <label>个性签名</label>
                <textarea value={bio} onChange={e => setBio(e.target.value)} placeholder='介绍一下自己' className='settings-textarea' rows={3} />
              </div>
              <div className='settings-field'>
                <label>邮箱</label>
                <input type='email' value={email} onChange={e => setEmail(e.target.value)} placeholder='输入邮箱' className='settings-input' />
              </div>
              <button className='settings-save-btn' onClick={handleSaveProfile} disabled={savingProfile}>
                <Save size={16} /> {savingProfile ? '保存中...' : '保存信息'}
              </button>
            </div>
            <div className='profile-settings-section'>
              <h3 className='settings-section-title'><Lock size={18} /> 修改密码</h3>
              {pwdMsg && (
                <div className={'settings-msg' + (pwdMsg.ok ? ' success' : ' error')}>
                  {pwdMsg.ok ? <Check size={14} /> : <X size={14} />}
                  {pwdMsg.text}
                </div>
              )}
              <div className='settings-field'>
                <label>当前密码</label>
                <input type='password' value={oldPassword} onChange={e => setOldPassword(e.target.value)} placeholder='输入当前密码' className='settings-input' />
              </div>
              <div className='settings-field'>
                <label>新密码</label>
                <input type='password' value={newPassword} onChange={e => setNewPassword(e.target.value)} placeholder='输入新密码（至少6位）' className='settings-input' />
              </div>
              <div className='settings-field'>
                <label>确认密码</label>
                <input type='password' value={confirmPassword} onChange={e => setConfirmPassword(e.target.value)} placeholder='再次输入新密码' className='settings-input' />
              </div>
              <button className='settings-save-btn' onClick={handleSavePassword} disabled={savingPwd}>
                <Save size={16} /> {savingPwd ? '修改中...' : '修改密码'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}