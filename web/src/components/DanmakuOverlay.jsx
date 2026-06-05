import { useRef, useEffect, useCallback } from 'react';

const COLORS = ['#ffffff', '#e54256', '#ffe133', '#64dd17', '#18ffff', '#ff9100'];
const MIN_Y_PADDING = 30;

class DanmakuItem {
  constructor(text, time, y, color, size, type, canvasWidth) {
    this.text = text;
    this.time = time;        // time_sec from API
    this.y = y;
    this.color = color || '#ffffff';
    this.size = size || 25;
    this.type = type || 1;   // 1=scroll, 5=top, 4=bottom
    this.opacity = 1;
    this.active = true;
    this.startTime = performance.now();
    this.duration = 4000;    // ms for static types
    this.x = canvasWidth;    // start from right edge
    this.speed = 200 + Math.random() * 100; // pixels per second
  }
}

export default function DanmakuOverlay({ videoRef, danmakuEnabled, danmakuList }) {
  const canvasRef = useRef(null);
  const activeItems = useRef([]);
  const animFrame = useRef(null);
  const lastTimeRef = useRef(0);
  const dimsRef = useRef({ width: 0, height: 0, dpr: 1 });

  const ctxRef = useRef(null);

  // Initialize canvas context
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    ctxRef.current = canvas.getContext('2d');
  }, []);

  // Resize handler
  const resizeCanvas = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const parent = canvas.parentElement;
    if (!parent) return;
    const dpr = window.devicePixelRatio || 1;
    const w = parent.clientWidth;
    const h = parent.clientHeight;
    dimsRef.current = { width: w, height: h, dpr };
    canvas.width = w * dpr;
    canvas.height = h * dpr;
    canvas.style.width = w + 'px';
    canvas.style.height = h + 'px';
    const ctx = ctxRef.current;
    if (ctx) {
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    }
  }, []);

  useEffect(() => {
    resizeCanvas();
    window.addEventListener('resize', resizeCanvas);
    return () => window.removeEventListener('resize', resizeCanvas);
  }, [resizeCanvas]);

  // Sync danmaku items when the list changes
  useEffect(() => {
    const existing = activeItems.current.slice();
    const incomingIds = new Set(danmakuList.map(d => d.id));
    const existingIds = new Set(existing.map(d => d._id));
    // Remove items whose danmaku was removed from list
    activeItems.current = existing.filter(d => incomingIds.has(d._id));
  }, [danmakuList]);

  // Animation loop
  useEffect(() => {
    if (!danmakuEnabled) {
      if (animFrame.current) {
        cancelAnimationFrame(animFrame.current);
        animFrame.current = null;
      }
      // Clear canvas
      const canvas = canvasRef.current;
      if (canvas && ctxRef.current) {
        ctxRef.current.clearRect(0, 0, canvas.width, canvas.height);
      }
      return;
    }

    const ctx = ctxRef.current;
    const canvas = canvasRef.current;
    if (!ctx || !canvas) return;

    let running = true;

    function getRandomY(height) {
      const usableHeight = height - MIN_Y_PADDING * 2;
      return MIN_Y_PADDING + Math.random() * usableHeight;
    }

    function findUnusedY(height, text, size) {
      // Try to find a lane that isn't currently occupied
      const fontSize = size;
      const totalLanes = Math.max(1, Math.floor((height - MIN_Y_PADDING * 2) / (fontSize + 4)));
      const lane = Math.floor(Math.random() * totalLanes);
      return MIN_Y_PADDING + lane * (fontSize + 4);
    }

    function render(timestamp) {
      if (!running) return;
      animFrame.current = requestAnimationFrame(render);

      const { width, height, dpr } = dimsRef.current;
      if (width === 0 || height === 0) return;

      // Clear
      ctx.clearRect(0, 0, width, height);

      // Get current video time
      const videoEl = videoRef?.current;
      const currentTime = videoEl ? videoEl.currentTime : 0;

      // Add new danmaku items that are within time window
      const timeWindow = 0.5;
      const pending = danmakuList.filter(d => {
        const diff = d.time_sec - currentTime;
        return diff >= -timeWindow && diff <= timeWindow + 1 && !activeItems.current.some(a => a._id === d.id);
      });

      for (const d of pending) {
        const fontSize = d.font_size || 25;
        const color = d.color || COLORS[Math.floor(Math.random() * COLORS.length)];
        const y = findUnusedY(height, d.content, fontSize);
        const item = new DanmakuItem(d.content, d.time_sec, y, color, fontSize, d.type, width);
        item._id = d.id;
        activeItems.current.push(item);
      }

      // Update and draw active items
      const dt = lastTimeRef.current ? (timestamp - lastTimeRef.current) / 1000 : 0;
      if (dt > 0.5) { lastTimeRef.current = timestamp; return; } // Skip large gaps
      lastTimeRef.current = timestamp;

      const now = performance.now();
      const toKeep = [];

      for (const item of activeItems.current) {
        if (!item.active) continue;

        if (item.type === 1) {
          // Scroll right-to-left
          item.x -= item.speed * (dt || 0.016);

          // Fade out at left edge
          if (item.x < -200) {
            item.active = false;
            continue;
          }

          // Draw
          ctx.globalAlpha = item.opacity;
          ctx.fillStyle = item.color;
          ctx.font = `bold ${item.size}px "Microsoft YaHei", "PingFang SC", sans-serif`;
          ctx.textBaseline = 'middle';
          // Add subtle text shadow
          ctx.shadowColor = 'rgba(0,0,0,0.5)';
          ctx.shadowBlur = 2;
          ctx.fillText(item.text, item.x, item.y);
          ctx.shadowBlur = 0;

          toKeep.push(item);
        } else {
          // Static types (5=top center, 4=bottom center)
          const elapsed = now - item.startTime;
          if (elapsed > item.duration) {
            item.active = false;
            continue;
          }

          // Fade in/out
          const fadeIn = Math.min(1, elapsed / 300);
          const fadeOut = elapsed > item.duration - 500
            ? Math.max(0, (item.duration - elapsed) / 500)
            : 1;
          item.opacity = Math.min(fadeIn, fadeOut);

          if (item.opacity <= 0) continue;

          const x = width / 2; // center
          const y = item.type === 5
            ? MIN_Y_PADDING + item.size
            : height - MIN_Y_PADDING - item.size;

          ctx.globalAlpha = item.opacity;
          ctx.fillStyle = item.color;
          ctx.font = `bold ${item.size}px "Microsoft YaHei", "PingFang SC", sans-serif`;
          ctx.textAlign = 'center';
          ctx.textBaseline = 'middle';
          ctx.shadowColor = 'rgba(0,0,0,0.5)';
          ctx.shadowBlur = 2;
          ctx.fillText(item.text, x, y);
          ctx.shadowBlur = 0;
          ctx.textAlign = 'start';

          toKeep.push(item);
        }
      }

      ctx.globalAlpha = 1;
      activeItems.current = toKeep;
    }

    animFrame.current = requestAnimationFrame(render);
    return () => {
      running = false;
      if (animFrame.current) {
        cancelAnimationFrame(animFrame.current);
        animFrame.current = null;
      }
    };
  }, [danmakuEnabled, videoRef, danmakuList]);

  return (
    <canvas
      ref={canvasRef}
      className="danmaku-canvas"
    />
  );
}