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
        <!-- Column 7 (Conditional): Rollout Strategy -->
        <th v-if="authStore.isLoggedIn">Rollout Strategy</th>
        <!-- Column 8: Action (Download) -->
        <th>Action</th>
        <!-- Column 9 (Conditional): Admin (Delete) -->
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

        <!-- Column 7: Release Strategy (Only for Latest) -->
        <td v-if="authStore.isLoggedIn">
          <div v-if="isLatest(apk)" class="strategy-cell">
            <div class="strategy-info">
              <span class="badge" :class="getRolloutClass(apk.rolloutRate)">
                {{ (apk.rolloutRate / 100).toFixed(0) }}%
              </span>
              <small v-if="apk.minForceVersionCode > 0">Force &lt; {{ apk.minForceVersionCode }}</small>
            </div>
            <button @click="openStrategyModal(apk)" class="button button-small button-outline">Edit</button>
          </div>
          <div v-else class="strategy-cell disabled">
            <span class="text-muted">Old Version</span>
          </div>
        </td>

        <!-- Column 8: Action (Download) -->
        <td>
          <a :href="getDownloadUrl(apk.fileName)" class="button">Download</a>
        </td>
        <!-- Column 9: Admin (Delete) -->
        <td v-if="authStore.isLoggedIn">
          <button @click="deleteApk(apk.fileName)" class="button button-danger">Delete</button>
        </td>
      </tr>
      </tbody>
    </table>

    <p v-else>No APKs have been uploaded yet.</p>

    <!-- Strategy Modal -->
    <div v-if="showStrategyModal" class="modal-overlay">
      <div class="modal-content">
        <h3>Release Strategy: {{ editingApk.appName }}</h3>
        <p class="modal-subtitle">{{ editingApk.packageName }}</p>

        <div class="form-group">
          <label>Rollout Rate (灰度比例):</label>
          <div class="rollout-control">
            <input
              type="range"
              v-model.number="editingForm.rolloutRate"
              min="0"
              max="10000"
              step="100"
              class="slider"
            >
            <div class="input-wrapper">
              <input
                type="number"
                v-model="rolloutPercentage"
                class="input-control percentage-input"
                min="0"
                max="100"
                step="0.01"
              >
              <span class="unit">%</span>
            </div>
          </div>
          <small class="help-text">0% = 暂停发布, 100% = 全量发布。支持 0.01% 精度。</small>
        </div>

        <div class="form-group">
          <label>Min Force Version Code (强制基线):</label>
          <input
            type="number"
            v-model.number="editingForm.minForceVersionCode"
            class="input-control"
            placeholder="Enter version code (e.g., 100)"
          >
          <small class="help-text">
            当前最新 VersionCode 为 <strong>{{ editingApk?.versionCode }}</strong>。<br>
            低于此数值的用户将收到强制更新弹窗。
          </small>
        </div>

        <div class="modal-actions">
          <button @click="closeStrategyModal" class="button button-secondary">Cancel</button>
          <button @click="saveStrategy" class="button" :disabled="isSaving">
            {{ isSaving ? 'Saving...' : 'Save Changes' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue';
import axios from 'axios';
import { useAuthStore } from '@/stores/auth';

// --- State ---
const apks = ref([]);
const isLoading = ref(true);
const showStrategyModal = ref(false);
const isSaving = ref(false);
const editingApk = ref(null);

// 表单数据
const editingForm = reactive({
  packageName: '',
  rolloutRate: 0,
  minForceVersionCode: 0
});

const authStore = useAuthStore();

// --- Computed Properties ---

// 1. 计算百分比显示 (0-10000 <-> 0-100.00)
const rolloutPercentage = computed({
  get: () => {
    return (editingForm.rolloutRate / 100).toFixed(2);
  },
  set: (val) => {
    editingForm.rolloutRate = Math.round(val * 100);
  }
});

// 2. 计算每个包名的最新 VersionCode
const latestVersionMap = computed(() => {
  const map = {};
  apks.value.forEach(apk => {
    // 假设 apks 列表不一定有序，找出最大值
    if (!map[apk.packageName] || apk.versionCode > map[apk.packageName]) {
      map[apk.packageName] = apk.versionCode;
    }
  });
  return map;
});

// 3. 判断是否为最新版
const isLatest = (apk) => {
  return apk.versionCode === latestVersionMap.value[apk.packageName];
};

// --- Methods ---

const fetchApks = async () => {
  isLoading.value = true;
  try {
    const response = await axios.get('/api/apks');
    apks.value = response.data || [];
  } catch (error) {
    console.error('Failed to fetch APKs:', error);
  } finally {
    isLoading.value = false;
  }
};

const getDownloadUrl = (fileName) => {
  const baseUrl = import.meta.env.DEV ? 'http://localhost:8080' : '';
  return `${baseUrl}/apks/${fileName}`;
};

const deleteApk = async (fileName) => {
  if (confirm(`Are you sure you want to delete ${fileName}?`)) {
    try {
      await axios.delete(`/api/apks/${fileName}`);
      await fetchApks();
    } catch (error) {
      console.error('Delete failed:', error);
      alert('Delete failed.');
    }
  }
};

// --- Strategy Modal ---

const openStrategyModal = (apk) => {
  editingApk.value = apk;
  editingForm.packageName = apk.packageName;
  editingForm.rolloutRate = apk.rolloutRate || 0;
  editingForm.minForceVersionCode = apk.minForceVersionCode || 0;
  showStrategyModal.value = true;
};

const closeStrategyModal = () => {
  showStrategyModal.value = false;
  editingApk.value = null;
};

const saveStrategy = async () => {
  isSaving.value = true;
  try {
    await axios.post('/api/config/update', editingForm);
    await fetchApks(); // Refresh list to reflect changes
    closeStrategyModal();
  } catch (error) {
    alert('Failed to update strategy: ' + (error.response?.data?.error || error.message));
  } finally {
    isSaving.value = false;
  }
};

const getRolloutClass = (rate) => {
  if (rate === 0) return 'badge-gray';
  if (rate === 10000) return 'badge-green';
  return 'badge-blue';
};

// --- Lifecycle ---
onMounted(() => {
  fetchApks();
});
</script>

<style scoped>
/* --- Button Styles --- */
.button-small {
  padding: 4px 8px;
  font-size: 0.8rem;
}
.button-outline {
  background: transparent;
  border: 1px solid #007bff;
  color: #007bff;
}
.button-outline:hover {
  background: #007bff;
  color: white;
}
.button-secondary {
  background-color: #6c757d;
  margin-right: 10px;
}
.button-secondary:hover {
  background-color: #5a6268;
}
.button-danger {
  background-color: #dc3545;
}

/* --- Badge / Strategy Cell Styles --- */
.strategy-cell {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.strategy-info {
  display: flex;
  flex-direction: column;
}
.badge {
  display: inline-block;
  padding: 0.25em 0.4em;
  font-size: 75%;
  font-weight: 700;
  line-height: 1;
  text-align: center;
  white-space: nowrap;
  vertical-align: baseline;
  border-radius: 0.25rem;
  color: #fff;
  width: fit-content;
}
.badge-gray { background-color: #6c757d; }
.badge-green { background-color: #28a745; }
.badge-blue { background-color: #17a2b8; }

.text-muted {
  color: #999;
  font-size: 0.85rem;
  font-style: italic;
}
.disabled {
  opacity: 0.6;
  pointer-events: none;
}

/* --- Modal Styles --- */
.modal-overlay {
  position: fixed;
  top: 0; left: 0; width: 100%; height: 100%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-content {
  background: white;
  padding: 2rem;
  border-radius: 8px;
  width: 400px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.2);
}
.modal-subtitle {
  color: #666;
  font-size: 0.9rem;
  margin-top: -10px;
  margin-bottom: 20px;
  border-bottom: 1px solid #eee;
  padding-bottom: 10px;
}
.form-group {
  margin-bottom: 1.5rem;
}
.input-control {
  width: 100%;
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
}
.help-text {
  display: block;
  margin-top: 5px;
  font-size: 0.8rem;
  color: #666;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}
.loading-spinner {
  text-align: center;
  padding: 2rem;
  font-size: 1.2rem;
  color: #666;
}

/* --- Rollout Control Specific --- */
.rollout-control {
  display: flex;
  align-items: center;
  gap: 15px;
}
.slider {
  flex-grow: 1;
}
.input-wrapper {
  position: relative;
  width: 100px;
  flex-shrink: 0;
}
.percentage-input {
  width: 100%;
  padding-right: 20px;
  text-align: right;
}
.unit {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  color: #666;
  font-size: 0.9rem;
}
</style>
