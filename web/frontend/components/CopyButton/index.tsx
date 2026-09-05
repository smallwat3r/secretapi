import { Icon } from '../Icon';
import { useCopyToClipboard } from '../../hooks/useCopyToClipboard';

interface CopyButtonProps {
  textToCopy: string;
  label?: string;
}

export function CopyButton({ textToCopy, label = 'Copy' }: CopyButtonProps) {
  const { copied, copyToClipboard } = useCopyToClipboard(textToCopy);

  return (
    <button type="button" class="btn btnSecondary" onClick={copyToClipboard}>
      <Icon name={copied ? 'check' : 'copy'} />
      {copied ? 'Copied' : label}
    </button>
  );
}
