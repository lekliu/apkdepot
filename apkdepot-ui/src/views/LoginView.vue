<template>
  <div class="container login-container">
    <h1>Admin Login</h1>
    <p>Please enter your credentials to manage APKs.</p>

    <form @submit.prevent="handleLogin" class="login-form">
      <div class="form-group">
        <label for="username">Username</label>
        <input
          type="text"
          id="username"
          v-model="username"
          required
          placeholder="Enter username"
        >
      </div>

      <div class="form-group">
        <label for="password">Password</label>
        <input
          type="password"
          id="password"
          v-model="password"
          required
          placeholder="Enter password"
        >
      </div>

      <button type="submit" class="button" :disabled="isLoading">
        {{ isLoading ? 'Logging in...' : 'Login' }}
      </button>

      <p v-if="errorMessage" class="error-message">{{ errorMessage }}</p>
    </form>

    <p class="back-link">
      <router-link to="/">← Back to APK list</router-link>
    </p>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

// --- State ---
const username = ref('');
const password = ref('');
const isLoading = ref(false);
const errorMessage = ref('');

// --- Composables ---
const authStore = useAuthStore();
const router = useRouter();

// --- Methods ---
const handleLogin = async () => {
  // Reset state before new attempt
  isLoading.value = true;
  errorMessage.value = '';

  try {
    // Call the login action from the auth store
    await authStore.login({
      username: username.value,
      password: password.value,
    });

    // On successful login, redirect to the home page
    router.push('/');
  } catch (error) {
    // Display an error message if login fails
    errorMessage.value = error.response?.data?.error || 'Invalid username or password.';
    console.error("Login failed:", error);
  } finally {
    // Always stop the loading indicator
    isLoading.value = false;
  }
};
</script>

<style scoped>
/*
  Scoped styles only apply to this component.
  General styles like .container, .button, .form-group are in main.css.
*/
.login-container {
  max-width: 400px;
  margin-left: auto;
  margin-right: auto;
  text-align: center;
}

.login-form {
  margin-top: 2rem;
  text-align: left;
}

.back-link {
  margin-top: 2rem;
}

.back-link a {
  text-decoration: none;
  color: #007bff;
}

.back-link a:hover {
  text-decoration: underline;
}
</style>
