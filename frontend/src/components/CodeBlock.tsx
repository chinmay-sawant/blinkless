import React, { useState } from 'react';

interface CodeBlockProps {
  code: string;
  filename?: string;
  language?: 'go' | 'bash';
}

/**
 * Colors one Go source line.
 * Comments, strings, keywords, types, functions, and numbers use the code palette.
 */
function highlightGoLine(line: string): React.ReactNode {
  const commentMatch = line.match(/^(\s*)(\/\/.*)$/);
  if (commentMatch) {
    return (
      <>
        <span>{commentMatch[1]}</span>
        <span className="text-codecomment italic">{commentMatch[2]}</span>
      </>
    );
  }

  const tokenRegex = /(\/\/.*$)|("(?:\\.|[^"\\])*"|`[^`]*`)|(\b(?:package|import|func|return|if|else|for|range|switch|case|default|go|defer|type|struct|interface|nil|true|false|var|const)\b)|(\b(?:context\.Context|string|error|int|int64|float64|bool|byte|\[\]byte|\*html\.Document|\*css\.Document|\*layout\.Display|css\.Options|layout\.Options|layout\.DisplayOp|layout\.DisplayOpFillRect|layout\.DisplayOpText|layout\.DisplayOpImage|layout\.DisplayOpLine)\b)|(\b(?:Parse|Apply|DisplayList|DisplayListOptions|Background|Printf|Println|panic|make|len|append)\b)|(\b\d+(?:\.\d+)?\b)|([^\s"'`\w]+|\s+|\w+)/g;

  const elements: React.ReactNode[] = [];
  let match: RegExpExecArray | null;
  let key = 0;

  while ((match = tokenRegex.exec(line)) !== null) {
    const [raw, comment, str, keyword, typeName, funcName, num] = match;

    if (comment) {
      elements.push(<span key={key++} className="text-codecomment italic">{comment}</span>);
    } else if (str) {
      elements.push(<span key={key++} className="text-codestring">{str}</span>);
    } else if (keyword) {
      elements.push(<span key={key++} className="text-codekeyword">{keyword}</span>);
    } else if (typeName) {
      elements.push(<span key={key++} className="text-codetext">{typeName}</span>);
    } else if (funcName) {
      elements.push(<span key={key++} className="text-codetext">{funcName}</span>);
    } else if (num) {
      elements.push(<span key={key++} className="text-codestring">{num}</span>);
    } else {
      elements.push(<span key={key++} className="text-codetext">{raw}</span>);
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
    <div className="rounded-md border border-line bg-codebg overflow-hidden font-mono text-[13px]">
      <div className="bg-codebar px-4 py-2 flex items-center justify-between">
        <div className="flex items-baseline gap-3 min-w-0">
          <span className="text-codetext truncate">{filename}</span>
          <span className="text-codecomment">{language}</span>
        </div>
        <button
          type="button"
          onClick={handleCopy}
          className="inline-flex items-center justify-center min-h-11 px-3 text-codecomment hover:text-codetext cursor-pointer"
        >
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>

      <div className="px-4 py-3 overflow-x-auto leading-relaxed">
        <pre className="font-mono text-[13px]">
          {lines.map((line, idx) => (
            <div key={idx} className="table-row">
              <span className="table-cell select-none text-codecomment pr-4 text-right w-6">
                {idx + 1}
              </span>
              <span className="table-cell whitespace-pre">
                {language === 'go' ? highlightGoLine(line) : (
                  line.startsWith('$') ? (
                    <>
                      <span className="text-codecomment">$</span>
                      <span className="text-codestring">{line.slice(1)}</span>
                    </>
                  ) : (
                    <span className="text-codetext">{line}</span>
                  )
                )}
              </span>
            </div>
          ))}
        </pre>
      </div>
    </div>
  );
};
