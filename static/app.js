document.addEventListener('DOMContentLoaded', () => {
    // --- DOM Elements ---
    const fileListBody = document.querySelector('#fileList tbody');
    const searchInput = document.getElementById('searchInput');
    const toastContainer = document.getElementById('toastContainer');
    
    // Upload Modal Elements
    const uploadModal = document.getElementById('uploadModal');
    const showUploadModalBtn = document.getElementById('showUploadModalBtn');
    const closeUploadModalBtn = document.getElementById('closeUploadModalBtn');
    const fileInput = document.getElementById('fileInput');
    const uploadButton = document.getElementById('uploadButton');
    const authTokenInput = document.getElementById('authToken');
    const uploadStatus = document.getElementById('uploadStatus');

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

    // --- Upload Modal ---
    showUploadModalBtn.addEventListener('click', () => showModal(uploadModal));
    closeUploadModalBtn.addEventListener('click', () => hideModal(uploadModal));
    uploadModal.addEventListener('click', (e) => {
        if (e.target === uploadModal) hideModal(uploadModal);
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
                        <button onclick="downloadFile(${file.id})">下载</button>
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
            uploadStatus.textContent = '请选择一个文件并提供授权令牌。';
            return;
        }

        uploadStatus.textContent = '上传中...';
        uploadButton.disabled = true;

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

            uploadStatus.textContent = '上传成功！';
            fileInput.value = '';
            setTimeout(() => {
                hideModal(uploadModal);
                uploadStatus.textContent = '';
                fetchFiles();
                showToast('文件上传成功！');
            }, 1000);

        } catch (error) {
            console.error('上传错误:', error);
            uploadStatus.textContent = `上传失败: ${error.message}`;
            showToast(`上传失败: ${error.message}`, 'error');
        } finally {
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

    // --- Initial Load ---
    uploadButton.addEventListener('click', uploadFile);
    fetchConfig().then(() => fetchFiles());
});