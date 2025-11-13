<template>
  <div class="container">
    <h1>Upload or Update APK</h1>
    <p>Select a .apk file to upload.</p>

    <form @submit.prevent="handleUpload">
      <p><input type="file" @change="onFileSelected" accept=".apk" required></p>
      <p><button type="submit" class="button" :disabled="isUploading">
        {{ isUploading ? 'Uploading...' : 'Upload' }}
      </button></p>
    </form>

    <div v-if="isUploading" class="progress-container">
      <div class="progress-bar" :style="{ width: progress + '%' }">{{ progress }}%</div>
    </div>
    <p v-if="statusMessage" :class="{ 'error-message': hasError }">{{ statusMessage }}</p>

    <router-link to="/">Back to list</router-link>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import axios from 'axios';
import { useRouter } from 'vue-router';

const selectedFile = ref(null);
const isUploading = ref(false);
const progress = ref(0);
const statusMessage = ref('');
const hasError = ref(false);
const router = useRouter();

const onFileSelected = (event) => {
  selectedFile.value = event.target.files[0];
};

const handleUpload = async () => {
  if (!selectedFile.value) return;

  const formData = new FormData();
  formData.append('apkfile', selectedFile.value);

  isUploading.value = true;
  statusMessage.value = '';
  hasError.value = false;
  progress.value = 0;

  try {
    await axios.post('/api/upload', formData, {
      onUploadProgress: (progressEvent) => {
        progress.value = Math.round((progressEvent.loaded * 100) / progressEvent.total);
      },
    });
    statusMessage.value = 'Upload successful! Redirecting...';
    setTimeout(() => router.push('/'), 1500); // 延迟跳转，让用户看到成功信息
  } catch (error) {
    hasError.value = true;
    statusMessage.value = error.response?.data?.error || 'Upload failed.';
  } finally {
    isUploading.value = false;
  }
};
</script>

<style scoped>
.error-message {
  color: red;
}
/* ... 复制之前的进度条样式 ... */
</style>
