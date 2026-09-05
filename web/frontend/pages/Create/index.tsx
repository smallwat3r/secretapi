import { JSX } from 'preact';
import { useState, useMemo, useEffect } from 'preact/hooks';
import { CopyableDiv } from '../../components/CopyableDiv';
import { Icon } from '../../components/Icon';
import { useCancellableFetch } from '../../hooks/useCancellableFetch';
import { ApiErrorResponse, ConfigResponse, CreateResponse, Expiry } from '../../types';
import styles from './Create.module.css';

interface CreateProps {
  config: ConfigResponse;
  path?: string;
}

const formatBytes = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`;
  return `${(bytes / 1024).toFixed(1)} KB`;
};

const formatExpiryLabel = (expiry: string): string => {
  const match = expiry.match(/^(\d+)([hd])$/);
  if (!match) return expiry;
  const [, num, unit] = match;
  const n = parseInt(num, 10);
  if (unit === 'h') return n === 1 ? '1 hour' : `${n} hours`;
  if (unit === 'd') return n === 1 ? '1 day' : `${n} days`;
  return expiry;
};

export function Create({ config }: CreateProps) {
  const [secret, setSecret] = useState<string>('');
  const [expiry, setExpiry] = useState<Expiry>('1d');
  const [result, setResult] = useState<CreateResponse | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const cancellableFetch = useCancellableFetch();

  // Clear sensitive data from memory on page hide/unload
  useEffect(() => {
    const clearSensitiveData = () => {
      setSecret('');
      setResult(null);
    };

    window.addEventListener('pagehide', clearSensitiveData);
    window.addEventListener('beforeunload', clearSensitiveData);

    return () => {
      window.removeEventListener('pagehide', clearSensitiveData);
      window.removeEventListener('beforeunload', clearSensitiveData);
      clearSensitiveData();
    };
  }, []);

  const secretByteLength = useMemo(() => new Blob([secret]).size, [secret]);
  const isSecretTooLong = secretByteLength > config.max_secret_size;
  const isSecretEmpty = secret.trim() === '';

  const handleSubmit = async (e: JSX.TargetedEvent<HTMLFormElement, Event>) => {
    e.preventDefault();
    if (loading) return;

    if (isSecretEmpty) {
      setError('Enter a secret first.');
      return;
    }
    if (isSecretTooLong) {
      setError(`The secret is over the ${formatBytes(config.max_secret_size)} limit.`);
      return;
    }

    setLoading(true);
    setResult(null);
    setError(null);

    try {
      const response = await cancellableFetch('/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ secret, expiry }),
      });

      if (response.ok) {
        const data: CreateResponse = await response.json();
        setSecret(''); // Clear secret from memory after successful submission
        setResult(data);
      } else {
        const errorData: ApiErrorResponse = await response.json();
        setError(errorData.error || 'Something went wrong.');
      }
    } catch (err: unknown) {
      if (err instanceof Error && err.name !== 'AbortError') {
        setError('Could not reach the server. Try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  if (result) {
    const expiresAt = new Date(result.expires_at).toUTCString();
    const messageTemplate = `I've shared a secret with you.

Link: ${result.read_url}
Passcode: ${result.passcode}

It expires on ${expiresAt}. You have ${config.max_read_attempts} attempts to enter the passcode, and the secret is deleted once it has been read.`;

    return (
      <div>
        <h1 class="title">Secret created</h1>
        <p class="lead">
          Send the link and the passcode to the recipient, ideally through two different channels.
        </p>
        <CopyableDiv value={result.read_url} header="Link" mono />
        <CopyableDiv value={result.passcode} header="Passcode" mono />
        <div class="field">
          <span class="label">Expires</span>
          <div class={styles.expires}>{expiresAt}</div>
        </div>
        <CopyableDiv value={messageTemplate} header="Message to send" />
        <button type="button" class="btn btnSecondary" onClick={() => setResult(null)}>
          Create another
        </button>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} novalidate>
      <h1 class="title">Share a secret</h1>
      <p class="lead">Encrypted on the server, opened once with a passcode, then deleted.</p>

      <div class="field">
        <label class="label" for="secret-input">
          Secret
        </label>
        <textarea
          id="secret-input"
          value={secret}
          onInput={(e: JSX.TargetedEvent<HTMLTextAreaElement, Event>) =>
            setSecret(e.currentTarget.value)
          }
          placeholder="Password, token, message"
          class={isSecretTooLong ? 'invalid' : ''}
          aria-invalid={isSecretTooLong}
          aria-describedby="secret-size"
        />
        <div id="secret-size" class={`hint ${styles.size} ${isSecretTooLong ? 'textDanger' : ''}`}>
          {formatBytes(secretByteLength)} of {formatBytes(config.max_secret_size)}
        </div>
      </div>

      <div class="fieldRow">
        <div class="field">
          <label class="label" for="expiry-select">
            Expires in
          </label>
          <div class="selectWrap">
            <select
              id="expiry-select"
              value={expiry}
              onChange={(e: JSX.TargetedEvent<HTMLSelectElement, Event>) =>
                setExpiry(e.currentTarget.value as Expiry)
              }
            >
              {config.expiry_options.map((opt) => (
                <option key={opt} value={opt}>
                  {formatExpiryLabel(opt)}
                </option>
              ))}
            </select>
          </div>
        </div>
        <button type="submit" class="btn" disabled={loading || isSecretTooLong}>
          {loading ? 'Creating' : 'Create secret'}
        </button>
      </div>

      {error && (
        <p class="error" role="alert">
          <Icon name="alert" />
          <span>{error}</span>
        </p>
      )}
    </form>
  );
}
