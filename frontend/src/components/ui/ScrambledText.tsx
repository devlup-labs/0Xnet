import React, { useEffect, useState, useRef } from 'react';

interface ScrambledTextProps {
  text: string;
  speed?: number;
  duration?: number;
  revealDuration?: number;
  scrambleChars?: string;
  className?: string;
}

const ScrambledText: React.FC<ScrambledTextProps> = ({
  text,
  speed = 50,
  duration = 800,
  revealDuration = 400,
  scrambleChars = 'abcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()',
  className = '',
}) => {
  const [displayText, setDisplayText] = useState(text);
  const intervalRef = useRef<number | null>(null);
  const startTimeRef = useRef<number>(0);

  const startScramble = () => {
    if (intervalRef.current) clearInterval(intervalRef.current);
    startTimeRef.current = Date.now();

    intervalRef.current = window.setInterval(() => {
      const elapsed = Date.now() - startTimeRef.current;
      const progress = Math.min(elapsed / duration, 1);
      
      const scrambled = text
        .split('')
        .map((char, index) => {
          if (char === ' ') return ' ';
          const charProgress = (elapsed - (index * revealDuration / text.length)) / (duration - revealDuration);
          if (charProgress >= 1) return char;
          
          return scrambleChars[Math.floor(Math.random() * scrambleChars.length)];
        })
        .join('');

      setDisplayText(scrambled);

      if (progress >= 1) {
        clearInterval(intervalRef.current!);
        setDisplayText(text);
      }
    }, speed);
  };

  useEffect(() => {
    startScramble();
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [text]);

  return (
    <span className={className} onMouseEnter={startScramble}>
      {displayText}
    </span>
  );
};

export default ScrambledText;
