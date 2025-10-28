
function handleApiError(error, resultDiv) {
    console.error('API Error:', error);
    resultDiv.textContent = 'An unexpected error occurred. Please try again.';
    resultDiv.className = 'error';
}

function showCopyFeedback(element) {
    const originalText = element.textContent;
    const value = element.dataset.value || originalText;

    const oldFeedback = element.nextElementSibling;
    if (oldFeedback && oldFeedback.classList.contains('copy-feedback')) {
        oldFeedback.remove();
    }

    navigator.clipboard.writeText(value).then(() => {
        const feedback = document.createElement('div');
        feedback.classList.add('copy-feedback');
        feedback.textContent = 'Copied!';
        element.after(feedback);
        setTimeout(() => {
            feedback.remove();
        }, 2000);
    }, err => {
        console.error('Could not copy text: ', err);
    });
}

function createCopyableDiv(value, header) {
    const fragment = document.createDocumentFragment();

    if (header) {
        const p = document.createElement('p');
        p.textContent = header;
        fragment.appendChild(p);
    }

    const div = document.createElement('div');
    div.className = 'copyable';
    div.dataset.value = value;
    div.textContent = value;
    div.addEventListener('click', () => showCopyFeedback(div));
    fragment.appendChild(div);

    return fragment;
}
