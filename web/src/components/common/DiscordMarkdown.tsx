import React, { useState } from "react";

interface DiscordMarkdownProps {
  content: string;
  className?: string;
}

export const SpoilerSpan: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [revealed, setRevealed] = useState(false);

  return (
    <span
      role="button"
      tabIndex={0}
      title={revealed ? "Click to hide spoiler" : "Click to reveal spoiler"}
      onClick={(e) => {
        e.stopPropagation();
        setRevealed(!revealed);
      }}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          setRevealed(!revealed);
        }
      }}
      className={`inline-block rounded-md px-1.5 py-0.5 transition-all duration-200 cursor-pointer font-normal ${
        revealed
          ? "bg-base-300 text-base-content border border-base-content/15 shadow-xs"
          : "bg-neutral-800 text-transparent hover:bg-neutral-700 select-none shadow-xs border border-neutral-700"
      }`}
    >
      <span
        className={
          revealed ? "opacity-100" : "opacity-0 select-none pointer-events-none"
        }
      >
        {children}
      </span>
    </span>
  );
};

export const DiscordMarkdown: React.FC<DiscordMarkdownProps> = ({
  content,
  className = "",
}) => {
  if (!content) return null;

  // Render text with Discord-like formatting:
  // - Spoilers: ||text||
  // - Bold Italic: ***text***
  // - Bold: **text**
  // - Underline: __text__
  // - Italic: *text* or _text_
  // - Strikethrough: ~~text~~
  // - Inline Code: `code`
  // - Block Quotes: > text
  // - Links: https://...
  const renderInline = (text: string): React.ReactNode[] => {
    const tokenRegex =
      /(\|\|.+?\|\||\*\*\*.+?\*\*\*|\*\*.+?\*\*|__.+?__|~~.+?~~|`[^`]+`|\*[^*]+?\*|_[^_]+?_|https?:\/\/[^\s<]+)/g;

    const parts: React.ReactNode[] = [];
    let lastIndex = 0;
    let match: RegExpExecArray | null;

    while ((match = tokenRegex.exec(text)) !== null) {
      if (match.index > lastIndex) {
        parts.push(text.substring(lastIndex, match.index));
      }

      const matchText = match[0];

      if (
        matchText.startsWith("||") &&
        matchText.endsWith("||") &&
        matchText.length >= 4
      ) {
        const inner = matchText.slice(2, -2);
        parts.push(
          <SpoilerSpan key={`spoiler-${match.index}`}>
            {renderInline(inner)}
          </SpoilerSpan>,
        );
      } else if (
        matchText.startsWith("***") &&
        matchText.endsWith("***") &&
        matchText.length >= 6
      ) {
        const inner = matchText.slice(3, -3);
        parts.push(
          <strong key={`bi-${match.index}`}>
            <em>{renderInline(inner)}</em>
          </strong>,
        );
      } else if (
        matchText.startsWith("**") &&
        matchText.endsWith("**") &&
        matchText.length >= 4
      ) {
        const inner = matchText.slice(2, -2);
        parts.push(
          <strong
            key={`b-${match.index}`}
            className="font-bold text-base-content"
          >
            {renderInline(inner)}
          </strong>,
        );
      } else if (
        matchText.startsWith("__") &&
        matchText.endsWith("__") &&
        matchText.length >= 4
      ) {
        const inner = matchText.slice(2, -2);
        parts.push(
          <u key={`u-${match.index}`} className="underline underline-offset-2">
            {renderInline(inner)}
          </u>,
        );
      } else if (
        matchText.startsWith("~~") &&
        matchText.endsWith("~~") &&
        matchText.length >= 4
      ) {
        const inner = matchText.slice(2, -2);
        parts.push(
          <del key={`del-${match.index}`} className="line-through opacity-70">
            {renderInline(inner)}
          </del>,
        );
      } else if (
        matchText.startsWith("`") &&
        matchText.endsWith("`") &&
        matchText.length >= 2
      ) {
        const inner = matchText.slice(1, -1);
        parts.push(
          <code
            key={`code-${match.index}`}
            className="rounded bg-base-200 px-1.5 py-0.5 font-mono text-xs text-primary border border-base-300"
          >
            {inner}
          </code>,
        );
      } else if (
        (matchText.startsWith("*") &&
          matchText.endsWith("*") &&
          matchText.length >= 2) ||
        (matchText.startsWith("_") &&
          matchText.endsWith("_") &&
          matchText.length >= 2)
      ) {
        const inner = matchText.slice(1, -1);
        parts.push(
          <em key={`em-${match.index}`} className="italic">
            {renderInline(inner)}
          </em>,
        );
      } else if (
        matchText.startsWith("http://") ||
        matchText.startsWith("https://")
      ) {
        parts.push(
          <a
            key={`link-${match.index}`}
            href={matchText}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary underline hover:opacity-80 transition-opacity break-all"
            onClick={(e) => e.stopPropagation()}
          >
            {matchText}
          </a>,
        );
      } else {
        parts.push(matchText);
      }

      lastIndex = tokenRegex.lastIndex;
    }

    if (lastIndex < text.length) {
      parts.push(text.substring(lastIndex));
    }

    return parts;
  };

  const lines = content.split("\n");

  return (
    <div className={`space-y-1.5 leading-relaxed wrap-break-word ${className}`}>
      {lines.map((line, idx) => {
        const trimmed = line.trimStart();
        if (trimmed.startsWith("> ")) {
          const quoteText = trimmed.slice(2);
          return (
            <blockquote
              key={idx}
              className="border-l-4 border-primary/40 pl-3 py-0.5 my-1 italic text-base-content/80 bg-base-200/20 rounded-r"
            >
              {renderInline(quoteText)}
            </blockquote>
          );
        }

        if (line.trim() === "") {
          return <div key={idx} className="h-2" />;
        }

        return (
          <p key={idx} className="my-0.5">
            {renderInline(line)}
          </p>
        );
      })}
    </div>
  );
};
