import React from 'react';
import specimen from '../data/specimen.json';

interface SpecimenOp {
  kind: string;
  label: string;
  x: number;
  y: number;
  w: number;
  h: number;
  r: number;
  g: number;
  b: number;
  size: number;
  text?: string;
}

const ops = specimen.ops as SpecimenOp[];
const canvasWidth = specimen.canvasWidth;
const canvasHeight = specimen.canvasHeight;

function paintColor(op: SpecimenOp): string {
  return `rgb(${op.r} ${op.g} ${op.b})`;
}

export const Specimen: React.FC = () => {
  return (
    <figure>
      <div className="rounded-md border border-line bg-sheet p-4 sm:p-5">
        <div
          className="relative w-full overflow-hidden border border-line"
          style={{ aspectRatio: `${canvasWidth} / ${canvasHeight}`, containerType: 'inline-size' }}
        >
          {ops.map((op, index) => {
            if (op.kind === 'Text') {
              const top = ((op.y - op.size * 0.8) / canvasHeight) * 100;
              const left = (op.x / canvasWidth) * 100;
              const font = (op.size / canvasWidth) * 100;
              return (
                <span
                  key={`${op.kind}-${index}`}
                  className="absolute whitespace-nowrap font-sans"
                  style={{
                    left: `${left}%`,
                    top: `${top}%`,
                    fontSize: `${font}cqw`,
                    lineHeight: 1,
                    color: paintColor(op),
                  }}
                >
                  {op.text}
                </span>
              );
            }

            const stroked = op.kind === 'StrokeRect' || op.kind === 'Line';
            const boxHeight = op.kind === 'Line' ? Math.max(op.h, 1) : op.h;
            return (
              <span
                key={`${op.kind}-${index}`}
                className="absolute block"
                style={{
                  left: `${(op.x / canvasWidth) * 100}%`,
                  top: `${(op.y / canvasHeight) * 100}%`,
                  width: `${(op.w / canvasWidth) * 100}%`,
                  height: `${(boxHeight / canvasHeight) * 100}%`,
                  background: stroked ? 'transparent' : paintColor(op),
                  boxShadow: stroked ? `inset 0 0 0 1px ${paintColor(op)}` : undefined,
                }}
              />
            );
          })}
        </div>

        <p className="mt-4 font-mono text-[11px] uppercase tracking-wide text-faint">Order</p>
        <ol className="mt-2 space-y-1.5">
          {ops.map((op, index) => (
            <li
              key={`row-${index}`}
              className="flex items-baseline justify-between gap-3 font-mono text-xs text-ink"
            >
              <span className="flex min-w-0 items-center gap-2">
                <span
                  className="inline-block h-2.5 w-2.5 shrink-0 border border-line"
                  style={{ background: paintColor(op) }}
                  aria-hidden="true"
                />
                <span>{op.label}</span>
                <span className="text-faint">{op.kind}</span>
              </span>
              <span className="shrink-0 text-faint">
                {op.x}, {op.y}
              </span>
            </li>
          ))}
        </ol>
      </div>
      <figcaption className="mt-2 text-xs text-faint">
        Outer is the page. Black and red are the two bars. Each paint has its own origin. Positions are CSS pixels. Text Y is the baseline.
      </figcaption>
      <details className="mt-3">
        <summary className="flex min-h-11 cursor-pointer items-center text-sm text-muted">
          Document passed to html.Parse
        </summary>
        <pre className="mt-2 overflow-x-auto rounded-md bg-codebg p-4 font-mono text-xs leading-relaxed text-codetext">
          {specimen.html}
        </pre>
      </details>
    </figure>
  );
};
