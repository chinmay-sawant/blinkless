import React, { useEffect, useState } from 'react';

const repoUrl = 'https://github.com/chinmay-sawant/blinkless';

interface GitHubLinkProps {
  className?: string;
}

export const GitHubLink: React.FC<GitHubLinkProps> = ({ className = '' }) => {
  const [stars, setStars] = useState<number | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    fetch('https://api.github.com/repos/chinmay-sawant/blinkless', { signal: controller.signal })
      .then((response) => (response.ok ? response.json() : Promise.reject(new Error('stars'))))
      .then((data: { stargazers_count?: unknown }) => {
        if (typeof data.stargazers_count === 'number') {
          setStars(data.stargazers_count);
        }
      })
      .catch(() => {
        // The star stays visible when the count cannot be loaded.
      });
    return () => controller.abort();
  }, []);

  return (
    <a
      href={repoUrl}
      target="_blank"
      rel="noopener noreferrer"
      className={`inline-flex h-11 items-center gap-2 text-sm text-muted hover:text-ink ${className}`}
    >
      <span>GitHub</span>
      <span className="inline-flex items-center gap-1 text-warn" aria-label={stars === null ? 'GitHub stars' : `${stars} GitHub stars`}>
        <svg width="14" height="14" viewBox="0 0 16 16" aria-hidden="true" className="fill-current">
          <path d="M8 1.2l1.7 3.9 4.3.4-3.2 2.9.9 4.2L8 10.5 4.3 12.6l.9-4.2L2 5.5l4.3-.4L8 1.2z" />
        </svg>
        {stars !== null ? <span className="font-mono text-xs text-ink">{stars}</span> : null}
      </span>
    </a>
  );
};
