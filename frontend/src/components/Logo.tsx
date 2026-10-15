import React from 'react';

interface LogoProps {
  className?: string;
  size?: number;
}

/**
 * Blinkless Logo:
 * A distinctive geometric "unblinking eye" integrating HTML/CSS layout box brackets
 * and a centered render aperture. Symbolizes pure layout computation without script flicker.
 */
export const Logo: React.FC<LogoProps> = ({ className = 'w-6 h-6', size = 24 }) => {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      {/* Outer stylized unblinking contour */}
      <path
        d="M2 16C5.5 8.5 10.5 5 16 5C21.5 5 26.5 8.5 30 16C26.5 23.5 21.5 27 16 27C10.5 27 5.5 23.5 2 16Z"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {/* Layout box model center (square pupil & bounding box) */}
      <rect
        x="11"
        y="11"
        width="10"
        height="10"
        rx="1.5"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {/* Focal core point */}
      <circle cx="16" cy="16" r="2" fill="currentColor" />
      {/* Corner alignment ticks representing layout anchors */}
      <path d="M11 8V11H8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M21 8V11H24" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M11 24V21H8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M21 24V21H24" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  );
};
