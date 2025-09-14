document.addEventListener('DOMContentLoaded', () => {
    const fileListBody = document.querySelector('#fileList tbody');
    const uploadButton = document.getElementById('uploadButton');
    const fileInput = document.getElementById('fileInput');
    const authTokenInput = document.getElementById('authToken');
    const uploadStatus = document.getElementById('uploadStatus');

    // Function to fetch config and set auth token
    async function fetchConfig() {
        try {
            const response = await fetch('/api/config');
            if (!response.ok) {
                throw new Error('Failed to fetch config');
            }
            const config = await response.json();
            if (config.authToken) {
                authTokenInput.value = config.authToken;
            }
        } catch (error) {
            console.error('Error fetching config:', error);
        }
    }

    // Function to fetch and display files
    async function fetchFiles() {
        try {
            const response = await fetch('/api/files');
            if (!response.ok) {
                throw new Error('Failed to fetch files');
            }
            const files = await response.json();
            fileListBody.innerHTML = ''; // Clear existing list

            if (files) {
                files.forEach(file => {
                    const row = document.createElement('tr');
                    const uploadDate = new Date(file.upload_timestamp).toLocaleString();
                    const fileSize = (file.filesize / 1024 / 1024).toFixed(2) + ' MB';

                    row.innerHTML = `
                        <td>${file.filename}</td>
                        <td>${fileSize}</td>
                        <td>${uploadDate}</td>
                        <td class="actions">
                            <button onclick="downloadFile(${file.id})">Download</button>
                            <button onclick="deleteFile(${file.id})">Delete</button>
                        </td>
                    `;
                    fileListBody.appendChild(row);
                });
            }
        } catch (error) {
            console.error('Error fetching files:', error);
            fileListBody.innerHTML = '<tr><td colspan="4">Error loading files.</td></tr>';
        }
    }

    // Function to handle file upload
    async function uploadFile() {
        const file = fileInput.files[0];
        const authToken = authTokenInput.value.trim();

        if (!file || !authToken) {
            uploadStatus.textContent = 'Please select a file and provide an Auth-Token.';
            return;
        }

        uploadStatus.textContent = 'Uploading...';
        uploadButton.disabled = true;

        const formData = new FormData();
        formData.append('image', file);

        try {
            const response = await fetch('/api/upload', {
                method: 'POST',
                headers: {
                    'Auth-Token': authToken
                },
                body: formData
            });

            const result = await response.json();
            if (response.ok && result.ok) {
                uploadStatus.textContent = 'Upload successful!';
                fileInput.value = ''; // Clear file input
                fetchFiles(); // Refresh the file list
            } else {
                throw new Error(result.message || 'Upload failed');
            }
        } catch (error) {
            console.error('Upload error:', error);
            uploadStatus.textContent = `Upload failed: ${error.message}`;
        } finally {
            uploadButton.disabled = false;
        }
    }

    // Function to download a file
    window.downloadFile = function(fileId) {
        window.location.href = `/api/download/${fileId}`;
    }

    // Function to delete a file
    window.deleteFile = async function(fileId) {
        const authToken = prompt("Please enter the Auth-Token to delete this file:", authTokenInput.value);
        if (!authToken) {
            return; // User cancelled
        }

        try {
            const response = await fetch(`/api/delete/${fileId}`, {
                method: 'DELETE',
                headers: {
                    'Auth-Token': authToken
                }
            });

            const result = await response.json();
            if (response.ok && result.ok) {
                alert('File deleted successfully.');
                fetchFiles(); // Refresh list
            } else {
                throw new Error(result.message || 'Deletion failed');
            }
        } catch (error) {
            console.error('Delete error:', error);
            alert(`Deletion failed: ${error.message}`);
        }
    }

    // Initial load
    uploadButton.addEventListener('click', uploadFile);
    fetchConfig().then(fetchFiles);
});