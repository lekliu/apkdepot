<template>
  <div class="container">
    <h1>ApkDepot</h1>
    <p>Available applications for download. Sorted by upload time (newest first).</p>

    <!-- The 'Upload New APK' link is now in the global navigation bar in App.vue -->

    <div v-if="isLoading" class="loading-spinner">
      Loading APKs...
    </div>

    <table v-else-if="apks.length > 0" class="apk-table">
      <thead>
      <tr>
        <!-- Column 1 & 2: Icon and App Name -->
        <th colspan="2">Application</th>
        <!-- Column 3: Version -->
        <th>Version</th>
        <!-- Column 4: Package Name -->
        <th>Package Name</th>
        <!-- Column 5: Size -->
        <th>Size</th>
        <!-- Column 6: Upload Time -->
        <th>Upload Time</th>
        <!-- Column 7: Action (Download) -->
        <th>Action</th>
        <!-- Column 8 (Conditional): Admin (Delete) -->
        <th v-if="authStore.isLoggedIn">Admin</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="apk in apks" :key="apk.fileName">
        <!-- Column 1: Icon -->
        <td>
          <img v-if="apk.iconBase64" :src="`data:image/png;base64,${apk.iconBase64}`" alt="icon" class="apk-icon">
          <div v-else class="apk-icon" style="background-color: #ccc;"></div>
        </td>
        <!-- Column 2: App Name -->
        <td>
          <strong>{{ apk.appName }}</strong>
        </td>
        <!-- Column 3: Version -->
        <td>
          {{ apk.versionName }} ({{ apk.versionCode }})
        </td>
        <!-- Column 4: Package Name -->
        <td>
          <small>{{ apk.packageName }}</small>
        </td>
        <!-- Column 5: Size -->
        <td>
          {{ apk.fileSize }}
        </td>
        <!-- Column 6: Upload Time -->
        <td>
          {{ apk.uploadTime }}
        </td>
        <!-- Column 7: Action (Download) -->
        <td>
          <a :href="getDownloadUrl(apk.fileName)" class="button">Download</a>
        </td>
        <!-- Column 8 (Conditional): Admin (Delete) -->
        <td v-if="authStore.isLoggedIn">
          <button @click="deleteApk(apk.fileName)" class="button button-danger">Delete</button>
        </td>
      </tr>
      </tbody>
    </table>

    <p v-else>No APKs have been uploaded yet.</p>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import axios from 'axios';
import { useAuthStore } from '@/stores/auth';

// --- State ---
const apks = ref([]);
const isLoading = ref(true); // Added for better UX

// --- Composables ---
const authStore = useAuthStore();

// --- Methods ---
const fetchApks = async () => {
  isLoading.value = true;
  try {
    const response = await axios.get('/api/apks');
    apks.value = response.data || [];
  } catch (error) {
    console.error('Failed to fetch APKs:', error);
    // Optionally, show an error message to the user
  } finally {
    isLoading.value = false;
  }
};

const getDownloadUrl = (fileName) => {
  // During development, point directly to the Go server.
  // In production, Nginx will handle proxying, so a relative URL works.
  const baseUrl = import.meta.env.DEV ? 'http://localhost:8080' : '';
  return `${baseUrl}/apks/${fileName}`;
};

const deleteApk = async (fileName) => {
  if (confirm(`Are you sure you want to delete ${fileName}?`)) {
    try {
      await axios.delete(`/api/apks/${fileName}`);
      // Refresh the list after successful deletion
      await fetchApks();
    } catch (error) {
      if (error.response?.status === 401) {
        alert('Your session has expired. Please log in again.');
        authStore.logout();
      } else {
        alert('Failed to delete the APK. See console for details.');
      }
      console.error('Delete failed:', error);
    }
  }
};

// --- Lifecycle Hooks ---
onMounted(() => {
  fetchApks();
});
</script>

<style scoped>
/* Scoped styles are specific to this component */
.button-danger {
  background-color: #dc3545;
}

.button-danger:hover {
  background-color: #c82333;
}

.loading-spinner {
  text-align: center;
  padding: 2rem;
  font-size: 1.2rem;
  color: #666;
}
</style>
