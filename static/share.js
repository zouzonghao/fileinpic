document.addEventListener('DOMContentLoaded', () => {
    const filenameSpan = document.getElementById('filename');
    const filesizeSpan = document.getElementById('filesize');
    const downloadPasswordInput = document.getElementById('downloadPassword');
    const downloadBtn = document.getElementById('downloadBtn');
    const errorMessage = document.getElementById('errorMessage');
    const toastContainer = document.getElementById('toastContainer');

    const urlParams = new URLSearchParams(window.location.search);
    const fileToken = urlParams.get('file');

    if (!fileToken) {
        document.body.innerHTML = '<h1>无效的分享链接</h1>';
        return;
    }

    function showToast(message, type = 'success') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        toastContainer.appendChild(toast);

        setTimeout(() => {
            toast.remove();
        }, 3000); // Toast disappears after 3 seconds
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
            downloadBtn.dataset.filename = info.filename;
        } catch (error) {
            document.body.innerHTML = `<h1>${error.message}</h1>`;
        }
    }

    downloadBtn.addEventListener('click', async () => {
        const password = downloadPasswordInput.value;
        let downloadUrl = `/api/share/download?file=${fileToken}`;
        if (password) {
            downloadUrl += `&password=${encodeURIComponent(password)}`;
        }

        errorMessage.textContent = '';
        downloadBtn.textContent = '下载中...';
        downloadBtn.disabled = true;

        try {
            const response = await fetch(downloadUrl);

            if (!response.ok) {
                let errorText = await response.text();
                if (errorText.trim() === 'Invalid password') {
                    errorText = '密码无效';
                }
                throw new Error(errorText || '下载失败');
            }

            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.style.display = 'none';
            a.href = url;
            a.download = downloadBtn.dataset.filename || 'download';
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);
            showToast('下载成功！');

        } catch (error) {
            showToast(error.message, 'error');
        } finally {
            downloadBtn.textContent = '下载';
            downloadBtn.disabled = false;
        }
    });

    fetchFileInfo();
});