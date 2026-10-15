import React, { useState } from 'react';
import { Check, Copy, Terminal } from 'lucide-react';

interface CodeBlockProps {
  code: string;
  filename?: string;
  language?: 'go' | 'bash';
}

/**
 * Highlights Go syntax tokens using semantic flat palette:
 * - Comments: text-zinc-500 italic
 * - Strings: text-emerald-400
 * - Keywords: text-purple-400 font-semibold
 * - Types: text-sky-300
 * - Functions: text-blue-400
 * - Numbers: text-amber-300
 */
function highlightGoLine(line: string): React.ReactNode {
  // Check full line comment
  const commentMatch = line.match(/^(\s*)(\/\/.*)$/);
  if (commentMatch) {
    return (
      <>
        <span>{commentMatch[1]}</span>
        <span className="text-zinc-500 italic">{commentMatch[2]}</span>
      </>
    );
  }

  // Regex tokenizer for Go code
  const tokenRegex = /(\/\/.*$)|("(?:\\.|[^"\\])*"|`[^`]*`)|(\b(?:package|import|func|return|if|else|for|range|switch|case|default|go|defer|type|struct|interface|nil|true|false|var|const)\b)|(\b(?:context\.Context|string|error|int|int64|float64|bool|byte|\[\]byte|\*html\.Document|\*css\.Document|\*layout\.Display|css\.Options|layout\.Options|layout\.DisplayOp|layout\.DisplayOpFillRect|layout\.DisplayOpText|layout\.DisplayOpImage|layout\.DisplayOpLine)\b)|(\b(?:Parse|Apply|DisplayList|DisplayListOptions|Background|Printf|Println|panic|make|len|append)\b)|(\b\d+(?:\.\d+)?\b)|([^\s"'`\w]+|\s+|\w+)/g;

  const elements: React.ReactNode[] = [];
  let match: RegExpExecArray | null;
  let key = 0;

  while ((match = tokenRegex.exec(line)) !== null) {
    const [raw, comment, str, keyword, typeName, funcName, num] = match;

    if (comment) {
      elements.push(<span key={key++} className="text-zinc-500 italic">{comment}</span>);
    } else if (str) {
      elements.push(<span key={key++} className="text-emerald-400">{str}</span>);
    } else if (keyword) {
      elements.push(<span key={key++} className="text-purple-400 font-semibold">{keyword}</span>);
    } else if (typeName) {
      elements.push(<span key={key++} className="text-sky-300">{typeName}</span>);
    } else if (funcName) {
      elements.push(<span key={key++} className="text-blue-400">{funcName}</span>);
    } else if (num) {
      elements.push(<span key={key++} className="text-amber-300">{num}</span>);
    } else {
      elements.push(<span key={key++} className="text-zinc-200">{raw}</span>);
    }
  }

  return elements.length > 0 ? elements : line;
}

export const CodeBlock: React.FC<CodeBlockProps> = ({ code, filename = 'main.go', language = 'go' }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const lines = code.trim().split('\n');

  return (
    <div className="rounded border border-zinc-800 bg-zinc-950 overflow-hidden text-xs font-mono shadow-sm">
      {/* Header Bar */}
      <div className="bg-zinc-900/90 px-4 py-2 border-b border-zinc-800 flex items-center justify-between text-zinc-400">
        <div className="flex items-center gap-2">
          <Terminal className="w-3.5 h-3.5 text-zinc-400" />
          <span className="text-zinc-300 font-medium">{filename}</span>
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-400 font-normal border border-zinc-700/60 uppercase">
            {language}
          </span>
        </div>
        <button
          onClick={handleCopy}
          className="flex items-center gap-1.5 px-2 py-1 rounded hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition-colors"
          title="Copy code snippet"
        >
          {copied ? (
            <>
              <Check className="w-3.5 h-3.5 text-emerald-400" />
              <span className="text-[11px] text-emerald-400 font-sans">Copied</span>
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              <span className="text-[11px] font-sans">Copy</span>
            </>
          )}
        </button>
      </div>

      {/* Code Area */}
      <div className="p-4 sm:p-5 overflow-x-auto leading-relaxed no-scrollbar bg-zinc-950">
        <pre className="font-mono text-xs">
          {lines.map((line, idx) => (
            <div key={idx} className="table-row">
              <span className="table-cell select-none text-zinc-600 pr-4 text-right w-6">
                {idx + 1}
              </span>
              <span className="table-cell whitespace-pre">
                {language === 'go' ? highlightGoLine(line) : (
                  <span className="text-zinc-300">
                    {line.startsWith('$') ? (
                      <>
                        <span className="text-zinc-500 mr-2">$</span>
                        <span className="text-emerald-400">{line.slice(1).trim()}</span>
                      </>
                    ) : (
                      line
                    )}
                  </span>
                )}
              </span>
            </div>
          ))}
        </pre>
      </div>
    </div>
  );
};
