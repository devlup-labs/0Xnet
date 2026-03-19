import React, { useState, useRef, useEffect } from 'react';
import { motion } from 'framer-motion';

interface PillNavProps {
  items: { id: string; label: string; icon?: string }[];
  activeId: string;
  onChange: (id: string) => void;
  className?: string;
}

const PillNav: React.FC<PillNavProps> = ({ items, activeId, onChange, className = '' }) => {
  const [hoveredId, setHoveredId] = useState<string | null>(null);

  return (
    <nav className={`pill-nav-container ${className}`} style={{
      display: 'flex',
      gap: '8px',
      padding: '6px',
      background: 'rgba(255, 255, 255, 0.05)',
      borderRadius: '999px',
      position: 'relative',
      backdropFilter: 'blur(10px)',
      border: '1px solid rgba(255,255,255,0.08)'
    }}>
      {items.map((item) => (
        <button
          key={item.id}
          onClick={() => onChange(item.id)}
          onMouseEnter={() => setHoveredId(item.id)}
          onMouseLeave={() => setHoveredId(null)}
          style={{
            position: 'relative',
            padding: '8px 18px',
            fontSize: '0.85rem',
            fontWeight: 700,
            color: activeId === item.id || hoveredId === item.id ? 'white' : 'rgba(255,255,255,0.6)',
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            transition: 'color 0.3s ease',
            zIndex: 1,
            display: 'flex',
            alignItems: 'center',
            gap: '8px'
          }}
        >
          {item.icon && <span>{item.icon}</span>}
          {item.label}
          
          {activeId === item.id && (
            <motion.div
              layoutId="pill-active"
              className="active-pill"
              style={{
                position: 'absolute',
                inset: 0,
                background: 'rgba(168, 85, 247, 0.8)',
                borderRadius: '999px',
                zIndex: -1,
                boxShadow: '0 4px 15px rgba(168, 85, 247, 0.4)'
              }}
              transition={{ type: 'spring', duration: 0.6 }}
            />
          )}

          {hoveredId === item.id && activeId !== item.id && (
            <motion.div
              layoutId="pill-hover"
              className="hover-pill"
              style={{
                position: 'absolute',
                inset: 0,
                background: 'rgba(255, 255, 255, 0.1)',
                borderRadius: '999px',
                zIndex: -1
              }}
              transition={{ type: 'spring', duration: 0.4 }}
            />
          )}
        </button>
      ))}
    </nav>
  );
};

export default PillNav;
