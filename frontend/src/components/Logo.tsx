import React from 'react';

interface LogoProps {
  className?: string;
  size?: number;
}

/** Crop mark: four corner ticks, a square, and a filled center. */
export const Logo: React.FC<LogoProps> = ({ className = 'w-6 h-6', size = 24 }) => {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-hidden="true"
    >
      <path
        d="M4 9V4H9M23 4H28V9M28 23V28H23M9 28H4V23"
        stroke="currentColor"
        strokeWidth="1.5"
      />
      <rect x="11" y="11" width="10" height="10" stroke="currentColor" strokeWidth="1.5" />
      <rect x="14.5" y="14.5" width="3" height="3" fill="currentColor" />
    </svg>
  );
};
