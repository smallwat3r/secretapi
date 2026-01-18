import { h } from 'preact';
import styles from './About.module.css';

export function About() {
  return (
    <div class={styles.pageWrapper}>
      <h1 class={styles.title}>About</h1>

      <section class={styles.section}>
        <h2>What is SecretAPI?</h2>
        <p>
          SecretAPI is a simple, secure way to share sensitive information. Create a secret, share
          the link and passcode with your recipient, and the secret is automatically deleted after
          it's read or when it expires.
        </p>
      </section>

      <section class={styles.section}>
        <h2>How it works</h2>
        <ul>
          <li>Enter your secret and choose an expiry time</li>
          <li>Share the generated URL and passcode with your recipient</li>
          <li>The recipient enters the passcode to view the secret</li>
          <li>The secret is permanently deleted after being read</li>
        </ul>
      </section>

      <section class={styles.section}>
        <h2>Security</h2>
        <ul>
          <li>Secrets are encrypted using AES-256-GCM with Argon2id key derivation</li>
          <li>The passcode never leaves your device unencrypted</li>
          <li>Secrets are stored only temporarily and deleted after reading</li>
          <li>Limited read attempts protect against brute-force attacks</li>
          <li>No accounts, no tracking, no data collection</li>
        </ul>
      </section>

      <section class={styles.section}>
        <h2>Self-hosting</h2>
        <p>
          For maximum security, you can deploy your own SecretAPI server. The project is open source
          and easy to self-host with Docker. Visit the{' '}
          <a href="https://github.com/smallwat3r/secretapi" class={styles.link}>
            GitHub repository
          </a>{' '}
          for instructions.
        </p>
      </section>

      <a href="/" class={styles.backLink}>
        Back to home
      </a>
    </div>
  );
}
