document.addEventListener('DOMContentLoaded', () => {
    const filenameSpan = document.getElementById('filename');
    const filesizeSpan = document.getElementById('filesize');
    const downloadPasswordInput = document.getElementById('downloadPassword');
    const downloadBtn = document.getElementById('downloadBtn');

    const urlParams = new URLSearchParams(window.location.search);
    const fileToken = urlParams.get('file');

    if (!fileToken) {
        document.body.innerHTML = '<h1>无效的分享链接</h1>';
        return;
    }

    async function fetchFileInfo() {
        try {
            const response = await fetch(`/api/share/info?file=${fileToken}`);
            if (!response.ok) {
                throw new Error('文件未找到或链接已失效');
            }
            const info = await response.json();
            filenameSpan.textContent = info.filename;
            filesizeSpan.textContent = (info.filesize / 1024 / 1024).toFixed(2) + ' MB';
        } catch (error) {
            document.body.innerHTML = `<h1>${error.message}</h1>`;
        }
    }

    downloadBtn.addEventListener('click', () => {
        const password = downloadPasswordInput.value;
        let downloadUrl = `/api/share/download?file=${fileToken}`;
        if (password) {
            downloadUrl += `&password=${encodeURIComponent(password)}`;
        }
        window.location.href = downloadUrl;
    });

    fetchFileInfo();
});