import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { MessageSquare } from 'lucide-react';
import { api } from '../api/client';
import './ForumHome.css';

export default function ForumHome() {
  const [boards, setBoards] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.listBoards()
      .then((data) => setBoards(data.boards || []))
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="forum-loading">加载中...</div>;

  return (
    <div className="forum-home">
      <h2 className="section-title">
        <MessageSquare size={20} />
        论坛板块
      </h2>
      {boards.length === 0 ? (
        <p className="forum-empty">暂无板块，部署后请管理员在数据库中创建</p>
      ) : (
        <div className="board-list">
          {boards.map((board) => (
            <Link to={`/forum/${board.id}`} key={board.id} className="board-card">
              <div className="board-name">{board.name}</div>
              {board.description && <div className="board-desc">{board.description}</div>}
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
