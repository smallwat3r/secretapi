import styles from './About.module.css';

export function About(_props: { path?: string }) {
  return (
    <div class={styles.page}>
      <h1 class="title">About</h1>
      <p class="lead">
        A small, self-hostable service for passing a password, a token or a short message to
        someone, once.
      </p>

      <section class={styles.section}>
        <h2>Why not just paste it in chat?</h2>
        <p>
          Anything sent through chat or email lives on in logs, search indexes, backups and third
          party systems. A secret shared here is stored encrypted, is only readable once, and is
          gone after that or when it expires.
        </p>
      </section>

      <section class={styles.section}>
        <h2>How it works</h2>
        <ol>
          <li>Paste the secret and pick how long it should live.</li>
          <li>Send the link and the passcode to the recipient, ideally separately.</li>
          <li>They enter the passcode and read the secret.</li>
          <li>
            The secret is deleted after being read, after three wrong passcodes, or on expiry.
          </li>
        </ol>
      </section>

      <section class={styles.section}>
        <h2>Security</h2>
        <ul>
          <li>AES-256-GCM encryption with an Argon2id derived key.</li>
          <li>The passcode is never stored, so the server cannot decrypt what it holds.</li>
          <li>Three wrong attempts delete the secret.</li>
          <li>No accounts, no tracking, no analytics.</li>
        </ul>
      </section>

      <section class={styles.section}>
        <h2>A habit worth keeping</h2>
        <p>
          Put only the value itself in the secret. Say what it is for when you send the link, not
          inside it. If the secret is ever intercepted, it is useless without that context.
        </p>
      </section>

      <section class={styles.section}>
        <h2>Run your own</h2>
        <p>
          The project is open source and ships as a small Docker image. See the{' '}
          <a href="https://github.com/smallwat3r/secretapi">repository</a> for setup.
        </p>
      </section>
    </div>
  );
}
