import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import FileExplorer from './pages/FileExplorer';
import SharePage from './pages/SharePage';
import ForumHome from './pages/ForumHome';
import PostListPage from './pages/PostListPage';
import PostDetailPage from './pages/PostDetailPage';
import VideoPlayerPage from './pages/VideoPlayerPage';
import AppLayout from './components/AppLayout';

function ProtectedRoute({ children }) {
  const { user } = useAuth();
  if (!user) return <Navigate to="/login" replace />;
  return children;
}

function PublicRoute({ children }) {
  const { user } = useAuth();
  if (user) return <Navigate to="/files" replace />;
  return children;
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<PublicRoute><LoginPage /></PublicRoute>} />
          <Route path="/register" element={<PublicRoute><RegisterPage /></PublicRoute>} />
          <Route path="/share/:token" element={<SharePage />} />
          <Route path="/" element={<ProtectedRoute><AppLayout /></ProtectedRoute>}>
            <Route index element={<Navigate to="/files" replace />} />
            <Route path="files" element={<FileExplorer />} />
            <Route path="files/:folderId" element={<FileExplorer />} />
            <Route path="video/:id" element={<VideoPlayerPage />} />
            <Route path="forum" element={<ForumHome />} />
            <Route path="forum/:boardId" element={<PostListPage />} />
            <Route path="forum/:boardId/:postId" element={<PostDetailPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/files" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}
