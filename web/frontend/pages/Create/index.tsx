import { h, JSX } from 'preact';
import { useState } from 'preact/hooks';
import CopyableDiv from '../../components/CopyableDiv';
import styles from './Create.module.css';
import { useCancellableFetch } from '../../hooks/useCancellableFetch';

type Expiry = '1h' | '6h' | '1d' | '3d';

interface CreateResponse {
  read_url: string;
  passcode: string;
  expires_at: string;
}

function Create() {
  const [secret, setSecret] = useState<string>('');
  const [expiry, setExpiry] = useState<Expiry>('1d');
  const [result, setResult] = useState<CreateResponse | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const cancellableFetch = useCancellableFetch();

  const handleSubmit = async (e: JSX.TargetedEvent<HTMLFormElement, Event>) => {
    e.preventDefault();
    if (loading) return;

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
        setResult(data);
      } else {
        const errorData: { error?: string } = await response.json();
        setError(errorData.error || 'An unknown error occurred.');
      }
    } catch (err: any) {
      if (err?.name !== 'AbortError') {
        setError('An unexpected error occurred. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  if (result) {
    return (
      <div class={`${styles.result} ${styles.pageWrapper}`}>
        <p class={styles.resultInfo}>
          Share the Read URL with the recipient. They will need the <strong>passcode</strong> to view the secret.
        </p>
        <CopyableDiv value={result.read_url} header="Read URL" />
        <CopyableDiv value={result.passcode} header="Passcode" />
        <p>Expires At</p>
        <div>{new Date(result.expires_at).toUTCString()}</div>
      </div>
    );
  }

  return (
    <form class={`${styles.form} ${styles.pageWrapper}`} onSubmit={handleSubmit}>
      <textarea
        value={secret}
        onInput={(e: JSX.TargetedEvent<HTMLTextAreaElement, Event>) => setSecret(e.currentTarget.value)}
        placeholder="Enter your secret"
        required
      />
      <p class={styles.expiryLabel}>Expiry</p>
      <select
        value={expiry}
        onChange={(e: JSX.TargetedEvent<HTMLSelectElement, Event>) =>
          setExpiry(e.currentTarget.value as Expiry)
        }
      >
        <option value="1h">1 hour</option>
        <option value="6h">6 hours</option>
        <option value="1d">1 day</option>
        <option value="3d">3 days</option>
      </select>
      <button type="submit" disabled={loading}>
        {loading ? 'Loading...' : 'Create Secret'}
      </button>
      {error && <div className="error">{error}</div>}
    </form>
  );
}

export default Create;
