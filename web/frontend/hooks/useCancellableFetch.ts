import { useRef, useEffect } from 'preact/hooks';

export function useCancellableFetch() {
  const controllerRef = useRef<AbortController | null>(null);

  // abort any ongoing request when the component unmounts
  useEffect(() => () => controllerRef.current?.abort(), []);

  const cancellableFetch = (url: string, options: RequestInit = {}) => {
    // abort the previous request before starting a new one
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;
    return fetch(url, { ...options, signal: controller.signal });
  };

  return cancellableFetch;
}
