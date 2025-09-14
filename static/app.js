document.addEventListener('DOMContentLoaded', () => {
    // --- DOM Elements ---
    const fileListBody = document.querySelector('#fileList tbody');
    const searchInput = document.getElementById('searchInput');
    const toastContainer = document.getElementById('toastContainer');
    const loadingOverlay = document.getElementById('loadingOverlay');

    // Upload Modal Elements
    const uploadModal = document.getElementById('uploadModal');
    const showUploadModalBtn = document.getElementById('showUploadModalBtn');
    const closeUploadModalBtn = document.getElementById('closeUploadModalBtn');
    const fileInput = document.getElementById('fileInput');
    const uploadButton = document.getElementById('uploadButton');
    const authTokenInput = document.getElementById('authToken');
    const uploadStatus = document.getElementById('uploadStatus');
    const fileNameSpan = document.querySelector('.file-name');

    // Delete Modal Elements
    const deleteModal = document.getElementById('deleteModal');
    const deleteFilenameSpan = document.getElementById('deleteFilename');
    const deleteAuthTokenInput = document.getElementById('deleteAuthToken');
    const cancelDeleteBtn = document.getElementById('cancelDeleteBtn');
    const confirmDeleteBtn = document.getElementById('confirmDeleteBtn');

    let searchTimeout;
    let fileToDelete = { id: null, filename: null };

    // --- Toast Notification ---
    function showToast(message, type = 'success') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        toastContainer.appendChild(toast);

        setTimeout(() => {
            toast.remove();
        }, 3000); // Toast disappears after 3 seconds
    }

    // --- Modal Logic (General) ---
    function showModal(modal) {
        modal.classList.remove('hidden');
        document.body.classList.add('modal-open');
    }

    function hideModal(modal) {
        modal.classList.add('hidden');
        document.body.classList.remove('modal-open');
    }

    // --- Loading Overlay ---
    function showLoading() {
        loadingOverlay.classList.remove('hidden');
    }

    function hideLoading() {
        loadingOverlay.classList.add('hidden');
    }

    // --- Upload Modal ---
    showUploadModalBtn.addEventListener('click', () => showModal(uploadModal));
    closeUploadModalBtn.addEventListener('click', () => hideModal(uploadModal));
    uploadModal.addEventListener('click', (e) => {
        if (e.target === uploadModal) hideModal(uploadModal);
    });
    fileInput.addEventListener('change', () => {
        if (fileInput.files.length > 0) {
            fileNameSpan.textContent = fileInput.files[0].name;
        } else {
            fileNameSpan.textContent = '未选择任何文件';
        }
    });

    // --- Delete Modal ---
    function showDeleteModal(fileId, filename) {
        fileToDelete = { id: fileId, filename: filename };
        deleteFilenameSpan.textContent = filename;
        deleteAuthTokenInput.value = authTokenInput.value; // Pre-fill token
        showModal(deleteModal);
    }

    cancelDeleteBtn.addEventListener('click', () => hideModal(deleteModal));
    deleteModal.addEventListener('click', (e) => {
        if (e.target === deleteModal) hideModal(deleteModal);
    });
    confirmDeleteBtn.addEventListener('click', () => {
        handleConfirmDelete();
    });


    // --- API & Data Fetching ---
    async function fetchConfig() {
        try {
            const response = await fetch('/api/config');
            if (!response.ok) throw new Error('无法获取应用配置');
            const config = await response.json();
            window.appConfig = config; // Store config globally
            if (config.authToken) {
                authTokenInput.value = config.authToken;
            }
        } catch (error) {
            console.error('获取配置时出错:', error);
        }
    }

    async function fetchFiles(searchTerm = '') {
        try {
            const url = searchTerm ? `/api/files?search=${encodeURIComponent(searchTerm)}` : '/api/files';
            const response = await fetch(url);
            if (!response.ok) throw new Error('无法获取文件列表');
            
            const files = await response.json();
            renderFileList(files);
        } catch (error) {
            console.error('获取文件时出错:', error);
            fileListBody.innerHTML = `<tr><td colspan="4" style="text-align: center;">加载文件失败。</td></tr>`;
        }
    }

    // --- UI Rendering ---
    function renderFileList(files) {
        fileListBody.innerHTML = '';
        if (files && files.length > 0) {
            files.forEach(file => {
                const row = document.createElement('tr');
                const uploadDate = new Date(file.upload_timestamp).toLocaleString('zh-CN');
                const fileSize = (file.filesize / 1024 / 1024).toFixed(2) + ' MB';

                row.innerHTML = `
                    <td>${file.filename}</td>
                    <td>${fileSize}</td>
                    <td>${uploadDate}</td>
                    <td class="actions">
                        <button class="download-btn" onclick="downloadFile(${file.id})">下载</button>
                        <button class="share-btn" onclick="shareFile(${file.id}, '${file.filename}')">分享</button>
                        <button class="delete-btn" onclick="deleteFile(${file.id}, '${file.filename}')">删除</button>
                    </td>
                `;
                fileListBody.appendChild(row);
            });
        } else {
            fileListBody.innerHTML = `<tr><td colspan="4" style="text-align: center;">没有找到文件。</td></tr>`;
        }
    }

    // --- Event Handlers ---
    async function uploadFile() {
        const file = fileInput.files[0];
        const authToken = authTokenInput.value.trim();

        if (!file || !authToken) {
            showToast('请选择一个文件并提供授权令牌。', 'error');
            return;
        }

        showLoading();
        uploadButton.disabled = true;
        uploadStatus.textContent = ''; // Clear previous status

        const formData = new FormData();
        formData.append('image', file);

        try {
            const response = await fetch('/api/upload', {
                method: 'POST',
                headers: { 'Auth-Token': authToken },
                body: formData
            });

            const result = await response.json();
            if (!response.ok || !result.ok) throw new Error(result.message || '上传失败');

            fileInput.value = '';
            fileNameSpan.textContent = '未选择任何文件';
            hideModal(uploadModal);
            fetchFiles();
            showToast('文件上传成功！');

        } catch (error) {
            console.error('上传错误:', error);
            showToast(`上传失败: ${error.message}`, 'error');
        } finally {
            hideLoading();
            uploadButton.disabled = false;
        }
    }

    searchInput.addEventListener('input', (e) => {
        clearTimeout(searchTimeout);
        const searchTerm = e.target.value;
        searchTimeout = setTimeout(() => {
            fetchFiles(searchTerm);
        }, 300);
    });

    async function handleConfirmDelete() {
        const authToken = deleteAuthTokenInput.value.trim();
        if (!authToken) {
            showToast("请输入授权令牌。", 'error');
            return;
        }

        try {
            const response = await fetch(`/api/delete/${fileToDelete.id}`, {
                method: 'DELETE',
                headers: { 'Auth-Token': authToken }
            });

            const result = await response.json();
            if (!response.ok || !result.ok) throw new Error(result.message || '删除失败');
            
            showToast('文件删除成功。');
            hideModal(deleteModal);
            fetchFiles(searchInput.value);
        } catch (error) {
            console.error('删除错误:', error);
            showToast(`删除失败: ${error.message}`, 'error');
        }
    }

    // --- Global Functions ---
    window.downloadFile = function(fileId) {
        window.location.href = `/api/download/${fileId}`;
    }

    window.deleteFile = function(fileId, filename) {
        showDeleteModal(fileId, filename);
    }

    // Share Modal Elements
    const shareModal = document.getElementById('shareModal');
    const closeShareModalBtn = document.getElementById('closeShareModalBtn');
    const shareFilenameSpan = document.getElementById('shareFilename');
    const sharePasswordInput = document.getElementById('sharePassword');
    const generateShareLinkBtn = document.getElementById('generateShareLinkBtn');
    const shareResultDiv = document.getElementById('shareResult');
    const shareLinkInput = document.getElementById('shareLink');
    const copyShareLinkBtn = document.getElementById('copyShareLinkBtn');
    const directDownloadLinkInput = document.getElementById('directDownloadLink');
    const copyDirectDownloadLinkBtn = document.getElementById('copyDirectDownloadLinkBtn');
    let fileToShare = { id: null, filename: null, share_token: null };

    // --- Share Modal ---
    async function showShareModal(fileId, filename) {
        fileToShare = { id: fileId, filename: filename, share_token: null };
        shareFilenameSpan.textContent = filename;
        sharePasswordInput.value = '';
        shareResultDiv.classList.add('hidden');
        shareLinkInput.value = '';
        directDownloadLinkInput.value = '';
        showModal(shareModal);

        try {
            const response = await fetch(`/api/file/share-details?id=${fileId}`);
            if (response.ok) {
                const details = await response.json();
                if (details.share_token) {
                    fileToShare.share_token = details.share_token;
                    sharePasswordInput.value = details.share_password;
                    updateShareLinks(details.share_token, details.share_password);
                    showToast('已加载已有的分享链接');
                }
            }
        } catch (error) {
            console.error('获取分享详情时出错:', error);
        }
    }

    function updateShareLinks(shareToken, password) {
        const host = window.appConfig && window.appConfig.host ? window.appConfig.host : window.location.origin;
        const shareLink = `${host}/share.html?file=${shareToken}`;
        shareLinkInput.value = shareLink;

        let directLink = `${host}/api/share/download?file=${shareToken}`;
        if (password) {
            directLink += `&password=${encodeURIComponent(password)}`;
        }
        directDownloadLinkInput.value = directLink;

        shareResultDiv.classList.remove('hidden');
    }

    closeShareModalBtn.addEventListener('click', () => hideModal(shareModal));
    shareModal.addEventListener('click', (e) => {
        if (e.target === shareModal) hideModal(shareModal);
    });

    generateShareLinkBtn.addEventListener('click', async () => {
        const password = sharePasswordInput.value;
        try {
            const response = await fetch('/api/share', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ file_id: fileToShare.id, password: password })
            });
            const result = await response.json();
            if (!response.ok) throw new Error(result.message || '生成链接失败');

            fileToShare.share_token = result.share_token;
            updateShareLinks(result.share_token, password);
            showToast('分享链接已生成/更新！');
        } catch (error) {
            showToast(`生成链接失败: ${error.message}`, 'error');
        }
    });

    copyShareLinkBtn.addEventListener('click', () => {
        if (shareLinkInput.value) {
            navigator.clipboard.writeText(shareLinkInput.value)
                .then(() => showToast('分享页面链接已复制！'))
                .catch(() => showToast('复制失败', 'error'));
        }
    });

    copyDirectDownloadLinkBtn.addEventListener('click', () => {
        if (directDownloadLinkInput.value) {
            navigator.clipboard.writeText(directDownloadLinkInput.value)
                .then(() => showToast('直接下载链接已复制！'))
                .catch(() => showToast('复制失败', 'error'));
        }
    });

    window.shareFile = function(fileId, filename) {
        showShareModal(fileId, filename);
    }

    // --- Initial Load ---
    uploadButton.addEventListener('click', uploadFile);
    fetchConfig().then(() => fetchFiles());
});