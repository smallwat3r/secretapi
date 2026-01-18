import { useState, useEffect } from 'preact/hooks';
import { ConfigResponse } from '../types';

const DEFAULT_CONFIG: ConfigResponse = {
  max_secret_size: 64 * 1024,
  expiry_options: ['1h', '6h', '1d', '3d'],
};

export function useConfig(): ConfigResponse {
  const [config, setConfig] = useState<ConfigResponse>(DEFAULT_CONFIG);

  useEffect(() => {
    fetch('/config')
      .then((res) => res.json())
      .then((data: ConfigResponse) => setConfig(data))
      .catch(() => {
        // Use defaults on error
      });
  }, []);

  return config;
}
