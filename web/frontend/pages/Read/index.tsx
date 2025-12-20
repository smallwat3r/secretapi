import { h, JSX } from 'preact';
import { useState } from 'preact/hooks';
import styles from './Read.module.css';
import { useCancellableFetch } from '../../hooks/useCancellableFetch';
import { NewSecretButton } from '../../components/NewSecretButton';
import { CopyButton } from '../../components/CopyButton';
import { ApiErrorResponse, ReadResponse } from '../../types';

interface ReadProps {
  id: string;
}

function Read(props: ReadProps) {
  const [passcode, setPasscode] = useState<string>('');
  const [secret, setSecret] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const id = props.id;
  const cancellableFetch = useCancellableFetch();

  const handleSubmit = async (e: JSX.TargetedEvent<HTMLFormElement, Event>) => {
    e.preventDefault();
    if (loading) return;

    setLoading(true);
    setError(null);

    try {
      const response = await cancellableFetch(`/read/${id}`, {
        method: 'POST',
        headers: { 'X-Passcode': passcode },
      });

      if (response.ok) {
        const data: ReadResponse = await response.json();
        setSecret(data.secret);
      } else {
        const errorData: ApiErrorResponse = await response.json();

        if (errorData.remaining_attempts !== undefined) {
          if (errorData.remaining_attempts > 0) {
            setError(`Invalid passcode. ${errorData.remaining_attempts} attempts remaining.`);
          } else {
            setError('No attempts remaining. Secret deleted.');
          }
        } else {
          setError(errorData.error || 'An unknown error occurred.');
        }
      }
    } catch (err: any) {
      if (err?.name !== 'AbortError') {
        setError('An unexpected error occurred. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  if (secret) {
    return (
      <div class={`${styles.result} ${styles.pageWrapper}`}>
        <div className={styles.secret}>{secret}</div>
        <div class={styles.warning}>
          Leaving this page will cause the message to be lost forever. Make sure to save it somewhere safe.
        </div>
        <CopyButton textToCopy={secret} />
        <NewSecretButton />
      </div>
    );
  }

  return (
    <form class={styles.pageWrapper} onSubmit={handleSubmit}>
      <input
        value={passcode}
        onInput={(e: JSX.TargetedEvent<HTMLInputElement, Event>) => setPasscode(e.currentTarget.value)}
        placeholder="Enter passcode"
        required
        autocomplete="off"
      />
      <button type="submit" disabled={loading}>
        {loading ? 'Loading...' : 'Read Secret'}
      </button>
      {error && <div className={`${styles.errorMessage} error`}>{error}</div>}
    </form>
  );
}

export default Read;
