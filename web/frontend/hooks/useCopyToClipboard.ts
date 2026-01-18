import { useState, useCallback, useEffect, useRef } from 'preact/hooks';

const COPY_FEEDBACK_DURATION_MS = 2000;

interface UseCopyToClipboardResult {
  copied: boolean;
  copyToClipboard: () => void;
}

export function useCopyToClipboard(text: string): UseCopyToClipboardResult {
  const [copied, setCopied] = useState(false);
  const timeoutRef = useRef<number | null>(null);

  // Clear timeout on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current !== null) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  const copyToClipboard = useCallback(() => {
    // Clear any existing timeout
    if (timeoutRef.current !== null) {
      clearTimeout(timeoutRef.current);
    }

    navigator.clipboard.writeText(text).then(
      () => {
        setCopied(true);
        timeoutRef.current = window.setTimeout(() => {
          setCopied(false);
          timeoutRef.current = null;
        }, COPY_FEEDBACK_DURATION_MS);
      },
      () => {
        // Silently fail - clipboard access may be denied
      }
    );
  }, [text]);

  return { copied, copyToClipboard };
}
